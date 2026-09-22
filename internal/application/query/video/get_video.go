package video

import (
	"context"
	"errors"
	"log"
	"time"

	"go-api/internal/domain/port"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
)

const thumbnailURLTTL = time.Hour

type GetVideoByIDQuery struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

type GetVideoByIDHandler struct {
	readRepo domainvideo.VideoReadRepository
	storage  port.Storage
}

func NewGetVideoByIDHandler(readRepo domainvideo.VideoReadRepository, storage port.Storage) *GetVideoByIDHandler {
	return &GetVideoByIDHandler{readRepo: readRepo, storage: storage}
}

func (h *GetVideoByIDHandler) Handle(ctx context.Context, q GetVideoByIDQuery) (*domainvideo.VideoView, error) {
	view, err := h.readRepo.FindByID(ctx, q.ID, q.UserID)
	if err != nil {
		return nil, errors.New("failed to get video")
	}
	if view == nil {
		return nil, domainvideo.ErrVideoNotFound
	}
	view.ThumbnailURL = presignThumbnailURL(ctx, h.storage, view.ID, view.ThumbnailKey)
	return view, nil
}

func presignThumbnailURL(ctx context.Context, storage port.Storage, videoID uuid.UUID, key string) string {
	if key == "" || storage == nil {
		return ""
	}
	url, err := storage.PresignedGetURL(ctx, key, thumbnailURLTTL)
	if err != nil {
		log.Printf("failed to presign thumbnail url for video %s: %v", videoID, err)
		return ""
	}
	return url
}
