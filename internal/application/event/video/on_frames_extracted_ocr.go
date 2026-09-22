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

	return h.ocr.Start(ctx, videocommand.StartOCRFramesCommand{VideoID: videoID})
}

type PersistOCRBatchHandler struct {
	ocr *videocommand.OCRFramesHandler
}

func NewPersistOCRBatchHandler(ocr *videocommand.OCRFramesHandler) *PersistOCRBatchHandler {
	return &PersistOCRBatchHandler{ocr: ocr}
}

func (h *PersistOCRBatchHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainvideo.VideoOCRBatchCompleted
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}

	videoID, err := uuid.Parse(evt.VideoID)
	if err != nil {
		return messaging.NonRetryable(err)
	}

	return h.ocr.PersistBatch(ctx, videocommand.PersistOCRBatchCommand{
		VideoID: videoID,
		Results: evt.Results,
	})
}
