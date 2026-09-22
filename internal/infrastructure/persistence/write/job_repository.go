package write

import (
	"context"
	"errors"

	domainjob "go-api/internal/domain/job"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type jobWriteRepository struct {
	db *gorm.DB
}

func NewJobWriteRepository(db *gorm.DB) domainjob.JobWriteRepository {
	return &jobWriteRepository{db: db}
}

func (r *jobWriteRepository) Save(ctx context.Context, job *domainjob.Job) error {
	return DBWithContext(ctx, r.db).Create(jobModelFromDomain(job)).Error
}

func (r *jobWriteRepository) Update(ctx context.Context, job *domainjob.Job) error {
	return DBWithContext(ctx, r.db).Save(jobModelFromDomain(job)).Error
}

func (r *jobWriteRepository) GetByID(ctx context.Context, id uuid.UUID) (*domainjob.Job, error) {
	return r.get(ctx, "id = ?", id)
}

func (r *jobWriteRepository) GetByVideoID(ctx context.Context, videoID uuid.UUID) (*domainjob.Job, error) {
	return r.get(ctx, "video_id = ?", videoID)
}

func (r *jobWriteRepository) get(ctx context.Context, query string, args ...any) (*domainjob.Job, error) {
	var model JobModel
	err := DBWithContext(ctx, r.db).Where(query, args...).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return jobDomainFromModel(&model), nil
}
