package write

import (
	"time"

	domainscanjob "go-api/internal/domain/scanjob"

	"github.com/google/uuid"
)

type ScanJobModel struct {
	ID                 uuid.UUID  `gorm:"column:id;primaryKey"`
	VideoID            uuid.UUID  `gorm:"column:video_id"`
	Status             string     `gorm:"column:status"`
	FailureReason      string     `gorm:"column:failure_reason"`
	AnalysisFPS        float64    `gorm:"column:analysis_fps"`
	DiffThreshold      float64    `gorm:"column:diff_threshold"`
	MaxIntervalSeconds int        `gorm:"column:max_interval_seconds"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	CompletedAt        *time.Time `gorm:"column:completed_at"`
}

func (ScanJobModel) TableName() string {
	return "scan_jobs"
}

func scanJobModelFromDomain(j *domainscanjob.ScanJob) *ScanJobModel {
	return &ScanJobModel{
		ID:                 j.ID,
		VideoID:            j.VideoID,
		Status:             j.Status,
		FailureReason:      j.FailureReason,
		AnalysisFPS:        j.AnalysisFPS,
		DiffThreshold:      j.DiffThreshold,
		MaxIntervalSeconds: j.MaxIntervalSeconds,
		CreatedAt:          j.CreatedAt,
		CompletedAt:        j.CompletedAt,
	}
}

func scanJobDomainFromModel(m *ScanJobModel) *domainscanjob.ScanJob {
	return &domainscanjob.ScanJob{
		ID:                 m.ID,
		VideoID:            m.VideoID,
		Status:             m.Status,
		FailureReason:      m.FailureReason,
		AnalysisFPS:        m.AnalysisFPS,
		DiffThreshold:      m.DiffThreshold,
		MaxIntervalSeconds: m.MaxIntervalSeconds,
		CreatedAt:          m.CreatedAt,
		CompletedAt:        m.CompletedAt,
	}
}
