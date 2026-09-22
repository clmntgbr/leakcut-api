package ocrresult

import (
	"context"

	"github.com/google/uuid"
)

type ResultWriteRepository interface {
	UpsertAll(ctx context.Context, results []*Result) error
	ListByFrameIDs(ctx context.Context, frameIDs []uuid.UUID) ([]*Result, error)
}
