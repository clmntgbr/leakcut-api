package presenter

import (
	"time"

	cmdvideo "go-api/internal/application/command/video"
	domainvideo "go-api/internal/domain/video"
)

type RequestUploadURLResponse struct {
	VideoID   string    `json:"videoId"`
	UploadURL string    `json:"uploadUrl"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func NewRequestUploadURLResponse(result *cmdvideo.RequestUploadURLResult) RequestUploadURLResponse {
	return RequestUploadURLResponse{
		VideoID:   result.VideoID,
		UploadURL: result.UploadURL,
		ExpiresAt: result.ExpiresAt,
	}
}

type VideoIngestResponse struct {
	VideoID string `json:"videoId"`
}

func NewVideoIngestResponse(result *cmdvideo.RequestIngestResult) VideoIngestResponse {
	return VideoIngestResponse{VideoID: result.VideoID}
}

type VideoDetailResponse struct {
	ID                 string             `json:"id"`
	OriginalFilename   *string            `json:"originalFilename"`
	StorageKey         string             `json:"storageKey"`
	ThumbnailKey       *string            `json:"thumbnailKey"`
	ThumbnailURL       *string            `json:"thumbnailUrl"`
	SizeBytes          int64              `json:"sizeBytes"`
	ContentType        *string            `json:"contentType"`
	Status             string             `json:"status"`
	JobID              *string            `json:"jobId"`
	JobStatus          *string            `json:"jobStatus"`
	FrameCount         int                `json:"frameCount"`
	ExpectedFrameCount int                `json:"expectedFrameCount"`
	OCRCompletedCount  int                `json:"ocrCompletedCount"`
	FailureReason      *string            `json:"failureReason"`
	Jobs               []VideoJobResponse `json:"jobs"`
	CreatedAt          time.Time          `json:"createdAt"`
	UpdatedAt          time.Time          `json:"updatedAt"`
}

type VideoJobResponse struct {
	ID                 string  `json:"id"`
	Type               string  `json:"type"`
	Status             string  `json:"status"`
	FrameCount         int     `json:"frameCount"`
	ExpectedFrameCount int     `json:"expectedFrameCount"`
	OCRCompletedCount  int     `json:"ocrCompletedCount"`
	FailureReason      *string `json:"failureReason,omitempty"`
}

type VideoListItemResponse struct {
	ID               string    `json:"id"`
	OriginalFilename *string   `json:"originalFilename"`
	ThumbnailURL     *string   `json:"thumbnailUrl"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
}

func NewVideoListResponseFromViews(views []domainvideo.VideoListView) []VideoListItemResponse {
	out := make([]VideoListItemResponse, 0, len(views))
	for _, view := range views {
		out = append(out, VideoListItemResponse{
			ID:               view.ID.String(),
			OriginalFilename: optionalNonEmptyString(view.OriginalFilename),
			ThumbnailURL:     optionalNonEmptyString(view.ThumbnailURL),
			Status:           view.Status,
			CreatedAt:        view.CreatedAt,
		})
	}
	return out
}

func NewVideoDetailResponseFromView(view domainvideo.VideoView) VideoDetailResponse {
	return VideoDetailResponse{
		ID:                 view.ID.String(),
		OriginalFilename:   optionalNonEmptyString(view.OriginalFilename),
		StorageKey:         view.StorageKey,
		ThumbnailKey:       optionalNonEmptyString(view.ThumbnailKey),
		ThumbnailURL:       optionalNonEmptyString(view.ThumbnailURL),
		SizeBytes:          view.SizeBytes,
		ContentType:        optionalNonEmptyString(view.ContentType),
		Status:             view.Status,
		JobID:              optionalUUIDString(view.JobID),
		JobStatus:          optionalNonEmptyString(view.JobStatus),
		FrameCount:         view.FrameCount,
		ExpectedFrameCount: view.ExpectedFrameCount,
		OCRCompletedCount:  view.OCRCompletedCount,
		FailureReason:      optionalNonEmptyString(view.FailureReason),
		Jobs:               newVideoJobResponses(view.Jobs),
		CreatedAt:          view.CreatedAt,
		UpdatedAt:          view.UpdatedAt,
	}
}

func newVideoJobResponses(views []domainvideo.JobView) []VideoJobResponse {
	out := make([]VideoJobResponse, 0, len(views))
	for _, job := range views {
		out = append(out, VideoJobResponse{
			ID:                 job.ID.String(),
			Type:               job.Type,
			Status:             job.Status,
			FrameCount:         job.FrameCount,
			ExpectedFrameCount: job.ExpectedFrameCount,
			OCRCompletedCount:  job.OCRCompletedCount,
			FailureReason:      optionalNonEmptyString(job.FailureReason),
		})
	}
	return out
}
