package job

import (
	"context"

	"github.com/google/uuid"
)

type JobWriteRepository interface {
	Save(ctx context.Context, job *Job) error
	Update(ctx context.Context, job *Job) error
	GetByID(ctx context.Context, id uuid.UUID) (*Job, error)
	GetByVideoID(ctx context.Context, videoID uuid.UUID) (*Job, error)
}
