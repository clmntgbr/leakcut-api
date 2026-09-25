package classification

import (
	"context"

	"github.com/google/uuid"
)

type WriteRepository interface {
	UpsertAll(ctx context.Context, items []*Classification) error
	ListByFrameIDs(ctx context.Context, frameIDs []uuid.UUID) ([]*Classification, error)
}
