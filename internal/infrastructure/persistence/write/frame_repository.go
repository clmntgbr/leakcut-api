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

	if err := DBWithContext(ctx, r.db).
		Clauses(
			clause.OnConflict{
				Columns:   []clause.Column{{Name: "video_id"}, {Name: "segment_index"}, {Name: "index"}},
				DoUpdates: clause.AssignmentColumns([]string{"timestamp_ms", "storage_key", "selection_reason", "phash_distance"}),
			},
			clause.Returning{Columns: []clause.Column{{Name: "id"}, {Name: "segment_index"}, {Name: "index"}}},
		).
		Create(&rows).Error; err != nil {
		return err
	}

	type key struct {
		SegmentIndex int
		Index        int
	}
	byKey := make(map[key]uuid.UUID, len(rows))
	for i := range rows {
		byKey[key{SegmentIndex: rows[i].SegmentIndex, Index: rows[i].Index}] = rows[i].ID
	}
	for _, frame := range frames {
		if id, ok := byKey[key{SegmentIndex: frame.SegmentIndex, Index: frame.Index}]; ok {
			frame.ID = id
		}
	}
	return nil
}

func (r *frameWriteRepository) UpdateRetention(ctx context.Context, frames []*domainframe.Frame) error {
	if len(frames) == 0 {
		return nil
	}
	db := DBWithContext(ctx, r.db)
	for _, frame := range frames {
		if err := db.Model(&FrameModel{}).
			Where("id = ?", frame.ID).
			Updates(map[string]any{
				"retained":     frame.Retained,
				"prune_reason": frame.PruneReason,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *frameWriteRepository) ListByVideoID(ctx context.Context, videoID uuid.UUID) ([]*domainframe.Frame, error) {
	var rows []FrameModel
	if err := DBWithContext(ctx, r.db).
		Where("video_id = ?", videoID).
		Order(`segment_index ASC, "index" ASC`).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]*domainframe.Frame, 0, len(rows))
	for i := range rows {
		out = append(out, frameDomainFromModel(&rows[i]))
	}
	return out, nil
}

func (r *frameWriteRepository) CountByVideoID(ctx context.Context, videoID uuid.UUID) (int, error) {
	var count int64
	err := DBWithContext(ctx, r.db).Model(&FrameModel{}).Where("video_id = ?", videoID).Count(&count).Error
	return int(count), err
}
