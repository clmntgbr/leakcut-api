package video

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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

var errPersistFrame = errors.New("persist extracted frame")

type ExtractFramesCommand struct {
	VideoID uuid.UUID
}

type ExtractFramesHandler struct {
	videoRepo  domainvideo.VideoWriteRepository
	jobRepo    domainjob.JobWriteRepository
	frameRepo  domainframe.FrameWriteRepository
	outbox     port.OutboxRepository
	storage    port.Storage
	extractor  port.FrameExtractor
	timeout    time.Duration
	maxWidthPx int
}

func NewExtractFramesHandler(
	videoRepo domainvideo.VideoWriteRepository,
	jobRepo domainjob.JobWriteRepository,
	frameRepo domainframe.FrameWriteRepository,
	outbox port.OutboxRepository,
	storage port.Storage,
	extractor port.FrameExtractor,
	timeout time.Duration,
	maxWidthPx int,
) *ExtractFramesHandler {
	if maxWidthPx <= 0 {
		maxWidthPx = domainjob.DefaultFrameMaxWidthPx
	}
	return &ExtractFramesHandler{
		videoRepo:  videoRepo,
		jobRepo:    jobRepo,
		frameRepo:  frameRepo,
		outbox:     outbox,
		storage:    storage,
		extractor:  extractor,
		timeout:    timeout,
		maxWidthPx: maxWidthPx,
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
	if domainvideo.ExtractionAlreadyDone(video.Status) {
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

	job, err := h.jobRepo.GetByVideoIDAndType(ctx, video.ID, domainjob.TypeExtractFrames)
	if err != nil {
		return nil, nil, messaging.Retryable(err)
	}
	if job == nil {
		return nil, nil, messaging.NonRetryable(errors.New("extract frames job not found"))
	}
	return video, job, nil
}

func (h *ExtractFramesHandler) markExtracting(ctx context.Context, video *domainvideo.Video, job *domainjob.Job) error {
	job.MarkExtracting()
	if err := video.MarkExtracting(job.ID, job.Type, job.Status); err != nil {
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

	payloads := make([]domainvideo.ExtractedFramePayload, 0)
	err = h.extractor.ExtractFrames(ctx, tmp.Name(), port.FrameSelectionParams{
		AnalysisFPS:            job.AnalysisFPS,
		PHashDistanceThreshold: job.PHashDistanceThreshold,
		MaxIntervalSeconds:     job.MaxIntervalSeconds,
		MaxWidthPx:             h.maxWidthPx,
	}, func(item port.ExtractedFrame) error {
		storageKey := domainvideo.NewFrameStorageKey(video.ID, item.Index)
		if err := h.storage.Put(ctx, storageKey, bytes.NewReader(item.Data), int64(len(item.Data)), "image/png"); err != nil {
			return fmt.Errorf("%w: %w", errPersistFrame, err)
		}
		frame := domainframe.NewFrame(
			job.ID,
			item.Index,
			item.TimestampMs,
			storageKey,
			item.SelectionReason,
			item.PHashDistance,
		)
		payload, err := h.persistFrameAndRequestOCR(ctx, video, frame)
		if err != nil {
			return fmt.Errorf("%w: %w", errPersistFrame, err)
		}
		payloads = append(payloads, payload)
		return nil
	})
	if err != nil {
		if errors.Is(err, errPersistFrame) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return payloads, err
		}
		return payloads, messaging.NonRetryable(err)
	}
	return payloads, nil
}

func (h *ExtractFramesHandler) persistFrameAndRequestOCR(
	ctx context.Context,
	video *domainvideo.Video,
	frame *domainframe.Frame,
) (domainvideo.ExtractedFramePayload, error) {
	var payload domainvideo.ExtractedFramePayload
	err := h.videoRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.frameRepo.UpsertAll(txCtx, []*domainframe.Frame{frame}); err != nil {
			return err
		}
		stored, err := h.frameRepo.ListByJobID(txCtx, frame.JobID)
		if err != nil {
			return err
		}
		payload = domainvideo.ExtractedFramePayload{
			ID:              frame.ID.String(),
			Index:           frame.Index,
			TimestampMs:     frame.TimestampMs,
			StorageKey:      frame.StorageKey,
			SelectionReason: frame.SelectionReason,
			PHashDistance:   frame.PHashDistance,
		}
		for _, row := range stored {
			if row.Index == frame.Index {
				payload.ID = row.ID.String()
				payload.StorageKey = row.StorageKey
				break
			}
		}
		video.RequestFrameOCR(payload)
		return h.outbox.StoreEvents(txCtx, video.PullEvents())
	})
	return payload, err
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
	job.MarkFramesReady(len(frames))
	if err := video.MarkFramesReady(job.ID, job.Type, job.Status, frames); err != nil {
		return err
	}

	return h.videoRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.videoRepo.Update(txCtx, video); err != nil {
			return err
		}
		if err := h.jobRepo.Update(txCtx, job); err != nil {
			return err
		}
		ocrJob, err := h.jobRepo.GetByVideoIDAndType(txCtx, video.ID, domainjob.TypeOCR)
		if err != nil {
			return err
		}
		if ocrJob == nil {
			if err := h.jobRepo.Save(txCtx, domainjob.NewOCRJob(video.ID)); err != nil {
				return err
			}
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
	if err := video.MarkExtractionFailed(job.ID, job.Type, job.Status, reason); err != nil {
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
