package video

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"go-api/internal/application/messaging"
	domainframe "go-api/internal/domain/frame"
	domainjob "go-api/internal/domain/job"
	"go-api/internal/domain/port"
	domainsegment "go-api/internal/domain/segment"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
)

var errPersistFrame = errors.New("persist extracted frame")

const frameContentType = "image/jpeg"

type ExtractFramesCommand struct {
	VideoID   uuid.UUID
	SegmentID uuid.UUID
}

type ExtractFramesHandler struct {
	videoRepo         domainvideo.VideoWriteRepository
	jobRepo           domainjob.JobWriteRepository
	segmentRepo       domainsegment.WriteRepository
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
	segmentRepo domainsegment.WriteRepository,
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
		segmentRepo:       segmentRepo,
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

	video, job, segment, err := h.load(ctx, cmd.VideoID, cmd.SegmentID)
	if err != nil {
		return err
	}
	if job.Status == domainjob.StatusSuccess {
		return nil
	}
	if video.Status == domainvideo.StatusExtractionFailed {
		return nil
	}
	if segment.Status == domainsegment.StatusSuccess {
		return h.maybeFinalize(ctx, video, job)
	}

	segment.MarkProcessing()
	if err := h.segmentRepo.Update(ctx, segment); err != nil {
		return messaging.Retryable(err)
	}

	frames, extractErr := h.extractAndStore(ctx, video, job, segment)
	if extractErr != nil {
		if isNonRetryableExtract(extractErr) {
			_ = h.failSegment(ctx, video, job, segment, publicReason(extractErr))
			return messaging.NonRetryable(extractErr)
		}
		return messaging.Retryable(extractErr)
	}

	if err := h.completeSegment(ctx, video, job, segment, frames); err != nil {
		return messaging.Retryable(err)
	}
	return nil
}

func (h *ExtractFramesHandler) load(
	ctx context.Context,
	videoID, segmentID uuid.UUID,
) (*domainvideo.Video, *domainjob.Job, *domainsegment.Segment, error) {
	video, err := h.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		return nil, nil, nil, messaging.Retryable(err)
	}
	if video == nil {
		return nil, nil, nil, messaging.NonRetryable(domainvideo.ErrVideoNotFound)
	}

	job, err := h.jobRepo.GetByVideoIDAndType(ctx, video.ID, domainjob.TypeFrame)
	if err != nil {
		return nil, nil, nil, messaging.Retryable(err)
	}
	if job == nil {
		return nil, nil, nil, messaging.NonRetryable(errors.New("frame job not found"))
	}

	segment, err := h.segmentRepo.GetByID(ctx, segmentID)
	if err != nil {
		return nil, nil, nil, messaging.Retryable(err)
	}
	if segment == nil || segment.VideoID != video.ID {
		return nil, nil, nil, messaging.NonRetryable(errors.New("segment not found"))
	}
	return video, job, segment, nil
}

