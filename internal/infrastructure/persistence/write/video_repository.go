package write

import (
	"context"
	"errors"
	"time"

	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type videoWriteRepository struct {
	db *gorm.DB
}

func NewVideoWriteRepository(db *gorm.DB) domainvideo.VideoWriteRepository {
	return &videoWriteRepository{db: db}
}

func (r *videoWriteRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ContextWithTx(ctx, tx))
	})
}

func (r *videoWriteRepository) Save(ctx context.Context, video *domainvideo.Video) error {
	return DBWithContext(ctx, r.db).Create(videoModelFromDomain(video)).Error
}

func (r *videoWriteRepository) Update(ctx context.Context, video *domainvideo.Video) error {
	return DBWithContext(ctx, r.db).Save(videoModelFromDomain(video)).Error
}

func (r *videoWriteRepository) GetByID(ctx context.Context, id uuid.UUID) (*domainvideo.Video, error) {
	return r.get(ctx, "id = ?", id)
}

func (r *videoWriteRepository) GetByStorageKey(ctx context.Context, storageKey string) (*domainvideo.Video, error) {
	return r.get(ctx, "storage_key = ?", storageKey)
}

func (r *videoWriteRepository) UpdateThumbnailKey(ctx context.Context, id uuid.UUID, thumbnailKey string) error {
	return DBWithContext(ctx, r.db).
		Model(&VideoModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"thumbnail_key": thumbnailKey,
			"updated_at":    time.Now().UTC(),
		}).Error
}

func (r *videoWriteRepository) ListExpiredPending(ctx context.Context, cutoff time.Time) ([]*domainvideo.Video, error) {
	var models []VideoModel
	err := DBWithContext(ctx, r.db).
		Where("status = ? AND created_at <= ?", domainvideo.StatusPendingUpload, cutoff).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	videos := make([]*domainvideo.Video, 0, len(models))
	for i := range models {
		videos = append(videos, videoDomainFromModel(&models[i]))
	}
	return videos, nil
}

func (r *videoWriteRepository) get(ctx context.Context, query string, args ...any) (*domainvideo.Video, error) {
	db := DBWithContext(ctx, r.db)
	if _, ok := ctx.Value(txCtxKey{}).(*gorm.DB); ok {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	var model VideoModel
	err := db.Where(query, args...).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return videoDomainFromModel(&model), nil
}
