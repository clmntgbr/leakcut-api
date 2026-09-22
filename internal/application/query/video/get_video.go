package video

import (
	"context"
	"fmt"
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
		return nil, fmt.Errorf("failed to get video: %w", err)
	}
	if view == nil {
		return nil, domainvideo.ErrVideoNotFound
	}
	view.ThumbnailURL = presignMediaURL(ctx, h.storage, view.ID, view.ThumbnailKey)
	view.VideoURL = presignMediaURL(ctx, h.storage, view.ID, view.StorageKey)

	frames, err := h.readRepo.ListFramesByVideoID(ctx, q.ID, q.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to list video frames: %w", err)
	}
	for i := range frames {
		frames[i].ImageURL = presignMediaURL(ctx, h.storage, view.ID, frames[i].StorageKey)
	}
	view.Frames = frames
	return view, nil
}

func presignMediaURL(ctx context.Context, storage port.Storage, videoID uuid.UUID, key string) string {
	if key == "" || storage == nil {
		return ""
	}
	url, err := storage.PresignedGetURL(ctx, key, thumbnailURLTTL)
	if err != nil {
		log.Printf("failed to presign media url for video %s: %v", videoID, err)
		return ""
	}
	return url
}
