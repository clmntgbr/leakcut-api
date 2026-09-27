package segment

import (
	"context"

	"github.com/google/uuid"
)

type WriteRepository interface {
	SaveAll(ctx context.Context, segments []*Segment) error
	Update(ctx context.Context, segment *Segment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Segment, error)
	ListByVideoID(ctx context.Context, videoID uuid.UUID) ([]*Segment, error)
	GetByVideoIDAndIndex(ctx context.Context, videoID uuid.UUID, index int) (*Segment, error)
}
