package write

import (
	"context"

	domainclassification "go-api/internal/domain/classification"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type classificationWriteRepository struct {
	db *gorm.DB
}

func NewClassificationWriteRepository(db *gorm.DB) domainclassification.WriteRepository {
	return &classificationWriteRepository{db: db}
}

func (r *classificationWriteRepository) UpsertAll(ctx context.Context, items []*domainclassification.Classification) error {
	if len(items) == 0 {
		return nil
	}

	rows := make([]ClassificationModel, 0, len(items))
	for _, item := range items {
		rows = append(rows, *classificationModelFromDomain(item))
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

func (r *classificationWriteRepository) ListByFrameIDs(ctx context.Context, frameIDs []uuid.UUID) ([]*domainclassification.Classification, error) {
	if len(frameIDs) == 0 {
		return nil, nil
	}

	var rows []ClassificationModel
	if err := DBWithContext(ctx, r.db).Where("frame_id IN ?", frameIDs).Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]*domainclassification.Classification, 0, len(rows))
	for i := range rows {
		out = append(out, classificationDomainFromModel(&rows[i]))
	}
	return out, nil
}