func (h *ExtractFramesHandler) extractAndStore(
	ctx context.Context,
	video *domainvideo.Video,
	job *domainjob.Job,
	segment *domainsegment.Segment,
) ([]domainvideo.ExtractedFramePayload, error) {
	tmp, err := os.CreateTemp("", "video-extract-seg-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	reader, err := h.storage.Get(ctx, segment.StorageKey)
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

	pool := newFramePersistPool(ctx, h, video, segment, h.uploadConcurrency)
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
	ctx        context.Context
	h          *ExtractFramesHandler
	video      *domainvideo.Video
	segment    *domainsegment.Segment
	sem        chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex
	publishMu  sync.Mutex
	err        error
	frames     []domainvideo.ExtractedFramePayload
	ocrJobID   uuid.UUID
	ocrJobOnce sync.Once
	ocrJobErr  error
}

func newFramePersistPool(
	ctx context.Context,
	h *ExtractFramesHandler,
	video *domainvideo.Video,
	segment *domainsegment.Segment,
	concurrency int,
) *framePersistPool {
	if concurrency <= 0 {
		concurrency = domainjob.DefaultFrameUploadConcurrency
	}
	return &framePersistPool{
		ctx:     ctx,
		h:       h,
		video:   video,
		segment: segment,
		sem:     make(chan struct{}, concurrency),
		frames:  make([]domainvideo.ExtractedFramePayload, 0),
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

		storageKey := domainvideo.NewFrameStorageKey(p.video.ID, p.segment.SegmentIndex, item.Index)
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

		timestampMs := p.segment.OffsetMs + item.TimestampMs
		frame := domainframe.NewFrame(
			p.video.ID,
			p.segment.SegmentIndex,
			item.Index,
			timestampMs,
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
		if err := p.publishOCRRequest(payload); err != nil {
			p.setErr(fmt.Errorf("%w: %w", errPersistFrame, err))
			return
		}
		p.mu.Lock()
		p.frames = append(p.frames, payload)
		p.mu.Unlock()
	}(item)

	return nil
}

func (p *framePersistPool) ensureOCRJob() (uuid.UUID, error) {
	p.ocrJobOnce.Do(func() {
		job, err := p.h.jobRepo.GetByVideoIDAndType(p.ctx, p.video.ID, domainjob.TypeOCR)
		if err != nil {
			p.ocrJobErr = err
			return
		}
		if job == nil {
			job = domainjob.NewOCRJob(p.video.ID)
			if err := p.h.jobRepo.Save(p.ctx, job); err != nil {
				p.ocrJobErr = err
				return
			}
		}
		p.ocrJobID = job.ID
	})
	return p.ocrJobID, p.ocrJobErr
}

// publishOCRRequest enqueues one OCR message as soon as the frame is stored.
func (p *framePersistPool) publishOCRRequest(payload domainvideo.ExtractedFramePayload) error {
	ocrJobID, err := p.ensureOCRJob()
	if err != nil {
		return err
	}
	p.publishMu.Lock()
	defer p.publishMu.Unlock()
	p.video.RequestOCRFrames(ocrJobID, []domainvideo.ExtractedFramePayload{payload})
	return p.h.videoRepo.WithTransaction(p.ctx, func(txCtx context.Context) error {
		return p.h.outbox.StoreEvents(txCtx, p.video.PullEvents())
	})
}

func (p *framePersistPool) Wait() ([]domainvideo.ExtractedFramePayload, error) {
	p.wg.Wait()
	p.mu.Lock()
	defer p.mu.Unlock()
	sort.Slice(p.frames, func(i, j int) bool {
		return p.frames[i].TimestampMs < p.frames[j].TimestampMs
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

func (h *ExtractFramesHandler) completeSegment(
	ctx context.Context,
	video *domainvideo.Video,
	job *domainjob.Job,
	segment *domainsegment.Segment,
	frames []domainvideo.ExtractedFramePayload,
) error {
	var (
		allDone   bool
		allFrames []domainvideo.ExtractedFramePayload
		completed int
		expected  int
	)
	frameCount := len(frames)

	err := h.videoRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		current, err := h.segmentRepo.GetByID(txCtx, segment.ID)
		if err != nil {
			return err
		}
		if current == nil {
			return errors.New("segment not found")
		}
		alreadyDone := current.Status == domainsegment.StatusSuccess
		if alreadyDone {
			jobReloaded, err := h.jobRepo.GetByID(txCtx, job.ID)
			if err != nil {
				return err
			}
			if jobReloaded != nil {
				*job = *jobReloaded
			}
			allDone = job.SegmentsComplete()
		} else {
			segment.MarkSuccess()
			if err := h.segmentRepo.Update(txCtx, segment); err != nil {
				return err
			}
			completed, expected, err = h.jobRepo.IncrementCompletedSegmentCount(txCtx, job.ID)
			if err != nil {
				return err
			}
			job.CompletedSegmentCount = completed
			job.ExpectedSegmentCount = expected
			allDone = expected > 0 && completed >= expected
		}

		video.RecordSegmentFramesExtracted(
			job.ID,
			segment.ID,
			segment.SegmentIndex,
			frameCount,
			job.ExpectedSegmentCount,
			job.CompletedSegmentCount,
		)

		ocrJob, err := h.jobRepo.GetByVideoIDAndType(txCtx, video.ID, domainjob.TypeOCR)
		if err != nil {
			return err
		}
		if ocrJob == nil {
			ocrJob = domainjob.NewOCRJob(video.ID)
			if err := h.jobRepo.Save(txCtx, ocrJob); err != nil {
				return err
			}
		}
		_ = ocrJob

		if allDone && job.Status != domainjob.StatusSuccess {
			listed, err := h.frameRepo.ListByVideoID(txCtx, video.ID)
			if err != nil {
				return err
			}
			allFrames = toExtractedPayloads(listed)
			job.SetExpectedFrameCount(len(allFrames))
			job.MarkSuccess()
			if err := video.MarkFramesReady(job.ID, job.Type, job.Status, allFrames); err != nil {
				return err
			}
			if err := h.jobRepo.Update(txCtx, job); err != nil {
				return err
			}
		}

		if err := h.videoRepo.Update(txCtx, video); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, video.PullEvents())
	})
	if err != nil {
		return err
	}

	h.cleanupSegmentObject(ctx, video, segment)
	if allDone {
		h.cleanupAllSegmentObjects(ctx, video)
	}
	return nil
}

func (h *ExtractFramesHandler) maybeFinalize(ctx context.Context, video *domainvideo.Video, job *domainjob.Job) error {
	if !job.SegmentsComplete() || job.Status == domainjob.StatusSuccess {
		return nil
	}
	listed, err := h.frameRepo.ListByVideoID(ctx, video.ID)
	if err != nil {
		return messaging.Retryable(err)
	}
	payloads := toExtractedPayloads(listed)
	job.SetExpectedFrameCount(len(payloads))
	job.MarkSuccess()
	if err := video.MarkFramesReady(job.ID, job.Type, job.Status, payloads); err != nil {
		return messaging.NonRetryable(err)
	}
	err = h.videoRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.jobRepo.Update(txCtx, job); err != nil {
			return err
		}
		if err := h.videoRepo.Update(txCtx, video); err != nil {
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
	if err != nil {
		return err
	}
	h.cleanupAllSegmentObjects(ctx, video)
	return nil
}

func (h *ExtractFramesHandler) failSegment(
	ctx context.Context,
	video *domainvideo.Video,
	job *domainjob.Job,
	segment *domainsegment.Segment,
	reason string,
) error {
	segment.MarkFailed(reason)
	job.MarkFailed(reason)
	if err := video.MarkExtractionFailed(job.ID, job.Type, job.Status, reason); err != nil {
		return err
	}
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	return h.videoRepo.WithTransaction(writeCtx, func(txCtx context.Context) error {
		if err := h.segmentRepo.Update(txCtx, segment); err != nil {
			return err
		}
		if err := h.videoRepo.Update(txCtx, video); err != nil {
			return err
		}
		if err := h.jobRepo.Update(txCtx, job); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, video.PullEvents())
	})
}

func (h *ExtractFramesHandler) cleanupSegmentObject(
	ctx context.Context,
	video *domainvideo.Video,
	segment *domainsegment.Segment,
) {
	if segment == nil || segment.StorageKey == "" || segment.StorageKey == video.StorageKey {
		return
	}
	_ = h.storage.Delete(ctx, segment.StorageKey)
}

func (h *ExtractFramesHandler) cleanupAllSegmentObjects(ctx context.Context, video *domainvideo.Video) {
	segments, err := h.segmentRepo.ListByVideoID(ctx, video.ID)
	if err != nil {
		return
	}
	for _, segment := range segments {
		h.cleanupSegmentObject(ctx, video, segment)
	}
}

func toExtractedPayloads(frames []*domainframe.Frame) []domainvideo.ExtractedFramePayload {
	out := make([]domainvideo.ExtractedFramePayload, 0, len(frames))
	for _, frame := range frames {
		out = append(out, domainvideo.ExtractedFramePayload{
			ID:              frame.ID.String(),
			Index:           frame.Index,
			TimestampMs:     frame.TimestampMs,
			StorageKey:      frame.StorageKey,
			SelectionReason: frame.SelectionReason,
			PHashDistance:   frame.PHashDistance,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].TimestampMs < out[j].TimestampMs
	})
	return out
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
