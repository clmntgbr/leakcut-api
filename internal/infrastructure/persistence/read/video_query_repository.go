package read

import (
	"context"
	"encoding/json"
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
	ID                      uuid.UUID
	OriginalFilename        string
	StorageKey              string
	ThumbnailKey            string
	SizeBytes               int64
	ContentType             string
	Status                  string
	CreatedAt               time.Time
	UpdatedAt               time.Time
	FinishedAt              *time.Time
	SegmentJobID            *uuid.UUID
	SegmentJobStatus        string
	SegmentFailureReason    string
	SegmentCreatedAt        time.Time
	SegmentStartedAt        *time.Time
	SegmentFinishedAt       *time.Time
	ExpectedSegmentCount    int
	CompletedSegmentCount   int
	ExtractJobID            *uuid.UUID
	ExtractJobStatus        string
	ExtractFailureReason    string
	ExtractCreatedAt        time.Time
	ExtractStartedAt        *time.Time
	ExtractFinishedAt       *time.Time
	OCRJobID                *uuid.UUID
	OCRJobStatus            string
	OCRFailureReason        string
	OCRCreatedAt            time.Time
	OCRStartedAt            *time.Time
	OCRFinishedAt           *time.Time
	ClassifyJobID           *uuid.UUID
	ClassifyJobStatus       string
	ClassifyFailureReason   string
	ClassifyCreatedAt       time.Time
	ClassifyStartedAt       *time.Time
	ClassifyFinishedAt      *time.Time
	FrameCount              int
	ExpectedFrameCount      int
	OCRCompletedCount       int
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
			videos.finished_at,
			segment_jobs.id AS segment_job_id,
			segment_jobs.status AS segment_job_status,
			segment_jobs.failure_reason AS segment_failure_reason,
			segment_jobs.created_at AS segment_created_at,
			segment_jobs.started_at AS segment_started_at,
			segment_jobs.finished_at AS segment_finished_at,
			COALESCE(segment_jobs.expected_segment_count, extract_jobs.expected_segment_count, 0) AS expected_segment_count,
			COALESCE(extract_jobs.completed_segment_count, 0) AS completed_segment_count,
			extract_jobs.id AS extract_job_id,
			extract_jobs.status AS extract_job_status,
			extract_jobs.failure_reason AS extract_failure_reason,
			extract_jobs.created_at AS extract_created_at,
			extract_jobs.started_at AS extract_started_at,
			extract_jobs.finished_at AS extract_finished_at,
			ocr_jobs.id AS ocr_job_id,
			ocr_jobs.status AS ocr_job_status,
			ocr_jobs.failure_reason AS ocr_failure_reason,
			ocr_jobs.created_at AS ocr_created_at,
			ocr_jobs.started_at AS ocr_started_at,
			ocr_jobs.finished_at AS ocr_finished_at,
			classify_jobs.id AS classify_job_id,
			classify_jobs.status AS classify_job_status,
			classify_jobs.failure_reason AS classify_failure_reason,
			classify_jobs.created_at AS classify_created_at,
			classify_jobs.started_at AS classify_started_at,
			classify_jobs.finished_at AS classify_finished_at,
			COALESCE(ocr_jobs.expected_frame_count, 0) AS expected_frame_count,
			COALESCE(ocr_jobs.ocr_completed_count, 0) AS ocr_completed_count,
			COALESCE((SELECT COUNT(*) FROM frames WHERE frames.video_id = videos.id), 0) AS frame_count
		`).
		Joins("LEFT JOIN jobs segment_jobs ON segment_jobs.video_id = videos.id AND segment_jobs.type = ?", domainjob.TypeSegment).
		Joins("LEFT JOIN jobs extract_jobs ON extract_jobs.video_id = videos.id AND extract_jobs.type = ?", domainjob.TypeFrame).
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
	jobs := make([]domainvideo.JobView, 0, 4)
	if row.SegmentJobID != nil {
		jobs = append(jobs, domainvideo.JobView{
			ID:                   *row.SegmentJobID,
			Type:                 domainjob.TypeSegment,
			Status:               row.SegmentJobStatus,
			ExpectedSegmentCount: row.ExpectedSegmentCount,
			FailureReason:        row.SegmentFailureReason,
			CreatedAt:            row.SegmentCreatedAt,
			StartedAt:            row.SegmentStartedAt,
			FinishedAt:           row.SegmentFinishedAt,
		})
	}
	if row.ExtractJobID != nil {
		jobs = append(jobs, domainvideo.JobView{
			ID:                    *row.ExtractJobID,
			Type:                  domainjob.TypeFrame,
			Status:                row.ExtractJobStatus,
			FrameCount:            row.FrameCount,
			ExpectedSegmentCount:  row.ExpectedSegmentCount,
			CompletedSegmentCount: row.CompletedSegmentCount,
			FailureReason:         row.ExtractFailureReason,
			CreatedAt:             row.ExtractCreatedAt,
			StartedAt:             row.ExtractStartedAt,
			FinishedAt:            row.ExtractFinishedAt,
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
			CreatedAt:          row.OCRCreatedAt,
			StartedAt:          row.OCRStartedAt,
			FinishedAt:         row.OCRFinishedAt,
		})
	}
	if row.ClassifyJobID != nil {
		jobs = append(jobs, domainvideo.JobView{
			ID:            *row.ClassifyJobID,
			Type:          domainjob.TypeClassify,
			Status:        row.ClassifyJobStatus,
			FailureReason: row.ClassifyFailureReason,
			CreatedAt:     row.ClassifyCreatedAt,
			StartedAt:     row.ClassifyStartedAt,
			FinishedAt:    row.ClassifyFinishedAt,
		})
	}

	currentID := row.SegmentJobID
	currentStatus := row.SegmentJobStatus
	failureReason := row.SegmentFailureReason
	if row.ExtractJobID != nil {
		currentID = row.ExtractJobID
		currentStatus = row.ExtractJobStatus
		failureReason = row.ExtractFailureReason
	}
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
		FinishedAt:         row.FinishedAt,
		JobID:              currentID,
		JobStatus:          currentStatus,
		FrameCount:         row.FrameCount,
		ExpectedFrameCount: row.ExpectedFrameCount,
		OCRCompletedCount:  row.OCRCompletedCount,
		FailureReason:      failureReason,
		Jobs:               jobs,
		Frames:             make([]domainvideo.VideoFrameDetailView, 0),
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

type frameDetailRow struct {
	ID                   uuid.UUID  `gorm:"column:id"`
	Index                int        `gorm:"column:index"`
	TimestampMs          int64      `gorm:"column:timestamp_ms"`
	StorageKey           string     `gorm:"column:storage_key"`
	SelectionReason      string     `gorm:"column:selection_reason"`
	PHashDistance        int        `gorm:"column:phash_distance"`
	Retained             bool       `gorm:"column:retained"`
	PruneReason          string     `gorm:"column:prune_reason"`
	OCRText              string     `gorm:"column:ocr_text"`
	OCRStatus            string     `gorm:"column:ocr_status"`
	OCRConfidence        float64    `gorm:"column:ocr_confidence"`
	OCRErrorReason       string     `gorm:"column:ocr_error_reason"`
	OCRLines             string     `gorm:"column:ocr_lines"`
	ClassificationID     *uuid.UUID `gorm:"column:classification_id"`
	Confidential         bool       `gorm:"column:confidential"`
	Probability          float64    `gorm:"column:probability"`
	Categories           string     `gorm:"column:categories"`
	ClassificationStatus string     `gorm:"column:classification_status"`
	ClassificationError  string     `gorm:"column:classification_error"`
}

func (r *videoReadRepository) ListFramesByVideoID(ctx context.Context, id, userID uuid.UUID) ([]domainvideo.VideoFrameDetailView, error) {
	var rows []frameDetailRow
	err := r.db.WithContext(ctx).
		Table("frames").
		Select(`
			frames.id,
			frames.index,
			frames.timestamp_ms,
			frames.storage_key,
			frames.selection_reason,
			frames.phash_distance,
			frames.retained,
			COALESCE(frames.prune_reason, '') AS prune_reason,
			COALESCE(ocrs.text, '') AS ocr_text,
			COALESCE(ocrs.status, '') AS ocr_status,
			COALESCE(ocrs.confidence, 0) AS ocr_confidence,
			COALESCE(ocrs.error_reason, '') AS ocr_error_reason,
			COALESCE(ocrs.lines::text, '[]') AS ocr_lines,
			classifications.id AS classification_id,
			COALESCE(classifications.confidential, FALSE) AS confidential,
			COALESCE(classifications.probability, 0) AS probability,
			COALESCE(classifications.categories::text, '[]') AS categories,
			COALESCE(classifications.status, '') AS classification_status,
			COALESCE(classifications.error_reason, '') AS classification_error
		`).
		Joins("JOIN videos ON videos.id = frames.video_id").
		Joins("LEFT JOIN ocrs ON ocrs.frame_id = frames.id").
		Joins("LEFT JOIN classifications ON classifications.frame_id = frames.id").
		Where("videos.id = ? AND videos.user_id = ?", id, userID).
		Order("frames.segment_index ASC, frames.index ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]domainvideo.VideoFrameDetailView, 0, len(rows))
	for _, row := range rows {
		out = append(out, domainvideo.VideoFrameDetailView{
			ID:              row.ID,
			Index:           row.Index,
			TimestampMs:     row.TimestampMs,
			StorageKey:      row.StorageKey,
			SelectionReason: row.SelectionReason,
			PHashDistance:   row.PHashDistance,
			Retained:        row.Retained,
			PruneReason:     row.PruneReason,
			OCRText:         row.OCRText,
			OCRStatus:       row.OCRStatus,
			OCRConfidence:   row.OCRConfidence,
			OCRErrorReason:  row.OCRErrorReason,
			OCRLines:        ocrLinesFromRow(row.OCRLines),
			Classification:  classificationViewFromRow(row),
		})
	}
	return out, nil
}

func ocrLinesFromRow(raw string) []domainvideo.OCRLineView {
	out := make([]domainvideo.OCRLineView, 0)
	if raw == "" {
		return out
	}
	var lines []struct {
		Text       string  `json:"text"`
		Confidence float64 `json:"confidence"`
		Box        []struct {
			X int `json:"x"`
			Y int `json:"y"`
		} `json:"box"`
	}
	if err := json.Unmarshal([]byte(raw), &lines); err != nil {
		return out
	}
	for _, line := range lines {
		box := make([]domainvideo.OCRPointView, 0, len(line.Box))
		for _, point := range line.Box {
			box = append(box, domainvideo.OCRPointView{X: point.X, Y: point.Y})
		}
		out = append(out, domainvideo.OCRLineView{
			Text:       line.Text,
			Confidence: line.Confidence,
			Box:        box,
		})
	}
	return out
}

func classificationViewFromRow(row frameDetailRow) *domainvideo.VideoFrameClassificationView {
	if row.ClassificationID == nil {
		return nil
	}
	categories := make([]domainvideo.ClassificationCategoryView, 0)
	if row.Categories != "" {
		_ = json.Unmarshal([]byte(row.Categories), &categories)
	}
	return &domainvideo.VideoFrameClassificationView{
		ID:           *row.ClassificationID,
		Confidential: row.Confidential,
		Probability:  row.Probability,
		Categories:   categories,
		Status:       row.ClassificationStatus,
		ErrorReason:  row.ClassificationError,
	}
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}
