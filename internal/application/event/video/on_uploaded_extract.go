package video

import (
	"context"
	"encoding/json"
	"log"

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

	log.Printf("frame worker received event type=%s videoId=%s", evt.EventType(), evt.VideoID)
	log.Printf("frame worker processing videoId=%s", evt.VideoID)
	if err := h.extract.Handle(ctx, videocommand.ExtractFramesCommand{VideoID: videoID}); err != nil {
		log.Printf("frame worker failed videoId=%s: %v", evt.VideoID, err)
		return err
	}
	log.Printf("frame worker finished videoId=%s", evt.VideoID)
	return nil
}
