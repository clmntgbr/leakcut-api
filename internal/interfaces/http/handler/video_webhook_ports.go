package handler

import (
	"context"

	cmdvideo "go-api/internal/application/command/video"
)

type videoRequestIngestHandler interface {
	Handle(ctx context.Context, cmd cmdvideo.RequestIngestCommand) (*cmdvideo.RequestIngestResult, error)
}
