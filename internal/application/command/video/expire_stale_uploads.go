package video

import (
	"context"
	"log"
	"time"

	"go-api/internal/domain/port"
	domainvideo "go-api/internal/domain/video"
)

type ExpireStaleUploadsHandler struct {
	repo   domainvideo.VideoWriteRepository
	outbox port.OutboxRepository
	ttl    time.Duration
}

func NewExpireStaleUploadsHandler(
	repo domainvideo.VideoWriteRepository,
	outbox port.OutboxRepository,
	ttl time.Duration,
) *ExpireStaleUploadsHandler {
	if ttl <= 0 {
		ttl = DefaultUploadURLTTL
	}
	return &ExpireStaleUploadsHandler{repo: repo, outbox: outbox, ttl: ttl}
}

func (h *ExpireStaleUploadsHandler) Handle(ctx context.Context) error {
	now := time.Now().UTC()
	videos, err := h.repo.ListExpiredPending(ctx, now.Add(-h.ttl))
	if err != nil {
		return err
	}

	for _, video := range videos {
		if err := video.MarkUploadExpired(); err != nil {
			continue
		}
		err := h.repo.WithTransaction(ctx, func(txCtx context.Context) error {
			if err := h.repo.Update(txCtx, video); err != nil {
				return err
			}
			return h.outbox.StoreEvents(txCtx, video.PullEvents())
		})
		if err != nil {
			log.Printf("failed to expire video %s: %v", video.ID, err)
		}
	}
	return nil
}
