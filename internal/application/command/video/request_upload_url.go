package video

import (
	"context"
	"errors"
	"time"

	"go-api/internal/domain/port"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
)

const DefaultUploadURLTTL = 15 * time.Minute

type RequestUploadURLCommand struct {
	UserID      uuid.UUID
	Filename    string
	ContentType string
	SizeBytes   int64
}

type RequestUploadURLResult struct {
	VideoID   string
	UploadURL string
	ExpiresAt time.Time
}

type RequestUploadURLHandler struct {
	repo    domainvideo.VideoWriteRepository
	outbox  port.OutboxRepository
	storage port.Storage
	ttl     time.Duration
}

func NewRequestUploadURLHandler(
	repo domainvideo.VideoWriteRepository,
	outbox port.OutboxRepository,
	storage port.Storage,
	ttl time.Duration,
) *RequestUploadURLHandler {
	if ttl <= 0 {
		ttl = DefaultUploadURLTTL
	}
	return &RequestUploadURLHandler{repo: repo, outbox: outbox, storage: storage, ttl: ttl}
}

func (h *RequestUploadURLHandler) Handle(ctx context.Context, cmd RequestUploadURLCommand) (*RequestUploadURLResult, error) {
	video, err := domainvideo.NewPresignedVideo(cmd.UserID, cmd.Filename, cmd.ContentType, cmd.SizeBytes)
	if err != nil {
		return nil, err
	}

	url, err := h.storage.PresignedPutURL(ctx, video.StorageKey, h.ttl)
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
		return nil, errors.New("failed to create video upload")
	}

	return &RequestUploadURLResult{
		VideoID:   video.ID.String(),
		UploadURL: url,
		ExpiresAt: video.CreatedAt.Add(h.ttl),
	}, nil
}
