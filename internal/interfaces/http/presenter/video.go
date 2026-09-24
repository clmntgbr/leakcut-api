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
	ID                 string               `json:"id"`
	OriginalFilename   *string              `json:"originalFilename"`
	StorageKey         string               `json:"storageKey"`
	ThumbnailKey       *string              `json:"thumbnailKey"`
	ThumbnailURL       *string              `json:"thumbnailUrl"`
	VideoURL           *string              `json:"videoUrl"`
	SizeBytes          int64                `json:"sizeBytes"`
	ContentType        *string              `json:"contentType"`
	Status             string               `json:"status"`
	JobID              *string              `json:"jobId"`
	JobStatus          *string              `json:"jobStatus"`
	FrameCount         int                  `json:"frameCount"`
	ExpectedFrameCount int                  `json:"expectedFrameCount"`
	OCRCompletedCount  int                  `json:"ocrCompletedCount"`
	FailureReason      *string              `json:"failureReason"`
	Jobs               []VideoJobResponse   `json:"jobs"`
	Frames             []VideoFrameResponse `json:"frames"`
	CreatedAt          time.Time            `json:"createdAt"`
	UpdatedAt          time.Time            `json:"updatedAt"`
}

type VideoFrameResponse struct {
	ID              string                 `json:"id"`
	Index           int                    `json:"index"`
	TimestampMs     int64                  `json:"timestampMs"`
	StorageKey      string                 `json:"storageKey"`
	ImageURL        *string                `json:"imageUrl"`
	SelectionReason string                 `json:"selectionReason"`
	PHashDistance   int                    `json:"phashDistance"`
	OCRText         string                 `json:"ocrText"`
	OCRStatus       *string                `json:"ocrStatus"`
	OCRConfidence   float64                `json:"ocrConfidence"`
	OCRErrorReason  *string                `json:"ocrErrorReason"`
	OCRLines        []VideoOCRLineResponse `json:"ocrLines"`
	Finding         *VideoFindingResponse  `json:"finding"`
}

type VideoOCRPointResponse struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type VideoOCRLineResponse struct {
	Text       string                  `json:"text"`
	Confidence float64                 `json:"confidence"`
	Box        []VideoOCRPointResponse `json:"box"`
}

type VideoFindingCategoryResponse struct {
	Name        string  `json:"name"`
	Probability float64 `json:"probability"`
}

type VideoFindingResponse struct {
	ID           string                         `json:"id"`
	Confidential bool                           `json:"confidential"`
	Probability  float64                        `json:"probability"`
	Categories   []VideoFindingCategoryResponse `json:"categories"`
	Status       string                         `json:"status"`
	ErrorReason  *string                        `json:"errorReason"`
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
		VideoURL:           optionalNonEmptyString(view.VideoURL),
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
		Frames:             newVideoFrameResponses(view.Frames),
		CreatedAt:          view.CreatedAt,
		UpdatedAt:          view.UpdatedAt,
	}
}

func newVideoFrameResponses(views []domainvideo.VideoFrameDetailView) []VideoFrameResponse {
	out := make([]VideoFrameResponse, 0, len(views))
	for _, frame := range views {
		out = append(out, VideoFrameResponse{
			ID:              frame.ID.String(),
			Index:           frame.Index,
			TimestampMs:     frame.TimestampMs,
			StorageKey:      frame.StorageKey,
			ImageURL:        optionalNonEmptyString(frame.ImageURL),
			SelectionReason: frame.SelectionReason,
			PHashDistance:   frame.PHashDistance,
			OCRText:         frame.OCRText,
			OCRStatus:       optionalNonEmptyString(frame.OCRStatus),
			OCRConfidence:   frame.OCRConfidence,
			OCRErrorReason:  optionalNonEmptyString(frame.OCRErrorReason),
			OCRLines:        newVideoOCRLineResponses(frame.OCRLines),
			Finding:         newVideoFindingResponse(frame.Finding),
		})
	}
	return out
}

func newVideoOCRLineResponses(lines []domainvideo.OCRLineView) []VideoOCRLineResponse {
	out := make([]VideoOCRLineResponse, 0, len(lines))
	for _, line := range lines {
		box := make([]VideoOCRPointResponse, 0, len(line.Box))
		for _, point := range line.Box {
			box = append(box, VideoOCRPointResponse{X: point.X, Y: point.Y})
		}
		out = append(out, VideoOCRLineResponse{
			Text:       line.Text,
			Confidence: line.Confidence,
			Box:        box,
		})
	}
	return out
}

func newVideoFindingResponse(view *domainvideo.VideoFrameFindingView) *VideoFindingResponse {
	if view == nil {
		return nil
	}
	categories := make([]VideoFindingCategoryResponse, 0, len(view.Categories))
	for _, category := range view.Categories {
		categories = append(categories, VideoFindingCategoryResponse{
			Name:        category.Name,
			Probability: category.Probability,
		})
	}
	return &VideoFindingResponse{
		ID:           view.ID.String(),
		Confidential: view.Confidential,
		Probability:  view.Probability,
		Categories:   categories,
		Status:       view.Status,
		ErrorReason:  optionalNonEmptyString(view.ErrorReason),
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
