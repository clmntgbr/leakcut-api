package write

import (
	"time"

	domainsegment "go-api/internal/domain/segment"

	"github.com/google/uuid"
)

type SegmentModel struct {
	ID            uuid.UUID  `gorm:"column:id;primaryKey"`
	VideoID       uuid.UUID  `gorm:"column:video_id"`
	JobID         uuid.UUID  `gorm:"column:job_id"`
	SegmentIndex  int        `gorm:"column:segment_index"`
	OffsetMs      int64      `gorm:"column:offset_ms"`
	DurationMs    int64      `gorm:"column:duration_ms"`
	StorageKey    string     `gorm:"column:storage_key"`
	Status        string     `gorm:"column:status"`
	FailureReason string     `gorm:"column:failure_reason"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	CompletedAt   *time.Time `gorm:"column:completed_at"`
}

func (SegmentModel) TableName() string {
	return "video_segments"
}

func segmentModelFromDomain(s *domainsegment.Segment) *SegmentModel {
	return &SegmentModel{
		ID:            s.ID,
		VideoID:       s.VideoID,
		JobID:         s.JobID,
		SegmentIndex:  s.SegmentIndex,
		OffsetMs:      s.OffsetMs,
		DurationMs:    s.DurationMs,
		StorageKey:    s.StorageKey,
		Status:        s.Status,
		FailureReason: s.FailureReason,
		CreatedAt:     s.CreatedAt,
		CompletedAt:   s.CompletedAt,
	}
}

func segmentDomainFromModel(m *SegmentModel) *domainsegment.Segment {
	return &domainsegment.Segment{
		ID:            m.ID,
		VideoID:       m.VideoID,
		JobID:         m.JobID,
		SegmentIndex:  m.SegmentIndex,
		OffsetMs:      m.OffsetMs,
		DurationMs:    m.DurationMs,
		StorageKey:    m.StorageKey,
		Status:        m.Status,
		FailureReason: m.FailureReason,
		CreatedAt:     m.CreatedAt,
		CompletedAt:   m.CompletedAt,
	}
}
