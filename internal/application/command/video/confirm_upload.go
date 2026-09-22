package video

import (
	"context"

	domainjob "go-api/internal/domain/job"
	"go-api/internal/domain/port"
	domainvideo "go-api/internal/domain/video"
)

type ConfirmUploadCommand struct {
	StorageKey  string
	ContentType string
	SizeBytes   int64
}

type ConfirmUploadHandler struct {
	videoRepo domainvideo.VideoWriteRepository
	jobRepo   domainjob.JobWriteRepository
	outbox    port.OutboxRepository
	maxSize   int64
}

func NewConfirmUploadHandler(
	videoRepo domainvideo.VideoWriteRepository,
	jobRepo domainjob.JobWriteRepository,
	outbox port.OutboxRepository,
	maxSize int64,
) *ConfirmUploadHandler {
	return &ConfirmUploadHandler{
		videoRepo: videoRepo,
		jobRepo:   jobRepo,
		outbox:    outbox,
		maxSize:   maxSize,
	}
}

func (h *ConfirmUploadHandler) Handle(ctx context.Context, cmd ConfirmUploadCommand) error {
	if h.maxSize > 0 && cmd.SizeBytes > h.maxSize {
		return domainvideo.ErrVideoTooLarge
	}

	return h.videoRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		video, err := h.videoRepo.GetByStorageKey(txCtx, cmd.StorageKey)
		if err != nil {
			return err
		}
		if video == nil {
			return domainvideo.ErrVideoNotFound
		}

		if err := video.MarkUploaded(cmd.ContentType, cmd.SizeBytes); err != nil {
			return err
		}

		existingJob, err := h.jobRepo.GetByVideoID(txCtx, video.ID)
		if err != nil {
			return err
		}
		if existingJob == nil {
			if err := h.jobRepo.Save(txCtx, domainjob.NewJob(video.ID)); err != nil {
				return err
			}
		}

		if err := h.videoRepo.Update(txCtx, video); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, video.PullEvents())
	})
}
