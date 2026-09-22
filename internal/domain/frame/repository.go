package frame

import (
	"context"

	"github.com/google/uuid"
)

type FrameWriteRepository interface {
	UpsertAll(ctx context.Context, frames []*Frame) error
	CountByScanJobID(ctx context.Context, scanJobID uuid.UUID) (int, error)
}
