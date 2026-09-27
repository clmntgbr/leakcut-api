package job

import (
	"time"

	"github.com/google/uuid"
)

const (
	TypeSegment  = "segment"
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
	DefaultAnalysisFPS                     = 2.0
	DefaultPHashDistanceThreshold          = 14
	DefaultMaxIntervalSeconds              = 15
	DefaultFrameMaxWidthPx                 = 960
	DefaultFrameUploadConcurrency          = 4
	DefaultRetentionRatioNeutral           = 0.5
	DefaultRetentionRatioEmpty             = 0.1
	DefaultRetentionContextWindowSeconds   = 4
	DefaultSegmentDurationSeconds          = 120
	DefaultMinVideoDurationForSplitSeconds = 90
	DefaultMaxSegments                     = 40
)

type Job struct {
	ID                              uuid.UUID
	VideoID                         uuid.UUID
	Type                            string
	Status                          string
	FailureReason                   string
	AnalysisFPS                     float64
	PHashDistanceThreshold          int
	MaxIntervalSeconds              int
	ExpectedFrameCount              int
	OCRCompletedCount               int
	ExpectedSegmentCount            int
	CompletedSegmentCount           int
	RetentionRatioNeutral           float64
	RetentionRatioEmpty             float64
	RetentionContextWindowSeconds   int
	SegmentDurationSeconds          int
	MinVideoDurationForSplitSeconds int
	MaxSegments                     int
	CreatedAt                       time.Time
	StartedAt                       *time.Time
	FinishedAt                      *time.Time
}

func NewSegmentJob(videoID uuid.UUID) *Job {
	return &Job{
		ID:                              uuid.New(),
		VideoID:                         videoID,
		Type:                            TypeSegment,
		Status:                          StatusPending,
		SegmentDurationSeconds:          DefaultSegmentDurationSeconds,
		MinVideoDurationForSplitSeconds: DefaultMinVideoDurationForSplitSeconds,
		MaxSegments:                     DefaultMaxSegments,
		CreatedAt:                       time.Now().UTC(),
	}
}

func NewFrameJob(videoID uuid.UUID) *Job {
	return &Job{
		ID:                            uuid.New(),
		VideoID:                       videoID,
		Type:                          TypeFrame,
		Status:                        StatusPending,
		AnalysisFPS:                   DefaultAnalysisFPS,
		PHashDistanceThreshold:        DefaultPHashDistanceThreshold,
		MaxIntervalSeconds:            DefaultMaxIntervalSeconds,
		RetentionRatioNeutral:         DefaultRetentionRatioNeutral,
		RetentionRatioEmpty:           DefaultRetentionRatioEmpty,
		RetentionContextWindowSeconds: DefaultRetentionContextWindowSeconds,
		CreatedAt:                     time.Now().UTC(),
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
	now := time.Now().UTC()
	j.Status = StatusProcessing
	j.FailureReason = ""
	j.FinishedAt = nil
	if j.StartedAt == nil {
		j.StartedAt = &now
	}
}

func (j *Job) MarkSuccess() {
	now := time.Now().UTC()
	j.Status = StatusSuccess
	j.FailureReason = ""
	if j.StartedAt == nil {
		j.StartedAt = &now
	}
	j.FinishedAt = &now
}

func (j *Job) MarkFailed(reason string) {
	now := time.Now().UTC()
	j.Status = StatusFailed
	j.FailureReason = reason
	if j.StartedAt == nil {
		j.StartedAt = &now
	}
	j.FinishedAt = &now
}

func (j *Job) SetExpectedFrameCount(n int) {
	j.ExpectedFrameCount = n
}

func (j *Job) SetSegmentProgress(expected, completed int) {
	j.ExpectedSegmentCount = expected
	j.CompletedSegmentCount = completed
}

func (j *Job) AddCompletedSegment(n int) {
	if n <= 0 {
		return
	}
	j.CompletedSegmentCount += n
}

func (j *Job) SegmentsComplete() bool {
	return j.ExpectedSegmentCount > 0 && j.CompletedSegmentCount >= j.ExpectedSegmentCount
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
