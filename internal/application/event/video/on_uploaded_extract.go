package video

import (
	"context"
	"encoding/json"

	videocommand "go-api/internal/application/command/video"
	"go-api/internal/application/messaging"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
)

type ExtractFramesOnUploadedHandler struct {
	extract *videocommand.ExtractFramesHandler
}

func NewExtractFramesOnUploadedHandler(extract *videocommand.ExtractFramesHandler) *ExtractFramesOnUploadedHandler {
	return &ExtractFramesOnUploadedHandler{extract: extract}
}

func (h *ExtractFramesOnUploadedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainvideo.VideoUploaded
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}

	videoID, err := uuid.Parse(evt.VideoID)
	if err != nil {
		return messaging.NonRetryable(err)
	}

	return h.extract.Handle(ctx, videocommand.ExtractFramesCommand{VideoID: videoID})
}
