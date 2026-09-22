package finding

import (
	"context"

	"github.com/google/uuid"
)

type FindingWriteRepository interface {
	UpsertAll(ctx context.Context, findings []*Finding) error
	ListByFrameIDs(ctx context.Context, frameIDs []uuid.UUID) ([]*Finding, error)
}
