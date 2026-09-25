package write

import (
	"context"

	domainocr "go-api/internal/domain/ocr"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ocrWriteRepository struct {
	db *gorm.DB
}

func NewOCRWriteRepository(db *gorm.DB) domainocr.WriteRepository {
	return &ocrWriteRepository{db: db}
}

func (r *ocrWriteRepository) UpsertAll(ctx context.Context, results []*domainocr.Result) error {
	if len(results) == 0 {
		return nil
	}

	rows := make([]OCRModel, 0, len(results))
	for _, item := range results {
		rows = append(rows, *ocrModelFromDomain(item))
	}

	return DBWithContext(ctx, r.db).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "frame_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"text", "confidence", "status", "error_reason", "lines"}),
		}).
		Create(&rows).Error
}

func (r *ocrWriteRepository) ListByFrameIDs(ctx context.Context, frameIDs []uuid.UUID) ([]*domainocr.Result, error) {
	if len(frameIDs) == 0 {
		return nil, nil
	}

	var rows []OCRModel
	if err := DBWithContext(ctx, r.db).Where("frame_id IN ?", frameIDs).Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]*domainocr.Result, 0, len(rows))
	for i := range rows {
		out = append(out, ocrDomainFromModel(&rows[i]))
	}
	return out, nil
}
