package write

import (
	"context"
	"errors"

	domainscanjob "go-api/internal/domain/scanjob"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type scanJobWriteRepository struct {
	db *gorm.DB
}

func NewScanJobWriteRepository(db *gorm.DB) domainscanjob.ScanJobWriteRepository {
	return &scanJobWriteRepository{db: db}
}

func (r *scanJobWriteRepository) Save(ctx context.Context, job *domainscanjob.ScanJob) error {
	return DBWithContext(ctx, r.db).Create(scanJobModelFromDomain(job)).Error
}

func (r *scanJobWriteRepository) Update(ctx context.Context, job *domainscanjob.ScanJob) error {
	return DBWithContext(ctx, r.db).Save(scanJobModelFromDomain(job)).Error
}

func (r *scanJobWriteRepository) GetByID(ctx context.Context, id uuid.UUID) (*domainscanjob.ScanJob, error) {
	return r.get(ctx, "id = ?", id)
}

func (r *scanJobWriteRepository) GetByVideoID(ctx context.Context, videoID uuid.UUID) (*domainscanjob.ScanJob, error) {
	return r.get(ctx, "video_id = ?", videoID)
}

func (r *scanJobWriteRepository) get(ctx context.Context, query string, args ...any) (*domainscanjob.ScanJob, error) {
	var model ScanJobModel
	err := DBWithContext(ctx, r.db).Where(query, args...).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return scanJobDomainFromModel(&model), nil
}
