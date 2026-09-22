package video

import (
	"context"
	"errors"

	"go-api/internal/domain/port"
	domainvideo "go-api/internal/domain/video"
)

type RequestIngestCommand struct {
	RemoteURL   string
	Filename    string
	ContentType string
	SizeBytes   int64
}

type RequestIngestResult struct {
	VideoID string
}

type RequestIngestHandler struct {
	repo   domainvideo.VideoWriteRepository
	outbox port.OutboxRepository
}

func NewRequestIngestHandler(
	repo domainvideo.VideoWriteRepository,
	outbox port.OutboxRepository,
) *RequestIngestHandler {
	return &RequestIngestHandler{repo: repo, outbox: outbox}
}

func (h *RequestIngestHandler) Handle(ctx context.Context, cmd RequestIngestCommand) (*RequestIngestResult, error) {
	video, err := domainvideo.NewWebhookVideo(cmd.Filename, cmd.ContentType, cmd.RemoteURL, cmd.SizeBytes)
	if err != nil {
		return nil, err
	}

	err = h.repo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.repo.Save(txCtx, video); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, video.PullEvents())
	})
	if err != nil {
		return nil, errors.New("failed to create video ingest")
	}

	return &RequestIngestResult{VideoID: video.ID.String()}, nil
}
