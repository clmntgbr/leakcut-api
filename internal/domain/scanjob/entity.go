package scanjob

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusPending          = "pending"
	StatusExtractingFrames = "extracting_frames"
	StatusFramesReady      = "frames_ready"
	StatusFailed           = "failed"
)

const (
	DefaultAnalysisFPS        = 3.0
	DefaultDiffThreshold      = 0.08
	DefaultMaxIntervalSeconds = 10
)

type ScanJob struct {
	ID                 uuid.UUID
	VideoID            uuid.UUID
	Status             string
	FailureReason      string
	AnalysisFPS        float64
	DiffThreshold      float64
	MaxIntervalSeconds int
	CreatedAt          time.Time
	CompletedAt        *time.Time
}

func NewScanJob(videoID uuid.UUID) *ScanJob {
	return &ScanJob{
		ID:                 uuid.New(),
		VideoID:            videoID,
		Status:             StatusPending,
		AnalysisFPS:        DefaultAnalysisFPS,
		DiffThreshold:      DefaultDiffThreshold,
		MaxIntervalSeconds: DefaultMaxIntervalSeconds,
		CreatedAt:          time.Now().UTC(),
	}
}

func (j *ScanJob) MarkExtracting() {
	j.Status = StatusExtractingFrames
	j.FailureReason = ""
}

func (j *ScanJob) MarkFramesReady() {
	now := time.Now().UTC()
	j.Status = StatusFramesReady
	j.FailureReason = ""
	j.CompletedAt = &now
}

func (j *ScanJob) MarkFailed(reason string) {
	now := time.Now().UTC()
	j.Status = StatusFailed
	j.FailureReason = reason
	j.CompletedAt = &now
}
