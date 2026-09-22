package frame

import (
	"context"

	"github.com/google/uuid"
)

type FrameWriteRepository interface {
	UpsertAll(ctx context.Context, frames []*Frame) error
	CountByJobID(ctx context.Context, jobID uuid.UUID) (int, error)
}
