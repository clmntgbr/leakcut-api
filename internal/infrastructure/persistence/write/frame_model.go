package write

import (
	domainframe "go-api/internal/domain/frame"

	"github.com/google/uuid"
)

type FrameModel struct {
	ID              uuid.UUID `gorm:"column:id;primaryKey"`
	VideoID         uuid.UUID `gorm:"column:video_id"`
	Index           int       `gorm:"column:index"`
	TimestampMs     int64     `gorm:"column:timestamp_ms"`
	StorageKey      string    `gorm:"column:storage_key"`
	SelectionReason string    `gorm:"column:selection_reason"`
	PHashDistance   int       `gorm:"column:phash_distance"`
}

func (FrameModel) TableName() string {
	return "frames"
}

func frameModelFromDomain(f *domainframe.Frame) *FrameModel {
	return &FrameModel{
		ID:              f.ID,
		VideoID:         f.VideoID,
		Index:           f.Index,
		TimestampMs:     f.TimestampMs,
		StorageKey:      f.StorageKey,
		SelectionReason: f.SelectionReason,
		PHashDistance:   f.PHashDistance,
	}
}

func frameDomainFromModel(m *FrameModel) *domainframe.Frame {
	return &domainframe.Frame{
		ID:              m.ID,
		VideoID:         m.VideoID,
		Index:           m.Index,
		TimestampMs:     m.TimestampMs,
		StorageKey:      m.StorageKey,
		SelectionReason: m.SelectionReason,
		PHashDistance:   m.PHashDistance,
	}
}
