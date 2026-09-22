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
		Status:           StatusPendingUpload,
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
		Status:           StatusPendingUpload,
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
	switch v.Status {
	case StatusPendingUpload:
		now := time.Now().UTC()
		if contentType != "" {
			v.ContentType = contentType
		}
		if sizeBytes > 0 {
			v.SizeBytes = sizeBytes
		}
		v.Status = StatusExtractionQueued
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
	case StatusUploaded, StatusExtractionQueued, StatusExtracting, StatusFramesReady:
		if contentType != "" {
			v.ContentType = contentType
		}
		if sizeBytes > 0 {
			v.SizeBytes = sizeBytes
		}
		v.UpdatedAt = time.Now().UTC()
		return nil
	default:
		return ErrInvalidTransition
	}
}

func (v *Video) MarkExtractionQueued() error {
	switch v.Status {
	case StatusUploaded:
		v.Status = StatusExtractionQueued
		v.UpdatedAt = time.Now().UTC()
		return nil
	case StatusExtractionQueued, StatusExtracting, StatusFramesReady, StatusExtractionFailed:
		return nil
	default:
		return ErrInvalidTransition
	}
}

func (v *Video) MarkExtracting(jobID uuid.UUID, jobStatus string) error {
	switch v.Status {
	case StatusUploaded, StatusExtractionQueued, StatusExtractionFailed:
		now := time.Now().UTC()
		v.Status = StatusExtracting
		v.UpdatedAt = now
		v.recordEvent(VideoExtracting{
			ID:        uuid.New().String(),
			VideoID:   v.ID.String(),
			UserID:    v.ownerUserID(),
			JobID:     jobID.String(),
			Status:    v.Status,
			JobStatus: jobStatus,
			Timestamp: now,
		})
		return nil
	case StatusExtracting:
		return nil
	default:
		return ErrInvalidTransition
	}
}

func (v *Video) MarkFramesReady(jobID uuid.UUID, jobStatus string, frames []ExtractedFramePayload) error {
	if v.Status != StatusExtracting && v.Status != StatusExtractionQueued && v.Status != StatusUploaded {
		if v.Status == StatusFramesReady {
			return nil
		}
		return ErrInvalidTransition
	}

	now := time.Now().UTC()
	v.Status = StatusFramesReady
	v.UpdatedAt = now
	v.recordEvent(VideoFramesExtracted{
		ID:         uuid.New().String(),
		VideoID:    v.ID.String(),
		UserID:     v.ownerUserID(),
		JobID:      jobID.String(),
		Status:     v.Status,
		JobStatus:  jobStatus,
		FrameCount: len(frames),
		Frames:     frames,
		Timestamp:  now,
	})
	return nil
}

func (v *Video) MarkExtractionFailed(jobID uuid.UUID, jobStatus, reason string) error {
	switch v.Status {
	case StatusExtracting, StatusExtractionQueued, StatusUploaded:
	case StatusExtractionFailed:
		return nil
	default:
		return ErrInvalidTransition
	}

	now := time.Now().UTC()
	v.Status = StatusExtractionFailed
	v.UpdatedAt = now
	v.recordEvent(VideoFrameExtractionFailed{
		ID:        uuid.New().String(),
		VideoID:   v.ID.String(),
		UserID:    v.ownerUserID(),
		JobID:     jobID.String(),
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

func (v *Video) MarkUploadExpired() error {
	if v.Status != StatusPendingUpload {
		return ErrInvalidTransition
	}
	now := time.Now().UTC()
	v.Status = StatusUploadExpired
	v.UpdatedAt = now
	v.recordEvent(VideoUploadExpired{
		ID:        uuid.New().String(),
		VideoID:   v.ID.String(),
		Timestamp: now,
	})
	return nil
}
