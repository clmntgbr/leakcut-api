package job

import (
	"time"

	"github.com/google/uuid"
)

const (
	TypeExtractFrames = "extract_frames"
	TypeOCR           = "ocr"
)

const (
	StatusPending          = "pending"
	StatusExtractingFrames = "extracting_frames"
	StatusFramesReady      = "frames_ready"
	StatusOCRProcessing    = "ocr_processing"
	StatusOCRReady         = "ocr_ready"
	StatusOCRFailed        = "ocr_failed"
	StatusFailed           = "failed"
)

const (
	DefaultAnalysisFPS        = 3.0
	DefaultDiffThreshold      = 0.08
	DefaultMaxIntervalSeconds = 10
)

type Job struct {
	ID                 uuid.UUID
	VideoID            uuid.UUID
	Type               string
	Status             string
	FailureReason      string
	AnalysisFPS        float64
	DiffThreshold      float64
	MaxIntervalSeconds int
	ExpectedFrameCount int
	OCRCompletedCount  int
	CreatedAt          time.Time
	CompletedAt        *time.Time
}

func NewExtractFramesJob(videoID uuid.UUID) *Job {
	return &Job{
		ID:                 uuid.New(),
		VideoID:            videoID,
		Type:               TypeExtractFrames,
		Status:             StatusPending,
		AnalysisFPS:        DefaultAnalysisFPS,
		DiffThreshold:      DefaultDiffThreshold,
		MaxIntervalSeconds: DefaultMaxIntervalSeconds,
		CreatedAt:          time.Now().UTC(),
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

func (j *Job) MarkExtracting() {
	j.Status = StatusExtractingFrames
	j.FailureReason = ""
	j.CompletedAt = nil
}

func (j *Job) MarkFramesReady(frameCount int) {
	now := time.Now().UTC()
	j.Status = StatusFramesReady
	j.FailureReason = ""
	j.ExpectedFrameCount = frameCount
	j.CompletedAt = &now
}

func (j *Job) MarkOCRProcessing(expected, alreadyCompleted int) {
	j.Status = StatusOCRProcessing
	j.FailureReason = ""
	j.ExpectedFrameCount = expected
	j.OCRCompletedCount = alreadyCompleted
	j.CompletedAt = nil
}

func (j *Job) AddOCRCompleted(n int) {
	if n <= 0 {
		return
	}
	j.OCRCompletedCount += n
}

func (j *Job) MarkOCRReady() {
	now := time.Now().UTC()
	j.Status = StatusOCRReady
	j.FailureReason = ""
	j.CompletedAt = &now
}

func (j *Job) MarkOCRFailed(reason string) {
	now := time.Now().UTC()
	j.Status = StatusOCRFailed
	j.FailureReason = reason
	j.CompletedAt = &now
}

func (j *Job) MarkFailed(reason string) {
	now := time.Now().UTC()
	j.Status = StatusFailed
	j.FailureReason = reason
	j.CompletedAt = &now
}
