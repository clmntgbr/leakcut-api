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

type ExtractFramesOnSegmentReadyHandler struct {
	extract *videocommand.ExtractFramesHandler
}

func NewExtractFramesOnSegmentReadyHandler(extract *videocommand.ExtractFramesHandler) *ExtractFramesOnSegmentReadyHandler {
	return &ExtractFramesOnSegmentReadyHandler{extract: extract}
}

func (h *ExtractFramesOnSegmentReadyHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainvideo.VideoSegmentReady
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	videoID, err := uuid.Parse(evt.VideoID)
	if err != nil {
		return messaging.NonRetryable(err)
	}
	segmentID, err := uuid.Parse(evt.SegmentID)
	if err != nil {
		return messaging.NonRetryable(err)
	}

	log.Printf(
		"frame worker received event type=%s videoId=%s segmentId=%s index=%d",
		evt.EventType(), evt.VideoID, evt.SegmentID, evt.SegmentIndex,
	)
	if err := h.extract.Handle(ctx, videocommand.ExtractFramesCommand{
		VideoID:   videoID,
		SegmentID: segmentID,
	}); err != nil {
		log.Printf("frame worker failed videoId=%s segmentId=%s: %v", evt.VideoID, evt.SegmentID, err)
		return err
	}
	log.Printf("frame worker finished videoId=%s segmentId=%s", evt.VideoID, evt.SegmentID)
	return nil
}
