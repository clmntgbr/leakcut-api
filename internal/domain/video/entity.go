package video

import (
	"path/filepath"
	"time"

	"go-api/internal/domain/event"

	"github.com/google/uuid"
)

type Video struct {
	ID               uuid.UUID
	UserID           *uuid.UUID
	OriginalFilename string
	StorageKey       string
	ThumbnailKey     string
	SizeBytes        int64
	ContentType      string
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	FinishedAt       *time.Time

	events []event.DomainEvent
}

func NewPresignedVideo(userID uuid.UUID, filename, contentType string, sizeBytes int64) (*Video, error) {
	filename = SanitizeFilename(filename)
	if filename == "" || filename == "." || filename == string(filepath.Separator) {
		return nil, ErrInvalidFilename
	}
	if !IsVideoFilename(filename) && !IsVideoContentType(contentType) {
		return nil, ErrUnsupportedType
	}

	now := time.Now().UTC()
	id := uuid.New()
	ownerID := userID
	v := &Video{
		ID:               id,
		UserID:           &ownerID,
		OriginalFilename: filename,
		StorageKey:       NewStorageKey(id),
		SizeBytes:        sizeBytes,
		ContentType:      ContentTypeFromFilename(filename, contentType),
		Status:           StatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	v.recordCreated()
	return v, nil
}

func NewWebhookVideo(filename, contentType, remoteURL string, sizeBytes int64) (*Video, error) {
	filename = SanitizeFilename(filename)
	if filename == "" || filename == "." {
		filename = OriginalObjectName
	}
	if !IsVideoFilename(filename) && !IsVideoContentType(contentType) {
		return nil, ErrUnsupportedType
	}

	now := time.Now().UTC()
	id := uuid.New()
	v := &Video{
		ID:               id,
		OriginalFilename: filename,
		StorageKey:       NewStorageKey(id),
		SizeBytes:        sizeBytes,
		ContentType:      ContentTypeFromFilename(filename, contentType),
		Status:           StatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	v.recordCreated()
	v.recordEvent(VideoIngestRequested{
		ID:         uuid.New().String(),
		VideoID:    v.ID.String(),
		RemoteURL:  remoteURL,
		StorageKey: v.StorageKey,
		Timestamp:  now,
	})
	return v, nil
}

func (v *Video) PullEvents() []event.DomainEvent {
	events := v.events
	v.events = nil
	return events
}

func (v *Video) recordEvent(e event.DomainEvent) {
	v.events = append(v.events, e)
}

func (v *Video) recordCreated() {
	v.recordEvent(VideoCreated{
		ID:         uuid.New().String(),
		VideoID:    v.ID.String(),
		UserID:     v.ownerUserID(),
		Filename:   v.OriginalFilename,
		StorageKey: v.StorageKey,
		Status:     v.Status,
		Timestamp:  v.CreatedAt,
	})
}

func (v *Video) ownerUserID() string {
	if v.UserID == nil {
		return ""
	}
	return v.UserID.String()
}

func (v *Video) MarkUploaded(contentType string, sizeBytes int64) error {
	if contentType != "" {
		v.ContentType = contentType
	}
	if sizeBytes > 0 {
		v.SizeBytes = sizeBytes
	}

	switch v.Status {
	case StatusPending:
		now := time.Now().UTC()
		v.Status = StatusProcessing
		v.UpdatedAt = now
		v.recordEvent(VideoUploaded{
			ID:          uuid.New().String(),
			VideoID:     v.ID.String(),
			UserID:      v.ownerUserID(),
			StorageKey:  v.StorageKey,
			ContentType: v.ContentType,
			SizeBytes:   v.SizeBytes,
			Status:      v.Status,
			Timestamp:   now,
		})
		return nil
	case StatusProcessing, StatusSuccess:
		v.UpdatedAt = time.Now().UTC()
		return nil
	default:
		return ErrInvalidTransition
	}
}

func (v *Video) MarkExtractionQueued() error {
	switch v.Status {
	case StatusPending:
		v.Status = StatusProcessing
		v.UpdatedAt = time.Now().UTC()
		return nil
	case StatusProcessing, StatusSuccess, StatusFailed:
		return nil
	default:
		return ErrInvalidTransition
	}
}

func (v *Video) MarkExtracting(jobID uuid.UUID, jobType, jobStatus string) error {
	switch v.Status {
	case StatusPending, StatusProcessing, StatusFailed:
		now := time.Now().UTC()
		v.Status = StatusProcessing
		v.UpdatedAt = now
		v.recordEvent(VideoExtracting{
			ID:        uuid.New().String(),
			VideoID:   v.ID.String(),
			UserID:    v.ownerUserID(),
			JobID:     jobID.String(),
			JobType:   jobType,
			Status:    v.Status,
			JobStatus: jobStatus,
			Timestamp: now,
		})
		return nil
	case StatusSuccess:
		return nil
	default:
		return ErrInvalidTransition
	}
}

func (v *Video) RecordSegmentReady(
	jobID, segmentID uuid.UUID,
	segmentIndex int,
	offsetMs, durationMs int64,
	storageKey string,
) {
	v.recordEvent(VideoSegmentReady{
		ID:           uuid.New().String(),
		VideoID:      v.ID.String(),
		UserID:       v.ownerUserID(),
		JobID:        jobID.String(),
		SegmentID:    segmentID.String(),
		SegmentIndex: segmentIndex,
		OffsetMs:     offsetMs,
		DurationMs:   durationMs,
		StorageKey:   storageKey,
		Timestamp:    time.Now().UTC(),
	})
}

func (v *Video) MarkFramesReady(jobID uuid.UUID, jobType, jobStatus string, frames []ExtractedFramePayload) error {
	switch v.Status {
	case StatusProcessing, StatusPending:
		now := time.Now().UTC()
		v.Status = StatusProcessing
		v.UpdatedAt = now
		v.recordEvent(VideoFramesExtracted{
			ID:         uuid.New().String(),
			VideoID:    v.ID.String(),
			UserID:     v.ownerUserID(),
			JobID:      jobID.String(),
			JobType:    jobType,
			Status:     v.Status,
			JobStatus:  jobStatus,
			FrameCount: len(frames),
			Frames:     frames,
			Timestamp:  now,
		})
		return nil
	case StatusSuccess:
		return nil
	default:
		return ErrInvalidTransition
	}
}

func (v *Video) MarkExtractionFailed(jobID uuid.UUID, jobType, jobStatus, reason string) error {
	switch v.Status {
	case StatusProcessing, StatusPending:
	case StatusFailed:
		return nil
	default:
		return ErrInvalidTransition
	}

	now := time.Now().UTC()
	v.Status = StatusFailed
	v.UpdatedAt = now
	v.recordEvent(VideoFrameExtractionFailed{
		ID:        uuid.New().String(),
		VideoID:   v.ID.String(),
		UserID:    v.ownerUserID(),
		JobID:     jobID.String(),
		JobType:   jobType,
		Status:    v.Status,
		JobStatus: jobStatus,
		Reason:    reason,
		Timestamp: now,
	})
	return nil
}

func (v *Video) SetThumbnailKey(key string) {
	if key == "" || v.ThumbnailKey == key {
		return
	}
	v.ThumbnailKey = key
	v.UpdatedAt = time.Now().UTC()
}

func (v *Video) MarkOCRProcessing(jobID uuid.UUID, jobType, jobStatus string, expected, completed int) error {
	switch v.Status {
	case StatusProcessing, StatusPending, StatusFailed:
		now := time.Now().UTC()
		v.Status = StatusProcessing
		v.UpdatedAt = now
		v.recordEvent(VideoOCRProcessing{
			ID:                 uuid.New().String(),
			VideoID:            v.ID.String(),
			UserID:             v.ownerUserID(),
			JobID:              jobID.String(),
			JobType:            jobType,
			Status:             v.Status,
			JobStatus:          jobStatus,
			ExpectedFrameCount: expected,
			OCRCompletedCount:  completed,
			Timestamp:          now,
		})
		return nil
	case StatusSuccess:
		return nil
	default:
		return ErrInvalidTransition
	}
}

// RequestOCRFrames enqueues one outbox message per frame for competing OCR consumers.
func (v *Video) RequestOCRFrames(jobID uuid.UUID, frames []ExtractedFramePayload) {
	if len(frames) == 0 {
		return
	}
	now := time.Now().UTC()
	userID := v.ownerUserID()
	for _, frame := range frames {
		v.recordEvent(VideoOCRFrameRequested{
			ID:          uuid.New().String(),
			VideoID:     v.ID.String(),
			UserID:      userID,
			JobID:       jobID.String(),
			FrameID:     frame.ID,
			FrameIndex:  frame.Index,
			TimestampMs: frame.TimestampMs,
			StorageKey:  frame.StorageKey,
			Timestamp:   now,
		})
	}
}

// RequestClassifyFrames enqueues one outbox message per frame for competing classify consumers.
func (v *Video) RequestClassifyFrames(jobID uuid.UUID, frameIDs []uuid.UUID) {
	if len(frameIDs) == 0 {
		return
	}
	now := time.Now().UTC()
	userID := v.ownerUserID()
	for _, frameID := range frameIDs {
		v.recordEvent(VideoClassifyFrameRequested{
			ID:        uuid.New().String(),
			VideoID:   v.ID.String(),
			UserID:    userID,
			JobID:     jobID.String(),
			FrameID:   frameID.String(),
			Timestamp: now,
		})
	}
}

func (v *Video) RecordOCRProgress(jobID uuid.UUID, jobType, jobStatus string, expected, completed int) error {
	if v.Status != StatusProcessing {
		return nil
	}

	now := time.Now().UTC()
	v.UpdatedAt = now
	v.recordEvent(VideoOCRProcessing{
		ID:                 uuid.New().String(),
		VideoID:            v.ID.String(),
		UserID:             v.ownerUserID(),
		JobID:              jobID.String(),
		JobType:            jobType,
		Status:             v.Status,
		JobStatus:          jobStatus,
		ExpectedFrameCount: expected,
		OCRCompletedCount:  completed,
		Timestamp:          now,
	})
	return nil
}

func (v *Video) MarkOCRReady(jobID uuid.UUID, jobType, jobStatus string, expected, completed int, results []OCRFrameResultPayload) error {
	switch v.Status {
	case StatusProcessing, StatusPending:
		now := time.Now().UTC()
		v.Status = StatusProcessing
		v.UpdatedAt = now
	case StatusSuccess:
		return nil
	default:
		return ErrInvalidTransition
	}

	v.recordEvent(VideoFramesOCRCompleted{
		ID:                 uuid.New().String(),
		VideoID:            v.ID.String(),
		UserID:             v.ownerUserID(),
		JobID:              jobID.String(),
		JobType:            jobType,
		Status:             v.Status,
		JobStatus:          jobStatus,
		ExpectedFrameCount: expected,
		OCRCompletedCount:  completed,
		Results:            results,
		Timestamp:          time.Now().UTC(),
	})
	return nil
}

func (v *Video) MarkOCRFailed(jobID uuid.UUID, jobType, jobStatus, reason string, expected, completed int) error {
	switch v.Status {
	case StatusProcessing, StatusPending:
	case StatusFailed:
		return nil
	default:
		return ErrInvalidTransition
	}

	now := time.Now().UTC()
	v.Status = StatusFailed
	v.UpdatedAt = now
	v.recordEvent(VideoOCRFailed{
		ID:                 uuid.New().String(),
		VideoID:            v.ID.String(),
		UserID:             v.ownerUserID(),
		JobID:              jobID.String(),
		JobType:            jobType,
		Status:             v.Status,
		JobStatus:          jobStatus,
		Reason:             reason,
		ExpectedFrameCount: expected,
		OCRCompletedCount:  completed,
		Timestamp:          now,
	})
	return nil
}

func (v *Video) MarkClassifying(jobID uuid.UUID, jobType, jobStatus string, expected int) error {
	switch v.Status {
	case StatusProcessing, StatusPending, StatusFailed:
		now := time.Now().UTC()
		v.Status = StatusProcessing
		v.UpdatedAt = now
		v.recordEvent(VideoClassifying{
			ID:                 uuid.New().String(),
			VideoID:            v.ID.String(),
			UserID:             v.ownerUserID(),
			JobID:              jobID.String(),
			JobType:            jobType,
			Status:             v.Status,
			JobStatus:          jobStatus,
			ExpectedFrameCount: expected,
			Timestamp:          now,
		})
		return nil
	case StatusSuccess:
		return nil
	default:
		return ErrInvalidTransition
	}
}

func (v *Video) MarkClassified(jobID uuid.UUID, jobType, jobStatus string, expected int, classifications []ClassificationPayload) error {
	switch v.Status {
	case StatusProcessing, StatusPending:
	case StatusSuccess:
		return nil
	default:
		return ErrInvalidTransition
	}

	now := time.Now().UTC()
	v.Status = StatusSuccess
	v.UpdatedAt = now
	v.FinishedAt = &now
	v.recordEvent(VideoFramesClassified{
		ID:                 uuid.New().String(),
		VideoID:            v.ID.String(),
		UserID:             v.ownerUserID(),
		JobID:              jobID.String(),
		JobType:            jobType,
		Status:             v.Status,
		JobStatus:          jobStatus,
		ExpectedFrameCount: expected,
		Classifications:    classifications,
		Timestamp:          now,
	})
	return nil
}

func (v *Video) MarkClassifyFailed(jobID uuid.UUID, jobType, jobStatus, reason string, expected int) error {
	switch v.Status {
	case StatusProcessing, StatusPending:
	case StatusFailed:
		return nil
	default:
		return ErrInvalidTransition
	}

	now := time.Now().UTC()
	v.Status = StatusFailed
	v.UpdatedAt = now
	v.recordEvent(VideoClassifyFailed{
		ID:                 uuid.New().String(),
		VideoID:            v.ID.String(),
		UserID:             v.ownerUserID(),
		JobID:              jobID.String(),
		JobType:            jobType,
		Status:             v.Status,
		JobStatus:          jobStatus,
		Reason:             reason,
		ExpectedFrameCount: expected,
		Timestamp:          now,
	})
	return nil
}

func (v *Video) MarkUploadExpired() error {
	if v.Status != StatusPending {
		return ErrInvalidTransition
	}
	now := time.Now().UTC()
	v.Status = StatusFailed
	v.UpdatedAt = now
	v.recordEvent(VideoUploadExpired{
		ID:        uuid.New().String(),
		VideoID:   v.ID.String(),
		Timestamp: now,
	})
	return nil
}
