package job

import (
	"time"

	"github.com/google/uuid"
)

const (
	TypeFrame    = "frame"
	TypeOCR      = "ocr"
	TypeClassify = "classify"
)

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusSuccess    = "success"
	StatusFailed     = "failed"
)

const (
	DefaultAnalysisFPS            = 2.0
	DefaultPHashDistanceThreshold = 14
	DefaultMaxIntervalSeconds     = 15
	DefaultFrameMaxWidthPx        = 960
	DefaultFrameUploadConcurrency = 4
)

type Job struct {
	ID                     uuid.UUID
	VideoID                uuid.UUID
	Type                   string
	Status                 string
	FailureReason          string
	AnalysisFPS            float64
	PHashDistanceThreshold int
	MaxIntervalSeconds     int
	ExpectedFrameCount     int
	OCRCompletedCount      int
	CreatedAt              time.Time
	CompletedAt            *time.Time
}

func NewFrameJob(videoID uuid.UUID) *Job {
	return &Job{
		ID:                     uuid.New(),
		VideoID:                videoID,
		Type:                   TypeFrame,
		Status:                 StatusPending,
		AnalysisFPS:            DefaultAnalysisFPS,
		PHashDistanceThreshold: DefaultPHashDistanceThreshold,
		MaxIntervalSeconds:     DefaultMaxIntervalSeconds,
		CreatedAt:              time.Now().UTC(),
	}
}

func NewOCRJob(videoID uuid.UUID) *Job {
	return &Job{
		ID:        uuid.New(),
		VideoID:   videoID,
		Type:      TypeOCR,
		Status:    StatusPending,
		CreatedAt: time.Now().UTC(),
	}
}

func NewClassifyJob(videoID uuid.UUID) *Job {
	return &Job{
		ID:        uuid.New(),
		VideoID:   videoID,
		Type:      TypeClassify,
		Status:    StatusPending,
		CreatedAt: time.Now().UTC(),
	}
}

func (j *Job) MarkProcessing() {
	j.Status = StatusProcessing
	j.FailureReason = ""
	j.CompletedAt = nil
}

func (j *Job) MarkSuccess() {
	now := time.Now().UTC()
	j.Status = StatusSuccess
	j.FailureReason = ""
	j.CompletedAt = &now
}

func (j *Job) MarkFailed(reason string) {
	now := time.Now().UTC()
	j.Status = StatusFailed
	j.FailureReason = reason
	j.CompletedAt = &now
}

func (j *Job) SetExpectedFrameCount(n int) {
	j.ExpectedFrameCount = n
}

func (j *Job) SetOCRProgress(expected, alreadyCompleted int) {
	j.ExpectedFrameCount = expected
	j.OCRCompletedCount = alreadyCompleted
}

func (j *Job) AddOCRCompleted(n int) {
	if n <= 0 {
		return
	}
	j.OCRCompletedCount += n
}
