package read

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-api/internal/domain/paginate"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type videoViewRow struct {
	ID               uuid.UUID
	OriginalFilename string
	StorageKey       string
	ThumbnailKey     string
	SizeBytes        int64
	ContentType      string
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	JobID            *uuid.UUID
	JobStatus        string
	FrameCount       int
	FailureReason    string
}

func (videoViewRow) TableName() string { return "videos" }

type videoReadRepository struct {
	db *gorm.DB
}

func NewVideoReadRepository(db *gorm.DB) domainvideo.VideoReadRepository {
	return &videoReadRepository{db: db}
}

func (r *videoReadRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*domainvideo.VideoView, error) {
	var row videoViewRow
	err := r.db.WithContext(ctx).
		Table("videos").
		Select(`
			videos.id,
			videos.original_filename,
			videos.storage_key,
			videos.thumbnail_key,
			videos.size_bytes,
			videos.content_type,
			videos.status,
			videos.created_at,
			videos.updated_at,
			jobs.id AS job_id,
			jobs.status AS job_status,
			jobs.failure_reason,
			COALESCE((SELECT COUNT(*) FROM frames WHERE frames.job_id = jobs.id), 0) AS frame_count
		`).
		Joins("LEFT JOIN jobs ON jobs.video_id = videos.id").
		Where("videos.id = ? AND videos.user_id = ?", id, userID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &domainvideo.VideoView{
		ID:               row.ID,
		OriginalFilename: row.OriginalFilename,
		StorageKey:       row.StorageKey,
		ThumbnailKey:     row.ThumbnailKey,
		SizeBytes:        row.SizeBytes,
		ContentType:      row.ContentType,
		Status:           row.Status,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		JobID:            row.JobID,
		JobStatus:        row.JobStatus,
		FrameCount:       row.FrameCount,
		FailureReason:    row.FailureReason,
	}, nil
}

type videoListRow struct {
	ID               uuid.UUID
	OriginalFilename string
	ThumbnailKey     string
	Status           string
	CreatedAt        time.Time
}

func (videoListRow) TableName() string { return "videos" }

var videoListSortColumns = map[string]string{
	"created_at":        "created_at",
	"updated_at":        "updated_at",
	"status":            "status",
	"original_filename": "original_filename",
}

func (r *videoReadRepository) List(ctx context.Context, userID uuid.UUID, query paginate.PaginateQuery) ([]domainvideo.VideoListView, int64, error) {
	query.SortBy = normalizeVideoListSort(query.SortBy)

	db := r.db.WithContext(ctx).Table("videos").Where("user_id = ?", userID)
	if search := strings.TrimSpace(query.Search); search != "" {
		db = db.Where("original_filename ILIKE ? ESCAPE '\\'", "%"+escapeLike(search)+"%")
	}

	scoped, total, err := Paginate(db, query)
	if err != nil {
		return nil, 0, err
	}

	var rows []videoListRow
	if err := scoped.Select("id, original_filename, thumbnail_key, status, created_at").Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	views := make([]domainvideo.VideoListView, 0, len(rows))
	for _, row := range rows {
		views = append(views, domainvideo.VideoListView{
			ID:               row.ID,
			OriginalFilename: row.OriginalFilename,
			ThumbnailKey:     row.ThumbnailKey,
			Status:           row.Status,
			CreatedAt:        row.CreatedAt,
		})
	}
	return views, total, nil
}

func normalizeVideoListSort(sortBy string) string {
	if column, ok := videoListSortColumns[sortBy]; ok {
		return column
	}
	return "created_at"
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}
