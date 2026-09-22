package video

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"os"
	"time"

	"go-api/internal/application/messaging"
	domainframe "go-api/internal/domain/frame"
	domainjob "go-api/internal/domain/job"
	"go-api/internal/domain/port"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
)

type ExtractFramesCommand struct {
	VideoID uuid.UUID
}

type ExtractFramesHandler struct {
	videoRepo domainvideo.VideoWriteRepository
	jobRepo   domainjob.JobWriteRepository
	frameRepo domainframe.FrameWriteRepository
	outbox    port.OutboxRepository
	storage   port.Storage
	extractor port.FrameExtractor
	timeout   time.Duration
}

func NewExtractFramesHandler(
	videoRepo domainvideo.VideoWriteRepository,
	jobRepo domainjob.JobWriteRepository,
	frameRepo domainframe.FrameWriteRepository,
	outbox port.OutboxRepository,
	storage port.Storage,
	extractor port.FrameExtractor,
	timeout time.Duration,
) *ExtractFramesHandler {
	return &ExtractFramesHandler{
		videoRepo: videoRepo,
		jobRepo:   jobRepo,
		frameRepo: frameRepo,
		outbox:    outbox,
		storage:   storage,
		extractor: extractor,
		timeout:   timeout,
	}
}

func (h *ExtractFramesHandler) Handle(ctx context.Context, cmd ExtractFramesCommand) error {
	if h.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.timeout)
		defer cancel()
	}
	video, job, err := h.load(ctx, cmd.VideoID)
	if err != nil {
		return err
	}
	if video.Status == domainvideo.StatusFramesReady {
		return nil
	}

	if err := h.markExtracting(ctx, video, job); err != nil {
		return err
	}

	frames, extractErr := h.extractAndStore(ctx, video, job)
	if extractErr != nil {
		if isNonRetryableExtract(extractErr) {
			_ = h.markFailed(ctx, video, job, publicReason(extractErr))
			return messaging.NonRetryable(extractErr)
		}
		return messaging.Retryable(extractErr)
	}

	if err := h.markReady(ctx, video, job, frames); err != nil {
		return messaging.Retryable(err)
	}
	return nil
}

func (h *ExtractFramesHandler) load(ctx context.Context, videoID uuid.UUID) (*domainvideo.Video, *domainjob.Job, error) {
	video, err := h.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		return nil, nil, messaging.Retryable(err)
	}
	if video == nil {
		return nil, nil, messaging.NonRetryable(domainvideo.ErrVideoNotFound)
	}

	job, err := h.jobRepo.GetByVideoID(ctx, video.ID)
	if err != nil {
		return nil, nil, messaging.Retryable(err)
	}
	if job == nil {
		return nil, nil, messaging.NonRetryable(errors.New("scan job not found"))
	}
	return video, job, nil
}

func (h *ExtractFramesHandler) markExtracting(ctx context.Context, video *domainvideo.Video, job *domainjob.Job) error {
	job.MarkExtracting()
	if err := video.MarkExtracting(job.ID, job.Status); err != nil {
		return messaging.NonRetryable(err)
	}

	err := h.videoRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.videoRepo.Update(txCtx, video); err != nil {
			return err
		}
		if err := h.jobRepo.Update(txCtx, job); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, video.PullEvents())
	})
	if err != nil {
		return messaging.Retryable(err)
	}
	return nil
}

func (h *ExtractFramesHandler) extractAndStore(
	ctx context.Context,
	video *domainvideo.Video,
	job *domainjob.Job,
) ([]domainvideo.ExtractedFramePayload, error) {
	tmp, err := os.CreateTemp("", "video-extract-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	reader, err := h.storage.Get(ctx, video.StorageKey)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(tmp, reader); err != nil {
		_ = reader.Close()
		return nil, err
	}
	_ = reader.Close()
	if err := tmp.Close(); err != nil {
		return nil, err
	}

	if err := h.storeThumbnail(ctx, video, tmp.Name()); err != nil {
		log.Printf("failed to store thumbnail for video %s: %v", video.ID, err)
	}

	extracted, err := h.extractor.ExtractFrames(ctx, tmp.Name(), port.FrameSelectionParams{
		AnalysisFPS:        job.AnalysisFPS,
		DiffThreshold:      job.DiffThreshold,
		MaxIntervalSeconds: job.MaxIntervalSeconds,
	})
	if err != nil {
		return nil, messaging.NonRetryable(err)
	}

	payloads := make([]domainvideo.ExtractedFramePayload, 0, len(extracted))
	rows := make([]*domainframe.Frame, 0, len(extracted))
	for _, item := range extracted {
		storageKey := domainvideo.NewFrameStorageKey(video.ID, item.Index)
		if err := h.storage.Put(ctx, storageKey, bytes.NewReader(item.Data), int64(len(item.Data)), "image/png"); err != nil {
			return nil, err
		}
		rows = append(rows, domainframe.NewFrame(
			job.ID,
			item.Index,
			item.TimestampMs,
			storageKey,
			item.SelectionReason,
			item.DiffScore,
		))
		payloads = append(payloads, domainvideo.ExtractedFramePayload{
			Index:           item.Index,
			TimestampMs:     item.TimestampMs,
			StorageKey:      storageKey,
			SelectionReason: item.SelectionReason,
			DiffScore:       item.DiffScore,
		})
	}

	if err := h.frameRepo.UpsertAll(ctx, rows); err != nil {
		return nil, err
	}
	return payloads, nil
}

func (h *ExtractFramesHandler) storeThumbnail(ctx context.Context, video *domainvideo.Video, videoPath string) error {
	data, err := h.extractor.ExtractThumbnail(ctx, videoPath)
	if err != nil {
		return err
	}

	key := domainvideo.NewThumbnailStorageKey(video.ID)
	if err := h.storage.Put(ctx, key, bytes.NewReader(data), int64(len(data)), "image/jpeg"); err != nil {
		return err
	}

	video.SetThumbnailKey(key)
	return h.videoRepo.UpdateThumbnailKey(ctx, video.ID, key)
}

func (h *ExtractFramesHandler) markReady(
	ctx context.Context,
	video *domainvideo.Video,
	job *domainjob.Job,
	frames []domainvideo.ExtractedFramePayload,
) error {
	job.MarkFramesReady()
	if err := video.MarkFramesReady(job.ID, job.Status, frames); err != nil {
		return err
	}

	return h.videoRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.videoRepo.Update(txCtx, video); err != nil {
			return err
		}
		if err := h.jobRepo.Update(txCtx, job); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, video.PullEvents())
	})
}

func (h *ExtractFramesHandler) markFailed(
	ctx context.Context,
	video *domainvideo.Video,
	job *domainjob.Job,
	reason string,
) error {
	job.MarkFailed(reason)
	if err := video.MarkExtractionFailed(job.ID, job.Status, reason); err != nil {
		return err
	}

	return h.videoRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.videoRepo.Update(txCtx, video); err != nil {
			return err
		}
		if err := h.jobRepo.Update(txCtx, job); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, video.PullEvents())
	})
}

func publicReason(err error) string {
	if err == nil {
		return "frame extraction failed"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "frame extraction timed out"
	}
	return "unreadable video"
}

func isNonRetryableExtract(err error) bool {
	var nonRetryable *messaging.NonRetryableError
	return errors.As(err, &nonRetryable)
}
