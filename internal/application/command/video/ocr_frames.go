package video

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"go-api/internal/application/messaging"
	domainframe "go-api/internal/domain/frame"
	domainjob "go-api/internal/domain/job"
	domainocr "go-api/internal/domain/ocrresult"
	"go-api/internal/domain/port"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
)

type OCRFramesCommand struct {
	VideoID uuid.UUID
}

type OCRFramesHandler struct {
	videoRepo     domainvideo.VideoWriteRepository
	jobRepo       domainjob.JobWriteRepository
	frameRepo     domainframe.FrameWriteRepository
	ocrRepo       domainocr.ResultWriteRepository
	outbox        port.OutboxRepository
	storage       port.Storage
	engine        port.OCREngine
	batchSize     int
	minConfidence float64
	lang          string
	timeout       time.Duration
}

func NewOCRFramesHandler(
	videoRepo domainvideo.VideoWriteRepository,
	jobRepo domainjob.JobWriteRepository,
	frameRepo domainframe.FrameWriteRepository,
	ocrRepo domainocr.ResultWriteRepository,
	outbox port.OutboxRepository,
	storage port.Storage,
	engine port.OCREngine,
	batchSize int,
	minConfidence float64,
	lang string,
	timeout time.Duration,
) *OCRFramesHandler {
	if batchSize <= 0 {
		batchSize = 8
	}
	if lang == "" {
		lang = "fr+en"
	}
	return &OCRFramesHandler{
		videoRepo:     videoRepo,
		jobRepo:       jobRepo,
		frameRepo:     frameRepo,
		ocrRepo:       ocrRepo,
		outbox:        outbox,
		storage:       storage,
		engine:        engine,
		batchSize:     batchSize,
		minConfidence: minConfidence,
		lang:          lang,
		timeout:       timeout,
	}
}

func (h *OCRFramesHandler) Handle(ctx context.Context, cmd OCRFramesCommand) error {
	if h.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.timeout)
		defer cancel()
	}

	video, job, extractJob, err := h.load(ctx, cmd.VideoID)
	if err != nil {
		return err
	}
	if job.Status == domainjob.StatusOCRReady || video.Status == domainvideo.StatusOCRReady {
		return nil
	}

	frames, err := h.frameRepo.ListByJobID(ctx, extractJob.ID)
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load frames")
	}

	existing, err := h.ocrRepo.ListByFrameIDs(ctx, frameIDs(frames))
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load ocr results")
	}
	known := resultsByFrame(existing)

	if err := h.markProcessing(ctx, video, job, len(frames), len(known)); err != nil {
		return err
	}

	pending := pendingFrames(frames, existing)
	for _, batch := range chunkFrames(pending, h.batchSize) {
		results, recErr := h.recognizeBatch(ctx, batch)
		if recErr != nil {
			return h.failOrRetry(ctx, video, job, recErr, "ocr service unavailable")
		}
		if err := h.persistBatch(ctx, job, results, known); err != nil {
			return h.failOrRetry(ctx, video, job, err, "failed to persist ocr results")
		}
	}

	stored, err := h.ocrRepo.ListByFrameIDs(ctx, frameIDs(frames))
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load ocr results")
	}
	if len(stored) < len(frames) {
		return h.failOrRetry(ctx, video, job, errors.New("ocr incomplete"), "ocr incomplete")
	}
	if err := h.markReady(ctx, video, job, frames, stored); err != nil {
		return messaging.Retryable(err)
	}
	return nil
}

func (h *OCRFramesHandler) load(ctx context.Context, videoID uuid.UUID) (*domainvideo.Video, *domainjob.Job, *domainjob.Job, error) {
	video, err := h.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		return nil, nil, nil, messaging.Retryable(err)
	}
	if video == nil {
		return nil, nil, nil, messaging.NonRetryable(domainvideo.ErrVideoNotFound)
	}

	extractJob, err := h.jobRepo.GetByVideoIDAndType(ctx, video.ID, domainjob.TypeExtractFrames)
	if err != nil {
		return nil, nil, nil, messaging.Retryable(err)
	}
	if extractJob == nil {
		return nil, nil, nil, messaging.NonRetryable(errors.New("extract frames job not found"))
	}

	job, err := h.jobRepo.GetByVideoIDAndType(ctx, video.ID, domainjob.TypeOCR)
	if err != nil {
		return nil, nil, nil, messaging.Retryable(err)
	}
	if job == nil {
		job = domainjob.NewOCRJob(video.ID)
		if err := h.jobRepo.Save(ctx, job); err != nil {
			return nil, nil, nil, messaging.Retryable(err)
		}
	}
	return video, job, extractJob, nil
}

