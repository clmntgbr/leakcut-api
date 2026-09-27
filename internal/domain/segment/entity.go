package segment

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusSuccess    = "success"
	StatusFailed     = "failed"
)

const (
	DefaultSegmentDurationSeconds         = 120
	DefaultMinVideoDurationForSplitSeconds = 90
	DefaultMaxSegments                    = 40
)

type Segment struct {
	ID            uuid.UUID
	VideoID       uuid.UUID
	JobID         uuid.UUID
	SegmentIndex  int
	OffsetMs      int64
	DurationMs    int64
	StorageKey    string
	Status        string
	FailureReason string
	CreatedAt     time.Time
	CompletedAt   *time.Time
}

func New(
	videoID, jobID uuid.UUID,
	segmentIndex int,
	offsetMs, durationMs int64,
	storageKey string,
) *Segment {
	return &Segment{
		ID:           uuid.New(),
		VideoID:      videoID,
		JobID:        jobID,
		SegmentIndex: segmentIndex,
		OffsetMs:     offsetMs,
		DurationMs:   durationMs,
		StorageKey:   storageKey,
		Status:       StatusPending,
		CreatedAt:    time.Now().UTC(),
	}
}

func (s *Segment) MarkProcessing() {
	s.Status = StatusProcessing
	s.FailureReason = ""
	s.CompletedAt = nil
}

func (s *Segment) MarkSuccess() {
	now := time.Now().UTC()
	s.Status = StatusSuccess
	s.FailureReason = ""
	s.CompletedAt = &now
}

func (s *Segment) MarkFailed(reason string) {
	now := time.Now().UTC()
	s.Status = StatusFailed
	s.FailureReason = reason
	s.CompletedAt = &now
}

func (s *Segment) IsDone() bool {
	return s.Status == StatusSuccess || s.Status == StatusFailed
}
