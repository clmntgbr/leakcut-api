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

type ClassifyFramesOnOCRCompletedHandler struct {
	classify *videocommand.ClassifyFramesHandler
}

func NewClassifyFramesOnOCRCompletedHandler(classify *videocommand.ClassifyFramesHandler) *ClassifyFramesOnOCRCompletedHandler {
	return &ClassifyFramesOnOCRCompletedHandler{classify: classify}
}

func (h *ClassifyFramesOnOCRCompletedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainvideo.VideoFramesOCRCompleted
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}

	videoID, err := uuid.Parse(evt.VideoID)
	if err != nil {
		return messaging.NonRetryable(err)
	}

	log.Printf("classify worker received event type=%s videoId=%s", evt.EventType(), evt.VideoID)
	log.Printf("classify worker processing videoId=%s", evt.VideoID)
	if err := h.classify.Handle(ctx, videocommand.ClassifyFramesCommand{VideoID: videoID}); err != nil {
		log.Printf("classify worker failed videoId=%s: %v", evt.VideoID, err)
		return err
	}
	log.Printf("classify worker finished videoId=%s", evt.VideoID)
	return nil
}
