package frame

import (
	"context"

	"github.com/google/uuid"
)

type FrameWriteRepository interface {
	UpsertAll(ctx context.Context, frames []*Frame) error
	ListByVideoID(ctx context.Context, videoID uuid.UUID) ([]*Frame, error)
	CountByVideoID(ctx context.Context, videoID uuid.UUID) (int, error)
}
