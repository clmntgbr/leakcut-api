package handler

import (
	"context"

	cmdvideo "go-api/internal/application/command/video"
	queryvideo "go-api/internal/application/query/video"
	domainvideo "go-api/internal/domain/video"
)

type videoRequestUploadURLHandler interface {
	Handle(ctx context.Context, cmd cmdvideo.RequestUploadURLCommand) (*cmdvideo.RequestUploadURLResult, error)
}

type videoGetByIDHandler interface {
	Handle(ctx context.Context, q queryvideo.GetVideoByIDQuery) (*domainvideo.VideoView, error)
}

type videoListHandler interface {
	Handle(ctx context.Context, q queryvideo.ListVideosQuery) ([]domainvideo.VideoListView, int64, error)
}
