package video

import (
	"context"
	"encoding/json"

	videocommand "go-api/internal/application/command/video"
	"go-api/internal/application/messaging"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
)

type PersistOCRFrameHandler struct {
	ocr *videocommand.OCRFramesHandler
}

func NewPersistOCRFrameHandler(ocr *videocommand.OCRFramesHandler) *PersistOCRFrameHandler {
	return &PersistOCRFrameHandler{ocr: ocr}
}

func (h *PersistOCRFrameHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainvideo.VideoOCRFrameCompleted
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}

	videoID, err := uuid.Parse(evt.VideoID)
	if err != nil {
		return messaging.NonRetryable(err)
	}

	return h.ocr.PersistFrame(ctx, videocommand.PersistOCRFrameCommand{
		VideoID: videoID,
		Results: evt.Results,
	})
}
