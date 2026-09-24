package write

import (
	"time"

	domainjob "go-api/internal/domain/job"

	"github.com/google/uuid"
)

type JobModel struct {
	ID                     uuid.UUID  `gorm:"column:id;primaryKey"`
	VideoID                uuid.UUID  `gorm:"column:video_id"`
	Type                   string     `gorm:"column:type"`
	Status                 string     `gorm:"column:status"`
	FailureReason          string     `gorm:"column:failure_reason"`
	AnalysisFPS            float64    `gorm:"column:analysis_fps"`
	PHashDistanceThreshold int        `gorm:"column:phash_distance_threshold"`
	MaxIntervalSeconds     int        `gorm:"column:max_interval_seconds"`
	ExpectedFrameCount     int        `gorm:"column:expected_frame_count"`
	OCRCompletedCount      int        `gorm:"column:ocr_completed_count"`
	CreatedAt              time.Time  `gorm:"column:created_at"`
	CompletedAt            *time.Time `gorm:"column:completed_at"`
}

func (JobModel) TableName() string {
	return "jobs"
}

func jobModelFromDomain(j *domainjob.Job) *JobModel {
	return &JobModel{
		ID:                     j.ID,
		VideoID:                j.VideoID,
		Type:                   j.Type,
		Status:                 j.Status,
		FailureReason:          j.FailureReason,
		AnalysisFPS:            j.AnalysisFPS,
		PHashDistanceThreshold: j.PHashDistanceThreshold,
		MaxIntervalSeconds:     j.MaxIntervalSeconds,
		ExpectedFrameCount:     j.ExpectedFrameCount,
		OCRCompletedCount:      j.OCRCompletedCount,
		CreatedAt:              j.CreatedAt,
		CompletedAt:            j.CompletedAt,
	}
}

func jobDomainFromModel(m *JobModel) *domainjob.Job {
	return &domainjob.Job{
		ID:                     m.ID,
		VideoID:                m.VideoID,
		Type:                   m.Type,
		Status:                 m.Status,
		FailureReason:          m.FailureReason,
		AnalysisFPS:            m.AnalysisFPS,
		PHashDistanceThreshold: m.PHashDistanceThreshold,
		MaxIntervalSeconds:     m.MaxIntervalSeconds,
		ExpectedFrameCount:     m.ExpectedFrameCount,
		OCRCompletedCount:      m.OCRCompletedCount,
		CreatedAt:              m.CreatedAt,
		CompletedAt:            m.CompletedAt,
	}
}
