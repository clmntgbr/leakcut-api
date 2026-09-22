package video

import (
	"context"
	"fmt"

	"go-api/internal/domain/paginate"
	"go-api/internal/domain/port"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
)

type ListVideosQuery struct {
	UserID uuid.UUID
	Query  paginate.PaginateQuery
}

type ListVideosHandler struct {
	readRepo domainvideo.VideoReadRepository
	storage  port.Storage
}

func NewListVideosHandler(readRepo domainvideo.VideoReadRepository, storage port.Storage) *ListVideosHandler {
	return &ListVideosHandler{readRepo: readRepo, storage: storage}
}

func (h *ListVideosHandler) Handle(ctx context.Context, q ListVideosQuery) ([]domainvideo.VideoListView, int64, error) {
	views, total, err := h.readRepo.List(ctx, q.UserID, q.Query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list videos: %w", err)
	}

	for i := range views {
		views[i].ThumbnailURL = presignThumbnailURL(ctx, h.storage, views[i].ID, views[i].ThumbnailKey)
	}
	return views, total, nil
}
