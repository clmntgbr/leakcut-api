package read

import (
	"context"
	"errors"
	"strings"
	"time"

	domainjob "go-api/internal/domain/job"
	"go-api/internal/domain/paginate"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type videoViewRow struct {
	ID                    uuid.UUID
	OriginalFilename      string
	StorageKey            string
	ThumbnailKey          string
	SizeBytes             int64
	ContentType           string
	Status                string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	ExtractJobID          *uuid.UUID
	ExtractJobStatus      string
	ExtractFailureReason  string
	OCRJobID              *uuid.UUID
	OCRJobStatus          string
	OCRFailureReason      string
	ClassifyJobID         *uuid.UUID
	ClassifyJobStatus     string
	ClassifyFailureReason string
	FrameCount            int
	ExpectedFrameCount    int
	OCRCompletedCount     int
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
			extract_jobs.id AS extract_job_id,
			extract_jobs.status AS extract_job_status,
			extract_jobs.failure_reason AS extract_failure_reason,
			ocr_jobs.id AS ocr_job_id,
			ocr_jobs.status AS ocr_job_status,
			ocr_jobs.failure_reason AS ocr_failure_reason,
			classify_jobs.id AS classify_job_id,
			classify_jobs.status AS classify_job_status,
			classify_jobs.failure_reason AS classify_failure_reason,
			COALESCE(ocr_jobs.expected_frame_count, 0) AS expected_frame_count,
			COALESCE(ocr_jobs.ocr_completed_count, 0) AS ocr_completed_count,
			COALESCE((SELECT COUNT(*) FROM frames WHERE frames.job_id = extract_jobs.id), 0) AS frame_count
		`).
		Joins("LEFT JOIN jobs extract_jobs ON extract_jobs.video_id = videos.id AND extract_jobs.type = ?", domainjob.TypeExtractFrames).
		Joins("LEFT JOIN jobs ocr_jobs ON ocr_jobs.video_id = videos.id AND ocr_jobs.type = ?", domainjob.TypeOCR).
		Joins("LEFT JOIN jobs classify_jobs ON classify_jobs.video_id = videos.id AND classify_jobs.type = ?", domainjob.TypeClassify).
		Where("videos.id = ? AND videos.user_id = ?", id, userID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return videoViewFromRow(row), nil
}

func videoViewFromRow(row videoViewRow) *domainvideo.VideoView {
	jobs := make([]domainvideo.JobView, 0, 3)
	if row.ExtractJobID != nil {
		jobs = append(jobs, domainvideo.JobView{
			ID:            *row.ExtractJobID,
			Type:          domainjob.TypeExtractFrames,
			Status:        row.ExtractJobStatus,
			FrameCount:    row.FrameCount,
			FailureReason: row.ExtractFailureReason,
		})
	}
	if row.OCRJobID != nil {
		jobs = append(jobs, domainvideo.JobView{
			ID:                 *row.OCRJobID,
			Type:               domainjob.TypeOCR,
			Status:             row.OCRJobStatus,
			ExpectedFrameCount: row.ExpectedFrameCount,
			OCRCompletedCount:  row.OCRCompletedCount,
			FailureReason:      row.OCRFailureReason,
		})
	}
	if row.ClassifyJobID != nil {
		jobs = append(jobs, domainvideo.JobView{
			ID:            *row.ClassifyJobID,
			Type:          domainjob.TypeClassify,
			Status:        row.ClassifyJobStatus,
			FailureReason: row.ClassifyFailureReason,
		})
	}

	currentID := row.ExtractJobID
	currentStatus := row.ExtractJobStatus
	failureReason := row.ExtractFailureReason
	if row.OCRJobID != nil {
		currentID = row.OCRJobID
		currentStatus = row.OCRJobStatus
		failureReason = row.OCRFailureReason
	}
	if row.ClassifyJobID != nil {
		currentID = row.ClassifyJobID
		currentStatus = row.ClassifyJobStatus
		failureReason = row.ClassifyFailureReason
	}

	return &domainvideo.VideoView{
		ID:                 row.ID,
		OriginalFilename:   row.OriginalFilename,
		StorageKey:         row.StorageKey,
		ThumbnailKey:       row.ThumbnailKey,
		SizeBytes:          row.SizeBytes,
		ContentType:        row.ContentType,
		Status:             row.Status,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
		JobID:              currentID,
		JobStatus:          currentStatus,
		FrameCount:         row.FrameCount,
		ExpectedFrameCount: row.ExpectedFrameCount,
		OCRCompletedCount:  row.OCRCompletedCount,
		FailureReason:      failureReason,
		Jobs:               jobs,
	}
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
