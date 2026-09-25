package video

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-api/internal/application/messaging"
	domainframe "go-api/internal/domain/frame"
	domainjob "go-api/internal/domain/job"
	domainocr "go-api/internal/domain/ocr"
	"go-api/internal/domain/port"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
)

type StartOCRFramesCommand struct {
	VideoID uuid.UUID
}

type PersistOCRBatchCommand struct {
	VideoID uuid.UUID
	Results []domainvideo.OCRFrameResultPayload
}

type OCRFramesHandler struct {
	videoRepo     domainvideo.VideoWriteRepository
	jobRepo       domainjob.JobWriteRepository
	frameRepo     domainframe.FrameWriteRepository
	ocrRepo       domainocr.WriteRepository
	outbox        port.OutboxRepository
	minConfidence float64
}

func NewOCRFramesHandler(
	videoRepo domainvideo.VideoWriteRepository,
	jobRepo domainjob.JobWriteRepository,
	frameRepo domainframe.FrameWriteRepository,
	ocrRepo domainocr.WriteRepository,
	outbox port.OutboxRepository,
	minConfidence float64,
) *OCRFramesHandler {
	return &OCRFramesHandler{
		videoRepo:     videoRepo,
		jobRepo:       jobRepo,
		frameRepo:     frameRepo,
		ocrRepo:       ocrRepo,
		outbox:        outbox,
		minConfidence: minConfidence,
	}
}

func (h *OCRFramesHandler) Start(ctx context.Context, cmd StartOCRFramesCommand) error {
	video, job, _, err := h.load(ctx, cmd.VideoID)
	if err != nil {
		return err
	}
	if job.Status == domainjob.StatusSuccess || video.Status == domainvideo.StatusOCRReady {
		return nil
	}

	frames, err := h.frameRepo.ListByVideoID(ctx, video.ID)
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load frames")
	}

	existing, err := h.ocrRepo.ListByFrameIDs(ctx, frameIDs(frames))
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load ocr")
	}
	if err := h.markProcessing(ctx, video, job, len(frames), countFinal(existing)); err != nil {
		return err
	}
	if len(frames) == 0 || len(existing) >= len(frames) {
		if err := h.markReady(ctx, video, job, frames, existing); err != nil {
			return messaging.Retryable(err)
		}
	}
	return nil
}

func (h *OCRFramesHandler) PersistBatch(ctx context.Context, cmd PersistOCRBatchCommand) error {
	video, job, extractJob, err := h.load(ctx, cmd.VideoID)
	if err != nil {
		return err
	}
	if job.Status == domainjob.StatusSuccess || video.Status == domainvideo.StatusOCRReady {
		return nil
	}

	frames, err := h.frameRepo.ListByVideoID(ctx, video.ID)
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load frames")
	}

	existing, err := h.ocrRepo.ListByFrameIDs(ctx, frameIDs(frames))
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load ocr")
	}
	known := resultsByFrame(existing)
	extractDone := extractJob.Status == domainjob.StatusSuccess

	if extractDone && (video.Status == domainvideo.StatusFramesReady || job.Status == domainjob.StatusPending) {
		if err := h.markProcessing(ctx, video, job, len(frames), countFinal(existing)); err != nil {
			return err
		}
	}

	incoming := h.normalizePayloads(cmd.Results)
	if err := h.persistBatch(ctx, video, job, incoming, known); err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to persist ocr")
	}
	if !extractDone {
		return nil
	}

	stored, err := h.ocrRepo.ListByFrameIDs(ctx, frameIDs(frames))
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load ocr")
	}
	if len(stored) < len(frames) || len(frames) == 0 {
		return nil
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

	extractJob, err := h.jobRepo.GetByVideoIDAndType(ctx, video.ID, domainjob.TypeFrame)
	if err != nil {
		return nil, nil, nil, messaging.Retryable(err)
	}
	if extractJob == nil {
		return nil, nil, nil, messaging.NonRetryable(errors.New("frame job not found"))
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
	job.SetOCRProgress(expected, alreadyCompleted)
	job.MarkProcessing()
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
	video *domainvideo.Video,
	job *domainjob.Job,
	results []*domainocr.Result,
	known map[uuid.UUID]*domainocr.Result,
) error {
	if len(results) == 0 {
		return nil
	}
	added := 0
	for _, result := range results {
		if existing, ok := known[result.FrameID]; !ok || !existing.IsFinal() {
			added++
		}
	}
	previousCompleted := job.OCRCompletedCount
	job.AddOCRCompleted(added)
	if added > 0 {
		if err := video.RecordOCRProgress(job.ID, job.Type, job.Status, job.ExpectedFrameCount, job.OCRCompletedCount); err != nil {
			job.OCRCompletedCount = previousCompleted
			return err
		}
	}
	err := h.videoRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.ocrRepo.UpsertAll(txCtx, results); err != nil {
			return err
		}
		if err := h.jobRepo.Update(txCtx, job); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, video.PullEvents())
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
	job.MarkFailed(reason)
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
	job.MarkSuccess()
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
		classifyJob, err := h.jobRepo.GetByVideoIDAndType(txCtx, video.ID, domainjob.TypeClassify)
		if err != nil {
			return err
		}
		if classifyJob == nil {
			if err := h.jobRepo.Save(txCtx, domainjob.NewClassifyJob(video.ID)); err != nil {
				return err
			}
		}
		return h.outbox.StoreEvents(txCtx, video.PullEvents())
	})
}

