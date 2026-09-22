package handler

import (
	"context"

	cmdvideo "go-api/internal/application/command/video"
)

type storageConfirmUploadHandler interface {
	Handle(ctx context.Context, cmd cmdvideo.ConfirmUploadCommand) error
}
