package write

import (
	"context"

	domainframe "go-api/internal/domain/frame"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type frameWriteRepository struct {
	db *gorm.DB
}

func NewFrameWriteRepository(db *gorm.DB) domainframe.FrameWriteRepository {
	return &frameWriteRepository{db: db}
}

func (r *frameWriteRepository) UpsertAll(ctx context.Context, frames []*domainframe.Frame) error {
	if len(frames) == 0 {
		return nil
	}

	rows := make([]FrameModel, 0, len(frames))
	for _, f := range frames {
		rows = append(rows, *frameModelFromDomain(f))
	}

	return DBWithContext(ctx, r.db).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "scan_job_id"}, {Name: "index"}},
			DoUpdates: clause.AssignmentColumns([]string{"timestamp_ms", "storage_key", "selection_reason", "diff_score"}),
		}).
		Create(&rows).Error
}

func (r *frameWriteRepository) CountByScanJobID(ctx context.Context, scanJobID uuid.UUID) (int, error) {
	var count int64
	err := DBWithContext(ctx, r.db).Model(&FrameModel{}).Where("scan_job_id = ?", scanJobID).Count(&count).Error
	return int(count), err
}
