package video

import (
	"context"
	"time"

	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
)

type VideoWriteRepository interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	Save(ctx context.Context, video *Video) error
	Update(ctx context.Context, video *Video) error
	GetByID(ctx context.Context, id uuid.UUID) (*Video, error)
	GetByStorageKey(ctx context.Context, storageKey string) (*Video, error)
	UpdateThumbnailKey(ctx context.Context, id uuid.UUID, thumbnailKey string) error
	ListExpiredPending(ctx context.Context, cutoff time.Time) ([]*Video, error)
}

type VideoReadRepository interface {
	FindByID(ctx context.Context, id, userID uuid.UUID) (*VideoView, error)
	List(ctx context.Context, userID uuid.UUID, query paginate.PaginateQuery) ([]VideoListView, int64, error)
}

type JobView struct {
	ID                 uuid.UUID
	Type               string
	Status             string
	FrameCount         int
	ExpectedFrameCount int
	OCRCompletedCount  int
	FailureReason      string
}

type VideoView struct {
	ID                 uuid.UUID
	OriginalFilename   string
	StorageKey         string
	ThumbnailKey       string
	ThumbnailURL       string
	SizeBytes          int64
	ContentType        string
	Status             string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	JobID              *uuid.UUID
	JobStatus          string
	FrameCount         int
	ExpectedFrameCount int
	OCRCompletedCount  int
	FailureReason      string
	Jobs               []JobView
}

type VideoListView struct {
	ID               uuid.UUID
	OriginalFilename string
	ThumbnailKey     string
	ThumbnailURL     string
	Status           string
	CreatedAt        time.Time
}