func (h *OCRFramesHandler) markProcessing(
	ctx context.Context,
	video *domainvideo.Video,
	job *domainjob.Job,
	expected, alreadyCompleted int,
) error {
	job.MarkOCRProcessing(expected, alreadyCompleted)
	if err := video.MarkOCRProcessing(job.ID, job.Type, job.Status, expected, alreadyCompleted); err != nil {
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

func (h *OCRFramesHandler) persistBatch(
	ctx context.Context,
	job *domainjob.Job,
	results []*domainocr.Result,
	known map[uuid.UUID]*domainocr.Result,
) error {
	if len(results) == 0 {
		return nil
	}
	added := 0
	for _, result := range results {
		if _, exists := known[result.FrameID]; !exists {
			added++
		}
	}
	previousCompleted := job.OCRCompletedCount
	job.AddOCRCompleted(added)
	err := h.videoRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.ocrRepo.UpsertAll(txCtx, results); err != nil {
			return err
		}
		return h.jobRepo.Update(txCtx, job)
	})
	if err != nil {
		job.OCRCompletedCount = previousCompleted
		return err
	}
	for _, result := range results {
		known[result.FrameID] = result
	}
	return nil
}

func (h *OCRFramesHandler) failOrRetry(
	ctx context.Context,
	video *domainvideo.Video,
	job *domainjob.Job,
	err error,
	reason string,
) error {
	if !messaging.LastAttempt(ctx) {
		return messaging.Retryable(err)
	}
	if markErr := h.markFailed(ctx, video, job, reason); markErr != nil {
		return messaging.NonRetryable(errors.Join(err, markErr))
	}
	return messaging.NonRetryable(err)
}

func (h *OCRFramesHandler) markFailed(
	ctx context.Context,
	video *domainvideo.Video,
	job *domainjob.Job,
	reason string,
) error {
	job.MarkOCRFailed(reason)
	if err := video.MarkOCRFailed(job.ID, job.Type, job.Status, reason, job.ExpectedFrameCount, job.OCRCompletedCount); err != nil {
		return err
	}

	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	return h.videoRepo.WithTransaction(writeCtx, func(txCtx context.Context) error {
		if err := h.videoRepo.Update(txCtx, video); err != nil {
			return err
		}
		if err := h.jobRepo.Update(txCtx, job); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, video.PullEvents())
	})
}

