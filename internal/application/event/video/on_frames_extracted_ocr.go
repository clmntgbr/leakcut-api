package video

import (
	"context"
	"encoding/json"

	videocommand "go-api/internal/application/command/video"
	"go-api/internal/application/messaging"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
)

type OCRFramesOnExtractedHandler struct {
	ocr *videocommand.OCRFramesHandler
}

func NewOCRFramesOnExtractedHandler(ocr *videocommand.OCRFramesHandler) *OCRFramesOnExtractedHandler {
	return &OCRFramesOnExtractedHandler{ocr: ocr}
}

func (h *OCRFramesOnExtractedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainvideo.VideoFramesExtracted
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}

	videoID, err := uuid.Parse(evt.VideoID)
	if err != nil {
		return messaging.NonRetryable(err)
	}

	return h.ocr.Handle(ctx, videocommand.OCRFramesCommand{VideoID: videoID})
}
