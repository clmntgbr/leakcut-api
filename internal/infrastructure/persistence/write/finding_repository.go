package write

import (
	"context"

	domainfinding "go-api/internal/domain/finding"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type findingWriteRepository struct {
	db *gorm.DB
}

func NewFindingWriteRepository(db *gorm.DB) domainfinding.FindingWriteRepository {
	return &findingWriteRepository{db: db}
}

func (r *findingWriteRepository) UpsertAll(ctx context.Context, findings []*domainfinding.Finding) error {
	if len(findings) == 0 {
		return nil
	}

	rows := make([]FindingModel, 0, len(findings))
	for _, item := range findings {
		rows = append(rows, *findingModelFromDomain(item))
	}

	return DBWithContext(ctx, r.db).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "frame_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"confidential",
				"probability",
				"categories",
				"status",
				"error_reason",
			}),
		}).
		Create(&rows).Error
}

func (r *findingWriteRepository) ListByFrameIDs(ctx context.Context, frameIDs []uuid.UUID) ([]*domainfinding.Finding, error) {
	if len(frameIDs) == 0 {
		return nil, nil
	}

	var rows []FindingModel
	if err := DBWithContext(ctx, r.db).Where("frame_id IN ?", frameIDs).Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]*domainfinding.Finding, 0, len(rows))
	for i := range rows {
		out = append(out, findingDomainFromModel(&rows[i]))
	}
	return out, nil
}