func (h *OCRFramesHandler) markReady(
	ctx context.Context,
	video *domainvideo.Video,
	job *domainjob.Job,
	frames []*domainframe.Frame,
	results []*domainocr.Result,
) error {
	job.MarkOCRReady()
	if err := video.MarkOCRReady(job.ID, job.Type, job.Status, job.ExpectedFrameCount, job.OCRCompletedCount, toOCRPayloads(frames, results)); err != nil {
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

func (h *OCRFramesHandler) recognizeBatch(ctx context.Context, frames []*domainframe.Frame) ([]*domainocr.Result, error) {
	images := make([]port.OCRImage, 0, len(frames))
	isolated := make([]*domainocr.Result, 0)

	for _, frame := range frames {
		data, err := h.downloadFrame(ctx, frame.StorageKey)
		if err != nil {
			isolated = append(isolated, domainocr.NewResult(
				frame.ID,
				"",
				0,
				domainocr.StatusFailed,
				"frame download failed",
			))
			continue
		}
		images = append(images, port.OCRImage{FrameID: frame.ID.String(), Data: data})
	}

	if len(images) == 0 {
		return isolated, nil
	}

	raw, err := h.engine.Recognize(ctx, images, h.lang)
	if err != nil {
		return nil, err
	}

	byFrame := make(map[string]port.OCRItemResult, len(raw))
	for _, item := range raw {
		byFrame[item.FrameID] = item
	}

	out := make([]*domainocr.Result, 0, len(frames))
	out = append(out, isolated...)
	for _, image := range images {
		frameID, err := uuid.Parse(image.FrameID)
		if err != nil {
			continue
		}
		item, ok := byFrame[image.FrameID]
		if !ok {
			out = append(out, domainocr.NewResult(frameID, "", 0, domainocr.StatusFailed, "missing ocr result"))
			continue
		}
		out = append(out, h.normalizeResult(frameID, item))
	}
	return out, nil
}

func (h *OCRFramesHandler) downloadFrame(ctx context.Context, key string) ([]byte, error) {
	reader, err := h.storage.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func (h *OCRFramesHandler) normalizeResult(frameID uuid.UUID, item port.OCRItemResult) *domainocr.Result {
	text := strings.TrimSpace(item.Text)
	status := item.Status
	if status == "" {
		if text == "" {
			status = domainocr.StatusEmpty
		} else {
			status = domainocr.StatusSuccess
		}
	}

	switch status {
	case domainocr.StatusFailed:
		return domainocr.NewResult(frameID, "", item.Confidence, domainocr.StatusFailed, "ocr engine failed")
	case domainocr.StatusEmpty:
		return domainocr.NewResult(frameID, "", 0, domainocr.StatusEmpty, "")
	default:
		if text == "" {
			return domainocr.NewResult(frameID, "", 0, domainocr.StatusEmpty, "")
		}
		if h.minConfidence > 0 && item.Confidence < h.minConfidence {
			return domainocr.NewResult(frameID, "", item.Confidence, domainocr.StatusFailed, "low_confidence")
		}
		return domainocr.NewResult(frameID, text, item.Confidence, domainocr.StatusSuccess, "")
	}
}

func frameIDs(frames []*domainframe.Frame) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(frames))
	for _, frame := range frames {
		ids = append(ids, frame.ID)
	}
	return ids
}

func resultsByFrame(results []*domainocr.Result) map[uuid.UUID]*domainocr.Result {
	out := make(map[uuid.UUID]*domainocr.Result, len(results))
	for _, result := range results {
		out[result.FrameID] = result
	}
	return out
}

func pendingFrames(frames []*domainframe.Frame, results []*domainocr.Result) []*domainframe.Frame {
	done := make(map[uuid.UUID]struct{}, len(results))
	for _, result := range results {
		if result.IsFinal() {
			done[result.FrameID] = struct{}{}
		}
	}

	pending := make([]*domainframe.Frame, 0, len(frames))
	for _, frame := range frames {
		if _, ok := done[frame.ID]; ok {
			continue
		}
		pending = append(pending, frame)
	}
	return pending
}

func chunkFrames(frames []*domainframe.Frame, size int) [][]*domainframe.Frame {
	if len(frames) == 0 {
		return nil
	}
	out := make([][]*domainframe.Frame, 0, (len(frames)+size-1)/size)
	for i := 0; i < len(frames); i += size {
		end := i + size
		if end > len(frames) {
			end = len(frames)
		}
		out = append(out, frames[i:end])
	}
	return out
}

func toOCRPayloads(frames []*domainframe.Frame, results []*domainocr.Result) []domainvideo.OCRFrameResultPayload {
	byID := make(map[uuid.UUID]*domainframe.Frame, len(frames))
	for _, frame := range frames {
		byID[frame.ID] = frame
	}

	out := make([]domainvideo.OCRFrameResultPayload, 0, len(results))
	for _, result := range results {
		payload := domainvideo.OCRFrameResultPayload{
			FrameID:    result.FrameID.String(),
			Text:       result.Text,
			Confidence: result.Confidence,
			Status:     result.Status,
		}
		if frame, ok := byID[result.FrameID]; ok {
			payload.FrameIndex = frame.Index
			payload.TimestampMs = frame.TimestampMs
		}
		out = append(out, payload)
	}
	return out
}
