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
	ID               string    `json:"id"`
	OriginalFilename *string   `json:"originalFilename"`
	StorageKey       string    `json:"storageKey"`
	ThumbnailKey     *string   `json:"thumbnailKey"`
	ThumbnailURL     *string   `json:"thumbnailUrl"`
	SizeBytes        int64     `json:"sizeBytes"`
	ContentType      *string   `json:"contentType"`
	Status           string    `json:"status"`
	Source           string    `json:"source"`
	ScanJobID        *string   `json:"scanJobId"`
	ScanJobStatus    *string   `json:"scanJobStatus"`
	FrameCount       int       `json:"frameCount"`
	FailureReason    *string   `json:"failureReason"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
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
		ID:               view.ID.String(),
		OriginalFilename: optionalNonEmptyString(view.OriginalFilename),
		StorageKey:       view.StorageKey,
		ThumbnailKey:     optionalNonEmptyString(view.ThumbnailKey),
		ThumbnailURL:     optionalNonEmptyString(view.ThumbnailURL),
		SizeBytes:        view.SizeBytes,
		ContentType:      optionalNonEmptyString(view.ContentType),
		Status:           view.Status,
		Source:           view.Source,
		ScanJobID:        optionalUUIDString(view.ScanJobID),
		ScanJobStatus:    optionalNonEmptyString(view.ScanJobStatus),
		FrameCount:       view.FrameCount,
		FailureReason:    optionalNonEmptyString(view.FailureReason),
		CreatedAt:        view.CreatedAt,
		UpdatedAt:        view.UpdatedAt,
	}
}
