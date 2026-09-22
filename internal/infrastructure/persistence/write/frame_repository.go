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
			Columns:   []clause.Column{{Name: "job_id"}, {Name: "index"}},
			DoUpdates: clause.AssignmentColumns([]string{"timestamp_ms", "storage_key", "selection_reason", "diff_score"}),
		}).
		Create(&rows).Error
}

func (r *frameWriteRepository) ListByJobID(ctx context.Context, jobID uuid.UUID) ([]*domainframe.Frame, error) {
	var rows []FrameModel
	if err := DBWithContext(ctx, r.db).
		Where("job_id = ?", jobID).
		Order(`"index" ASC`).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]*domainframe.Frame, 0, len(rows))
	for i := range rows {
		out = append(out, frameDomainFromModel(&rows[i]))
	}
	return out, nil
}

func (r *frameWriteRepository) CountByJobID(ctx context.Context, jobID uuid.UUID) (int, error) {
	var count int64
	err := DBWithContext(ctx, r.db).Model(&FrameModel{}).Where("job_id = ?", jobID).Count(&count).Error
	return int(count), err
}
