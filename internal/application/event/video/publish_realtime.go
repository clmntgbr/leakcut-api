package video

import (
	"context"
	"encoding/json"

	"go-api/internal/application/messaging"
	domainvideo "go-api/internal/domain/video"
)

// PublishRealtimeHandler is the video counterpart of the user/scan Centrifugo
// publishers. It is wired on domain events so the path exists; it does not
// publish to the frontend yet.
type PublishRealtimeHandler struct{}

func NewPublishRealtimeHandler() *PublishRealtimeHandler {
	return &PublishRealtimeHandler{}
}

func (h *PublishRealtimeHandler) OnCreated(_ context.Context, payload []byte) error {
	return decodeVideoEvent[domainvideo.VideoCreated](payload)
}

func (h *PublishRealtimeHandler) OnUploaded(_ context.Context, payload []byte) error {
	return decodeVideoEvent[domainvideo.VideoUploaded](payload)
}

func (h *PublishRealtimeHandler) OnFramesExtracted(_ context.Context, payload []byte) error {
	return decodeVideoEvent[domainvideo.VideoFramesExtracted](payload)
}

func (h *PublishRealtimeHandler) OnExtractionFailed(_ context.Context, payload []byte) error {
	return decodeVideoEvent[domainvideo.VideoFrameExtractionFailed](payload)
}

func (h *PublishRealtimeHandler) OnUploadExpired(_ context.Context, payload []byte) error {
	return decodeVideoEvent[domainvideo.VideoUploadExpired](payload)
}

func decodeVideoEvent[T any](payload []byte) error {
	var evt T
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return nil
}
