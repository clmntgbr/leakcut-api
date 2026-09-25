package ocr

import (
	"context"

	"github.com/google/uuid"
)

type WriteRepository interface {
	UpsertAll(ctx context.Context, results []*Result) error
	ListByFrameIDs(ctx context.Context, frameIDs []uuid.UUID) ([]*Result, error)
}
