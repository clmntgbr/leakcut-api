package write

import (
	"time"

	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
)

type VideoModel struct {
	ID               uuid.UUID  `gorm:"column:id;primaryKey"`
	UserID           *uuid.UUID `gorm:"column:user_id"`
	OriginalFilename string     `gorm:"column:original_filename"`
	StorageKey       string     `gorm:"column:storage_key"`
	ThumbnailKey     string     `gorm:"column:thumbnail_key"`
	SizeBytes        int64      `gorm:"column:size_bytes"`
	ContentType      string     `gorm:"column:content_type"`
	Status           string     `gorm:"column:status"`
	Source           string     `gorm:"column:source"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}

func (VideoModel) TableName() string {
	return "videos"
}

func videoModelFromDomain(v *domainvideo.Video) *VideoModel {
	return &VideoModel{
		ID:               v.ID,
		UserID:           v.UserID,
		OriginalFilename: v.OriginalFilename,
		StorageKey:       v.StorageKey,
		ThumbnailKey:     v.ThumbnailKey,
		SizeBytes:        v.SizeBytes,
		ContentType:      v.ContentType,
		Status:           v.Status,
		Source:           v.Source,
		CreatedAt:        v.CreatedAt,
		UpdatedAt:        v.UpdatedAt,
	}
}

func videoDomainFromModel(m *VideoModel) *domainvideo.Video {
	return &domainvideo.Video{
		ID:               m.ID,
		UserID:           m.UserID,
		OriginalFilename: m.OriginalFilename,
		StorageKey:       m.StorageKey,
		ThumbnailKey:     m.ThumbnailKey,
		SizeBytes:        m.SizeBytes,
		ContentType:      m.ContentType,
		Status:           m.Status,
		Source:           m.Source,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}
