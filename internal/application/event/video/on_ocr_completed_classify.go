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

// ClassifyFramesOnOCRCompletedHandler finalizes classify when OCR finishes after
// per-frame classify messages (covers the race where the last OCR row lands last).
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

	log.Printf("classify worker finalize on OCR completed videoId=%s", evt.VideoID)
	return h.classify.Finalize(ctx, videocommand.ClassifyFrameCommand{VideoID: videoID})
}