func (h *OCRFramesHandler) normalizePayloads(items []domainvideo.OCRFrameResultPayload) []*domainocr.Result {
	out := make([]*domainocr.Result, 0, len(items))
	for _, item := range items {
		frameID, err := uuid.Parse(item.FrameID)
		if err != nil {
			continue
		}
		out = append(out, h.normalizeResult(frameID, item.Text, item.Confidence, item.Status, item.Lines))
	}
	return out
}

func (h *OCRFramesHandler) normalizeResult(
	frameID uuid.UUID,
	text string,
	confidence float64,
	status string,
	lines []domainvideo.OCRLinePayload,
) *domainocr.Result {
	text = strings.TrimSpace(text)
	if status == "" {
		if text == "" {
			status = domainocr.StatusEmpty
		} else {
			status = domainocr.StatusSuccess
		}
	}

	switch status {
	case domainocr.StatusFailed:
		return domainocr.NewResult(frameID, "", confidence, domainocr.StatusFailed, "ocr engine failed", nil)
	case domainocr.StatusEmpty:
		return domainocr.NewResult(frameID, "", 0, domainocr.StatusEmpty, "", nil)
	default:
		if text == "" {
			return domainocr.NewResult(frameID, "", 0, domainocr.StatusEmpty, "", nil)
		}
		if h.minConfidence > 0 && confidence < h.minConfidence {
			return domainocr.NewResult(frameID, "", confidence, domainocr.StatusFailed, "low_confidence", nil)
		}
		return domainocr.NewResult(frameID, text, confidence, domainocr.StatusSuccess, "", toOCRLines(lines))
	}
}

func toOCRLines(items []domainvideo.OCRLinePayload) []domainocr.Line {
	out := make([]domainocr.Line, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(item.Text)
		if text == "" {
			continue
		}
		box := make([]domainocr.Point, 0, len(item.Box))
		for _, point := range item.Box {
			box = append(box, domainocr.Point{X: point.X, Y: point.Y})
		}
		if len(box) != 4 {
			box = []domainocr.Point{}
		}
		out = append(out, domainocr.Line{
			Text:       text,
			Confidence: item.Confidence,
			Box:        box,
		})
	}
	return out
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

func countFinal(results []*domainocr.Result) int {
	n := 0
	for _, result := range results {
		if result.IsFinal() {
			n++
		}
	}
	return n
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
