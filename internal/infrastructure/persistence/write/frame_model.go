package write

import (
	domainframe "go-api/internal/domain/frame"

	"github.com/google/uuid"
)

type FrameModel struct {
	ID              uuid.UUID `gorm:"column:id;primaryKey"`
	JobID           uuid.UUID `gorm:"column:job_id"`
	Index           int       `gorm:"column:index"`
	TimestampMs     int64     `gorm:"column:timestamp_ms"`
	StorageKey      string    `gorm:"column:storage_key"`
	SelectionReason string    `gorm:"column:selection_reason"`
	DiffScore       float64   `gorm:"column:diff_score"`
}

func (FrameModel) TableName() string {
	return "frames"
}

func frameModelFromDomain(f *domainframe.Frame) *FrameModel {
	return &FrameModel{
		ID:              f.ID,
		JobID:           f.JobID,
		Index:           f.Index,
		TimestampMs:     f.TimestampMs,
		StorageKey:      f.StorageKey,
		SelectionReason: f.SelectionReason,
		DiffScore:       f.DiffScore,
	}
}
