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

type SegmentVideoOnUploadedHandler struct {
	segment *videocommand.SegmentVideoHandler
}

func NewSegmentVideoOnUploadedHandler(segment *videocommand.SegmentVideoHandler) *SegmentVideoOnUploadedHandler {
	return &SegmentVideoOnUploadedHandler{segment: segment}
}

func (h *SegmentVideoOnUploadedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainvideo.VideoUploaded
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	videoID, err := uuid.Parse(evt.VideoID)
	if err != nil {
		return messaging.NonRetryable(err)
	}

	log.Printf("segment worker received event type=%s videoId=%s", evt.EventType(), evt.VideoID)
	if err := h.segment.Handle(ctx, videocommand.SegmentVideoCommand{VideoID: videoID}); err != nil {
		log.Printf("segment worker failed videoId=%s: %v", evt.VideoID, err)
		return err
	}
	log.Printf("segment worker finished videoId=%s", evt.VideoID)
	return nil
}
