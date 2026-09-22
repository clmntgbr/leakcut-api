package video

import (
	"context"
	"encoding/json"

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

	return h.classify.Handle(ctx, videocommand.ClassifyFramesCommand{VideoID: videoID})
}
