package write

import (
	"context"

	domainocr "go-api/internal/domain/ocrresult"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ocrResultWriteRepository struct {
	db *gorm.DB
}

func NewOCRResultWriteRepository(db *gorm.DB) domainocr.ResultWriteRepository {
	return &ocrResultWriteRepository{db: db}
}

func (r *ocrResultWriteRepository) UpsertAll(ctx context.Context, results []*domainocr.Result) error {
	if len(results) == 0 {
		return nil
	}

	rows := make([]OCRResultModel, 0, len(results))
	for _, item := range results {
		rows = append(rows, *ocrResultModelFromDomain(item))
	}

	return DBWithContext(ctx, r.db).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "frame_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"text", "confidence", "status", "error_reason"}),
		}).
		Create(&rows).Error
}

func (r *ocrResultWriteRepository) ListByFrameIDs(ctx context.Context, frameIDs []uuid.UUID) ([]*domainocr.Result, error) {
	if len(frameIDs) == 0 {
		return nil, nil
	}

	var rows []OCRResultModel
	if err := DBWithContext(ctx, r.db).Where("frame_id IN ?", frameIDs).Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]*domainocr.Result, 0, len(rows))
	for i := range rows {
		out = append(out, ocrResultDomainFromModel(&rows[i]))
	}
	return out, nil
}
