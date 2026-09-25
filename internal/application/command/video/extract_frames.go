package video

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"go-api/internal/application/messaging"
	domainframe "go-api/internal/domain/frame"
	domainjob "go-api/internal/domain/job"
	"go-api/internal/domain/port"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
)

var errPersistFrame = errors.New("persist extracted frame")

const frameContentType = "image/jpeg"

type ExtractFramesCommand struct {
	VideoID uuid.UUID
}

type ExtractFramesHandler struct {
	videoRepo         domainvideo.VideoWriteRepository
	jobRepo           domainjob.JobWriteRepository
	frameRepo         domainframe.FrameWriteRepository
	outbox            port.OutboxRepository
	storage           port.Storage
	extractor         port.FrameExtractor
	timeout           time.Duration
	maxWidthPx        int
	uploadConcurrency int
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
	uploadConcurrency int,
) *ExtractFramesHandler {
	if maxWidthPx <= 0 {
		maxWidthPx = domainjob.DefaultFrameMaxWidthPx
	}
	if uploadConcurrency <= 0 {
		uploadConcurrency = domainjob.DefaultFrameUploadConcurrency
	}
	return &ExtractFramesHandler{
		videoRepo:         videoRepo,
		jobRepo:           jobRepo,
		frameRepo:         frameRepo,
		outbox:            outbox,
		storage:           storage,
		extractor:         extractor,
		timeout:           timeout,
		maxWidthPx:        maxWidthPx,
		uploadConcurrency: uploadConcurrency,
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

	job, err := h.jobRepo.GetByVideoIDAndType(ctx, video.ID, domainjob.TypeFrame)
	if err != nil {
		return nil, nil, messaging.Retryable(err)
	}
	if job == nil {
		return nil, nil, messaging.NonRetryable(errors.New("frame job not found"))
	}
	return video, job, nil
}

func (h *ExtractFramesHandler) markExtracting(ctx context.Context, video *domainvideo.Video, job *domainjob.Job) error {
	job.MarkProcessing()
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

	pool := newFramePersistPool(ctx, h, video, h.uploadConcurrency)
	extractErr := h.extractor.ExtractFrames(ctx, tmp.Name(), port.FrameSelectionParams{
		AnalysisFPS:            job.AnalysisFPS,
		PHashDistanceThreshold: job.PHashDistanceThreshold,
		MaxIntervalSeconds:     job.MaxIntervalSeconds,
		MaxWidthPx:             h.maxWidthPx,
	}, pool.Submit)
	payloads, waitErr := pool.Wait()
	if extractErr != nil {
		if errors.Is(extractErr, errPersistFrame) || errors.Is(extractErr, context.DeadlineExceeded) || errors.Is(extractErr, context.Canceled) {
			return payloads, extractErr
		}
		if waitErr != nil {
			return payloads, waitErr
		}
		return payloads, messaging.NonRetryable(extractErr)
	}
	if waitErr != nil {
		if errors.Is(waitErr, errPersistFrame) || errors.Is(waitErr, context.DeadlineExceeded) || errors.Is(waitErr, context.Canceled) {
			return payloads, waitErr
		}
		return payloads, waitErr
	}
	return payloads, nil
}

type framePersistPool struct {
	ctx    context.Context
	h      *ExtractFramesHandler
	video  *domainvideo.Video
	sem    chan struct{}
	wg     sync.WaitGroup
	mu     sync.Mutex
	err    error
	frames []domainvideo.ExtractedFramePayload
}

func newFramePersistPool(
	ctx context.Context,
	h *ExtractFramesHandler,
	video *domainvideo.Video,
	concurrency int,
) *framePersistPool {
	if concurrency <= 0 {
		concurrency = domainjob.DefaultFrameUploadConcurrency
	}
	return &framePersistPool{
		ctx:    ctx,
		h:      h,
		video:  video,
		sem:    make(chan struct{}, concurrency),
		frames: make([]domainvideo.ExtractedFramePayload, 0),
	}
}

func (p *framePersistPool) Submit(item port.ExtractedFrame) error {
	if err := p.getErr(); err != nil {
		return err
	}

	p.wg.Add(1)
	go func(item port.ExtractedFrame) {
		defer p.wg.Done()

		select {
		case p.sem <- struct{}{}:
		case <-p.ctx.Done():
			p.setErr(fmt.Errorf("%w: %w", errPersistFrame, p.ctx.Err()))
			return
		}
		defer func() { <-p.sem }()

		if err := p.getErr(); err != nil {
			return
		}

		storageKey := domainvideo.NewFrameStorageKey(p.video.ID, item.Index)
		if err := p.h.storage.Put(
			p.ctx,
			storageKey,
			bytes.NewReader(item.Data),
			int64(len(item.Data)),
			frameContentType,
		); err != nil {
			p.setErr(fmt.Errorf("%w: %w", errPersistFrame, err))
			return
		}

		frame := domainframe.NewFrame(
			p.video.ID,
			item.Index,
			item.TimestampMs,
			storageKey,
			item.SelectionReason,
			item.PHashDistance,
		)
		if err := p.h.frameRepo.UpsertAll(p.ctx, []*domainframe.Frame{frame}); err != nil {
			p.setErr(fmt.Errorf("%w: %w", errPersistFrame, err))
			return
		}
		payload := domainvideo.ExtractedFramePayload{
			ID:              frame.ID.String(),
			Index:           frame.Index,
			TimestampMs:     frame.TimestampMs,
			StorageKey:      frame.StorageKey,
			SelectionReason: frame.SelectionReason,
			PHashDistance:   frame.PHashDistance,
		}
		p.mu.Lock()
		p.frames = append(p.frames, payload)
		p.mu.Unlock()
	}(item)

	return nil
}

func (p *framePersistPool) Wait() ([]domainvideo.ExtractedFramePayload, error) {
	p.wg.Wait()
	p.mu.Lock()
	defer p.mu.Unlock()
	sort.Slice(p.frames, func(i, j int) bool {
		return p.frames[i].Index < p.frames[j].Index
	})
	return p.frames, p.err
}

func (p *framePersistPool) getErr() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

func (p *framePersistPool) setErr(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err == nil {
		p.err = err
	}
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
	job.SetExpectedFrameCount(len(frames))
	job.MarkSuccess()
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
