package write

import (
	"context"

	domainsegment "go-api/internal/domain/segment"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type segmentWriteRepository struct {
	db *gorm.DB
}

func NewSegmentWriteRepository(db *gorm.DB) domainsegment.WriteRepository {
	return &segmentWriteRepository{db: db}
}

func (r *segmentWriteRepository) SaveAll(ctx context.Context, segments []*domainsegment.Segment) error {
	if len(segments) == 0 {
		return nil
	}
	rows := make([]SegmentModel, 0, len(segments))
	for _, s := range segments {
		rows = append(rows, *segmentModelFromDomain(s))
	}
	return DBWithContext(ctx, r.db).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "video_id"}, {Name: "segment_index"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"job_id", "offset_ms", "duration_ms", "storage_key", "status", "failure_reason", "completed_at",
			}),
		}).
		Create(&rows).Error
}

func (r *segmentWriteRepository) Update(ctx context.Context, segment *domainsegment.Segment) error {
	return DBWithContext(ctx, r.db).Save(segmentModelFromDomain(segment)).Error
}

func (r *segmentWriteRepository) GetByID(ctx context.Context, id uuid.UUID) (*domainsegment.Segment, error) {
	var model SegmentModel
	err := DBWithContext(ctx, r.db).Where("id = ?", id).First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return segmentDomainFromModel(&model), nil
}

func (r *segmentWriteRepository) ListByVideoID(ctx context.Context, videoID uuid.UUID) ([]*domainsegment.Segment, error) {
	var rows []SegmentModel
	if err := DBWithContext(ctx, r.db).
		Where("video_id = ?", videoID).
		Order("segment_index ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainsegment.Segment, 0, len(rows))
	for i := range rows {
		out = append(out, segmentDomainFromModel(&rows[i]))
	}
	return out, nil
}

func (r *segmentWriteRepository) GetByVideoIDAndIndex(ctx context.Context, videoID uuid.UUID, index int) (*domainsegment.Segment, error) {
	var model SegmentModel
	err := DBWithContext(ctx, r.db).
		Where("video_id = ? AND segment_index = ?", videoID, index).
		First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return segmentDomainFromModel(&model), nil
}
