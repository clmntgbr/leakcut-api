package scanjob

import (
	"context"

	"github.com/google/uuid"
)

type ScanJobWriteRepository interface {
	Save(ctx context.Context, job *ScanJob) error
	Update(ctx context.Context, job *ScanJob) error
	GetByID(ctx context.Context, id uuid.UUID) (*ScanJob, error)
	GetByVideoID(ctx context.Context, videoID uuid.UUID) (*ScanJob, error)
}
