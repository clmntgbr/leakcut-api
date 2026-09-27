package write

import (
	"time"

	domainjob "go-api/internal/domain/job"

	"github.com/google/uuid"
)

type JobModel struct {
	ID                              uuid.UUID  `gorm:"column:id;primaryKey"`
	VideoID                         uuid.UUID  `gorm:"column:video_id"`
	Type                            string     `gorm:"column:type"`
	Status                          string     `gorm:"column:status"`
	FailureReason                   string     `gorm:"column:failure_reason"`
	AnalysisFPS                     float64    `gorm:"column:analysis_fps"`
	PHashDistanceThreshold          int        `gorm:"column:phash_distance_threshold"`
	MaxIntervalSeconds              int        `gorm:"column:max_interval_seconds"`
	ExpectedFrameCount              int        `gorm:"column:expected_frame_count"`
	OCRCompletedCount               int        `gorm:"column:ocr_completed_count"`
	ExpectedSegmentCount            int        `gorm:"column:expected_segment_count"`
	CompletedSegmentCount           int        `gorm:"column:completed_segment_count"`
	RetentionRatioNeutral           float64    `gorm:"column:retention_ratio_neutral"`
	RetentionRatioEmpty             float64    `gorm:"column:retention_ratio_empty"`
	RetentionContextWindowSeconds   int        `gorm:"column:retention_context_window_seconds"`
	SegmentDurationSeconds          int        `gorm:"column:segment_duration_seconds"`
	MinVideoDurationForSplitSeconds int        `gorm:"column:min_video_duration_for_split_seconds"`
	MaxSegments                     int        `gorm:"column:max_segments"`
	CreatedAt                       time.Time  `gorm:"column:created_at"`
	StartedAt                       *time.Time `gorm:"column:started_at"`
	FinishedAt                      *time.Time `gorm:"column:finished_at"`
}

func (JobModel) TableName() string {
	return "jobs"
}

func jobModelFromDomain(j *domainjob.Job) *JobModel {
	return &JobModel{
		ID:                              j.ID,
		VideoID:                         j.VideoID,
		Type:                            j.Type,
		Status:                          j.Status,
		FailureReason:                   j.FailureReason,
		AnalysisFPS:                     j.AnalysisFPS,
		PHashDistanceThreshold:          j.PHashDistanceThreshold,
		MaxIntervalSeconds:              j.MaxIntervalSeconds,
		ExpectedFrameCount:              j.ExpectedFrameCount,
		OCRCompletedCount:               j.OCRCompletedCount,
		ExpectedSegmentCount:            j.ExpectedSegmentCount,
		CompletedSegmentCount:           j.CompletedSegmentCount,
		RetentionRatioNeutral:           j.RetentionRatioNeutral,
		RetentionRatioEmpty:             j.RetentionRatioEmpty,
		RetentionContextWindowSeconds:   j.RetentionContextWindowSeconds,
		SegmentDurationSeconds:          j.SegmentDurationSeconds,
		MinVideoDurationForSplitSeconds: j.MinVideoDurationForSplitSeconds,
		MaxSegments:                     j.MaxSegments,
		CreatedAt:                       j.CreatedAt,
		StartedAt:                       j.StartedAt,
		FinishedAt:                      j.FinishedAt,
	}
}

func jobDomainFromModel(m *JobModel) *domainjob.Job {
	return &domainjob.Job{
		ID:                              m.ID,
		VideoID:                         m.VideoID,
		Type:                            m.Type,
		Status:                          m.Status,
		FailureReason:                   m.FailureReason,
		AnalysisFPS:                     m.AnalysisFPS,
		PHashDistanceThreshold:          m.PHashDistanceThreshold,
		MaxIntervalSeconds:              m.MaxIntervalSeconds,
		ExpectedFrameCount:              m.ExpectedFrameCount,
		OCRCompletedCount:               m.OCRCompletedCount,
		ExpectedSegmentCount:            m.ExpectedSegmentCount,
		CompletedSegmentCount:           m.CompletedSegmentCount,
		RetentionRatioNeutral:           m.RetentionRatioNeutral,
		RetentionRatioEmpty:             m.RetentionRatioEmpty,
		RetentionContextWindowSeconds:   m.RetentionContextWindowSeconds,
		SegmentDurationSeconds:          m.SegmentDurationSeconds,
		MinVideoDurationForSplitSeconds: m.MinVideoDurationForSplitSeconds,
		MaxSegments:                     m.MaxSegments,
		CreatedAt:                       m.CreatedAt,
		StartedAt:                       m.StartedAt,
		FinishedAt:                      m.FinishedAt,
	}
}
