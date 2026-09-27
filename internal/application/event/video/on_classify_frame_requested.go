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

type ClassifyFrameRequestedHandler struct {
	classify *videocommand.ClassifyFramesHandler
}

func NewClassifyFrameRequestedHandler(classify *videocommand.ClassifyFramesHandler) *ClassifyFrameRequestedHandler {
	return &ClassifyFrameRequestedHandler{classify: classify}
}

func (h *ClassifyFrameRequestedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainvideo.VideoClassifyFrameRequested
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}

	videoID, err := uuid.Parse(evt.VideoID)
	if err != nil {
		return messaging.NonRetryable(err)
	}
	frameID, err := uuid.Parse(evt.FrameID)
	if err != nil {
		return messaging.NonRetryable(err)
	}

	log.Printf("classify worker received event type=%s videoId=%s frameId=%s", evt.EventType(), evt.VideoID, evt.FrameID)
	if err := h.classify.HandleFrame(ctx, videocommand.ClassifyFrameCommand{
		VideoID: videoID,
		FrameID: frameID,
	}); err != nil {
		log.Printf("classify worker failed videoId=%s frameId=%s: %v", evt.VideoID, evt.FrameID, err)
		return err
	}
	log.Printf("classify worker finished videoId=%s frameId=%s", evt.VideoID, evt.FrameID)
	return nil
}
