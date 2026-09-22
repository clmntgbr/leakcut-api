package video

import (
	"context"
	"encoding/json"
	"time"

	"go-api/internal/application/messaging"
	"go-api/internal/application/realtime"
	"go-api/internal/domain/port"
	domainvideo "go-api/internal/domain/video"
)

type PublishRealtimeHandler struct {
	publisher *realtime.Publisher
}

func NewPublishRealtimeHandler(realtimePublisher port.RealtimePublisher) *PublishRealtimeHandler {
	return &PublishRealtimeHandler{
		publisher: realtime.NewPublisher(realtimePublisher),
	}
}

type videoRealtimePayload struct {
	VideoID          string    `json:"videoId"`
	OriginalFilename string    `json:"originalFilename,omitempty"`
	Status           string    `json:"status"`
	OccurredAt       time.Time `json:"occurredAt"`
}

type jobRealtimePayload struct {
	ID            string    `json:"id"`
	VideoID       string    `json:"videoId"`
	Status        string    `json:"status"`
	VideoStatus   string    `json:"videoStatus"`
	FrameCount    int       `json:"frameCount"`
	FailureReason string    `json:"failureReason,omitempty"`
	OccurredAt    time.Time `json:"occurredAt"`
}

func (h *PublishRealtimeHandler) OnCreated(ctx context.Context, payload []byte) error {
	evt, err := decodeVideoEvent[domainvideo.VideoCreated](payload)
	if err != nil {
		return err
	}
	return h.publishToOwner(ctx, realtime.EntityVideo, realtime.ActionCreated, evt.UserID, videoRealtimePayload{
		VideoID:          evt.VideoID,
		OriginalFilename: evt.Filename,
		Status:           evt.Status,
		OccurredAt:       evt.Timestamp,
	})
}

func (h *PublishRealtimeHandler) OnUploaded(ctx context.Context, payload []byte) error {
	evt, err := decodeVideoEvent[domainvideo.VideoUploaded](payload)
	if err != nil {
		return err
	}
	return h.publishToOwner(ctx, realtime.EntityVideo, realtime.ActionUploaded, evt.UserID, videoRealtimePayload{
		VideoID:    evt.VideoID,
		Status:     evt.Status,
		OccurredAt: evt.Timestamp,
	})
}

func (h *PublishRealtimeHandler) OnExtracting(ctx context.Context, payload []byte) error {
	evt, err := decodeVideoEvent[domainvideo.VideoExtracting](payload)
	if err != nil {
		return err
	}
	return h.publishToOwner(ctx, realtime.EntityJob, realtime.ActionUpdated, evt.UserID, jobRealtimePayload{
		ID:          evt.JobID,
		VideoID:     evt.VideoID,
		Status:      evt.JobStatus,
		VideoStatus: evt.Status,
		OccurredAt:  evt.Timestamp,
	})
}

func (h *PublishRealtimeHandler) OnFramesExtracted(ctx context.Context, payload []byte) error {
	evt, err := decodeVideoEvent[domainvideo.VideoFramesExtracted](payload)
	if err != nil {
		return err
	}
	return h.publishToOwner(ctx, realtime.EntityJob, realtime.ActionUpdated, evt.UserID, jobRealtimePayload{
		ID:          evt.JobID,
		VideoID:     evt.VideoID,
		Status:      evt.JobStatus,
		VideoStatus: evt.Status,
		FrameCount:  evt.FrameCount,
		OccurredAt:  evt.Timestamp,
	})
}

func (h *PublishRealtimeHandler) OnExtractionFailed(ctx context.Context, payload []byte) error {
	evt, err := decodeVideoEvent[domainvideo.VideoFrameExtractionFailed](payload)
	if err != nil {
		return err
	}
	return h.publishToOwner(ctx, realtime.EntityJob, realtime.ActionUpdated, evt.UserID, jobRealtimePayload{
		ID:            evt.JobID,
		VideoID:       evt.VideoID,
		Status:        evt.JobStatus,
		VideoStatus:   evt.Status,
		FailureReason: evt.Reason,
		OccurredAt:    evt.Timestamp,
	})
}

func (h *PublishRealtimeHandler) publishToOwner(ctx context.Context, entity, action, userID string, payload any) error {
	if userID == "" {
		return nil
	}
	return h.publisher.ToUser(ctx, entity, action, userID, payload)
}

func decodeVideoEvent[T any](payload []byte) (T, error) {
	var evt T
	if err := json.Unmarshal(payload, &evt); err != nil {
		return evt, messaging.NonRetryable(err)
	}
	return evt, nil
}
