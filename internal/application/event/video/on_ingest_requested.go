package video

import (
	"context"
	"encoding/json"

	videocommand "go-api/internal/application/command/video"
	"go-api/internal/application/messaging"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
)

type IngestRemoteOnRequestedHandler struct {
	ingest *videocommand.IngestRemoteHandler
}

func NewIngestRemoteOnRequestedHandler(ingest *videocommand.IngestRemoteHandler) *IngestRemoteOnRequestedHandler {
	return &IngestRemoteOnRequestedHandler{ingest: ingest}
}

func (h *IngestRemoteOnRequestedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainvideo.VideoIngestRequested
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}

	videoID, err := uuid.Parse(evt.VideoID)
	if err != nil {
		return messaging.NonRetryable(err)
	}

	return h.ingest.Handle(ctx, videocommand.IngestRemoteCommand{
		VideoID:   videoID,
		RemoteURL: evt.RemoteURL,
	})
}
