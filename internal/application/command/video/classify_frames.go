package video

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-api/internal/application/messaging"
	domainclassification "go-api/internal/domain/classification"
	domainframe "go-api/internal/domain/frame"
	domainjob "go-api/internal/domain/job"
	domainocr "go-api/internal/domain/ocr"
	"go-api/internal/domain/port"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
)

type ClassifyFrameCommand struct {
	VideoID uuid.UUID
	FrameID uuid.UUID
}

type ClassifyFramesHandler struct {
	videoRepo          domainvideo.VideoWriteRepository
	jobRepo            domainjob.JobWriteRepository
	frameRepo          domainframe.FrameWriteRepository
	ocrRepo            domainocr.WriteRepository
	classificationRepo domainclassification.WriteRepository
	outbox             port.OutboxRepository
	classifier         port.Classifier
	threshold          float64
	timeout            time.Duration
}

func NewClassifyFramesHandler(
	videoRepo domainvideo.VideoWriteRepository,
	jobRepo domainjob.JobWriteRepository,
	frameRepo domainframe.FrameWriteRepository,
	ocrRepo domainocr.WriteRepository,
	classificationRepo domainclassification.WriteRepository,
	outbox port.OutboxRepository,
	classifier port.Classifier,
	threshold float64,
	timeout time.Duration,
) *ClassifyFramesHandler {
	if threshold <= 0 {
		threshold = 0.7
	}
	return &ClassifyFramesHandler{
		videoRepo:          videoRepo,
		jobRepo:            jobRepo,
		frameRepo:          frameRepo,
		ocrRepo:            ocrRepo,
		classificationRepo: classificationRepo,
		outbox:             outbox,
		classifier:         classifier,
		threshold:          threshold,
		timeout:            timeout,
	}
}

func (h *ClassifyFramesHandler) HandleFrame(ctx context.Context, cmd ClassifyFrameCommand) error {
	if h.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.timeout)
		defer cancel()
	}

	video, job, frameJob, ocrJob, err := h.load(ctx, cmd.VideoID)
	if err != nil {
		return err
	}
	if job.Status == domainjob.StatusSuccess || video.Status == domainvideo.StatusSuccess {
		return nil
	}

	frames, err := h.frameRepo.ListByVideoID(ctx, video.ID)
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load frames")
	}
	var target *domainframe.Frame
	for _, frame := range frames {
		if frame.ID == cmd.FrameID {
			target = frame
			break
		}
	}
	if target == nil {
		return messaging.NonRetryable(errors.New("frame not found"))
	}

	ocrResults, err := h.ocrRepo.ListByFrameIDs(ctx, []uuid.UUID{cmd.FrameID})
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load ocr")
	}
	existing, err := h.classificationRepo.ListByFrameIDs(ctx, []uuid.UUID{cmd.FrameID})
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load classifications")
	}
	if len(existing) > 0 && existing[0].IsFinal() {
		return h.tryFinalize(ctx, video, job, frameJob, ocrJob)
	}

	if err := h.markProcessing(ctx, video, job, len(frames)); err != nil {
		return err
	}

	pending, skipped := pendingClassifyFrames([]*domainframe.Frame{target}, ocrResults, existing)
	known := classificationsByFrame(existing)
	if err := h.persistClassifications(ctx, skipped, known); err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to persist classifications")
	}
	if len(pending) > 0 {
		raw, recErr := h.classifier.Classify(ctx, pending, h.threshold)
		if recErr != nil {
			return h.failOrRetry(ctx, video, job, recErr, "classify service unavailable")
		}
		results := h.normalizeResults(pending, raw)
		if err := h.persistClassifications(ctx, results, known); err != nil {
			return h.failOrRetry(ctx, video, job, err, "failed to persist classifications")
		}
	}

	return h.tryFinalize(ctx, video, job, frameJob, ocrJob)
}

// Finalize marks classify success when extract, OCR, and classify rows are complete.
func (h *ClassifyFramesHandler) Finalize(ctx context.Context, cmd ClassifyFrameCommand) error {
	video, job, frameJob, ocrJob, err := h.load(ctx, cmd.VideoID)
	if err != nil {
		return err
	}
	if job.Status == domainjob.StatusSuccess || video.Status == domainvideo.StatusSuccess {
		return nil
	}
	return h.tryFinalize(ctx, video, job, frameJob, ocrJob)
}

func (h *ClassifyFramesHandler) tryFinalize(
	ctx context.Context,
	video *domainvideo.Video,
	job *domainjob.Job,
	frameJob *domainjob.Job,
	_ *domainjob.Job,
) error {
	if frameJob.Status != domainjob.StatusSuccess {
		return nil
	}

	frames, err := h.frameRepo.ListByVideoID(ctx, video.ID)
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load frames")
	}
	if len(frames) == 0 {
		return h.markReady(ctx, video, job, nil)
	}

	ocrResults, err := h.ocrRepo.ListByFrameIDs(ctx, frameIDs(frames))
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load ocr")
	}
	if len(ocrResults) < len(frames) {
		return nil
	}

	stored, err := h.classificationRepo.ListByFrameIDs(ctx, frameIDs(frames))
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load classifications")
	}
	if len(stored) < len(frames) {
		return nil
	}

	if err := h.markReady(ctx, video, job, stored); err != nil {
		return messaging.Retryable(err)
	}
	return nil
}

func (h *ClassifyFramesHandler) load(
	ctx context.Context,
	videoID uuid.UUID,
) (*domainvideo.Video, *domainjob.Job, *domainjob.Job, *domainjob.Job, error) {
	video, err := h.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		return nil, nil, nil, nil, messaging.Retryable(err)
	}
	if video == nil {
		return nil, nil, nil, nil, messaging.NonRetryable(domainvideo.ErrVideoNotFound)
	}

	extractJob, err := h.jobRepo.GetByVideoIDAndType(ctx, video.ID, domainjob.TypeFrame)
	if err != nil {
		return nil, nil, nil, nil, messaging.Retryable(err)
	}
	if extractJob == nil {
		return nil, nil, nil, nil, messaging.NonRetryable(errors.New("frame job not found"))
	}

	ocrJob, err := h.jobRepo.GetByVideoIDAndType(ctx, video.ID, domainjob.TypeOCR)
	if err != nil {
		return nil, nil, nil, nil, messaging.Retryable(err)
	}

	job, err := h.jobRepo.GetByVideoIDAndType(ctx, video.ID, domainjob.TypeClassify)
	if err != nil {
		return nil, nil, nil, nil, messaging.Retryable(err)
	}
	if job == nil {
		job = domainjob.NewClassifyJob(video.ID)
		if err := h.jobRepo.Save(ctx, job); err != nil {
			return nil, nil, nil, nil, messaging.Retryable(err)
		}
	}
	return video, job, extractJob, ocrJob, nil
}

func (h *ClassifyFramesHandler) markProcessing(ctx context.Context, video *domainvideo.Video, job *domainjob.Job, expected int) error {
	if job.Status == domainjob.StatusProcessing {
		return nil
	}
	job.SetExpectedFrameCount(expected)
	job.MarkProcessing()
	if err := video.MarkClassifying(job.ID, job.Type, job.Status, expected); err != nil {
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

func (h *ClassifyFramesHandler) persistClassifications(
	ctx context.Context,
	items []*domainclassification.Classification,
	known map[uuid.UUID]*domainclassification.Classification,
) error {
	if len(items) == 0 {
		return nil
	}
	if err := h.classificationRepo.UpsertAll(ctx, items); err != nil {
		return err
	}
	for _, item := range items {
		known[item.FrameID] = item
	}
	return nil
}

func (h *ClassifyFramesHandler) failOrRetry(
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

func (h *ClassifyFramesHandler) markFailed(ctx context.Context, video *domainvideo.Video, job *domainjob.Job, reason string) error {
	job.MarkFailed(reason)
	if err := video.MarkClassifyFailed(job.ID, job.Type, job.Status, reason, job.ExpectedFrameCount); err != nil {
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

func (h *ClassifyFramesHandler) markReady(
	ctx context.Context,
	video *domainvideo.Video,
	job *domainjob.Job,
	items []*domainclassification.Classification,
) error {
	job.SetExpectedFrameCount(len(items))
	job.MarkSuccess()
	if err := video.MarkClassified(job.ID, job.Type, job.Status, job.ExpectedFrameCount, toClassificationPayloads(items)); err != nil {
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

func (h *ClassifyFramesHandler) normalizeResults(
	frames []port.ClassifyFrame,
	raw []port.ClassifyItemResult,
) []*domainclassification.Classification {
	byFrame := make(map[string]port.ClassifyItemResult, len(raw))
	for _, item := range raw {
		byFrame[item.FrameID] = item
	}

	out := make([]*domainclassification.Classification, 0, len(frames))
	for _, frame := range frames {
		frameID, err := uuid.Parse(frame.FrameID)
		if err != nil {
			continue
		}
		item, ok := byFrame[frame.FrameID]
		if !ok {
			out = append(out, domainclassification.New(frameID, false, 0, nil, domainclassification.StatusFailed, "missing classify result"))
			continue
		}
		status := item.Status
		if status == "" {
			status = domainclassification.StatusSuccess
		}
		reason := ""
		if status == domainclassification.StatusFailed {
			reason = "classify engine failed"
		}
		out = append(out, domainclassification.New(
			frameID,
			item.Confidential,
			item.Probability,
			toDomainCategories(item.Categories),
			status,
			reason,
		))
	}
	return out
}

func pendingClassifyFrames(
	frames []*domainframe.Frame,
	ocrResults []*domainocr.Result,
	items []*domainclassification.Classification,
) ([]port.ClassifyFrame, []*domainclassification.Classification) {
	done := make(map[uuid.UUID]struct{}, len(items))
	for _, item := range items {
		if item.IsFinal() {
			done[item.FrameID] = struct{}{}
		}
	}

	ocrByFrame := make(map[uuid.UUID]*domainocr.Result, len(ocrResults))
	for _, result := range ocrResults {
		ocrByFrame[result.FrameID] = result
	}

	pending := make([]port.ClassifyFrame, 0)
	skipped := make([]*domainclassification.Classification, 0)
	for _, frame := range frames {
		if _, ok := done[frame.ID]; ok {
			continue
		}
		result := ocrByFrame[frame.ID]
		text := ""
		if result != nil {
			text = strings.TrimSpace(result.Text)
		}
		if text == "" {
			skipped = append(skipped, domainclassification.New(
				frame.ID,
				false,
				0,
				nil,
				domainclassification.StatusSkipped,
				"",
			))
			continue
		}
		pending = append(pending, port.ClassifyFrame{FrameID: frame.ID.String(), Text: text})
	}
	return pending, skipped
}

func classificationsByFrame(items []*domainclassification.Classification) map[uuid.UUID]*domainclassification.Classification {
	out := make(map[uuid.UUID]*domainclassification.Classification, len(items))
	for _, item := range items {
		out[item.FrameID] = item
	}
	return out
}

func toDomainCategories(items []port.ClassifyCategory) []domainclassification.Category {
	out := make([]domainclassification.Category, 0, len(items))
	for _, item := range items {
		out = append(out, domainclassification.Category{Name: item.Name, Probability: item.Probability})
	}
	return out
}

func toClassificationPayloads(items []*domainclassification.Classification) []domainvideo.ClassificationPayload {
	out := make([]domainvideo.ClassificationPayload, 0, len(items))
	for _, item := range items {
		categories := make([]domainvideo.ClassificationCategoryPayload, 0, len(item.Categories))
		for _, category := range item.Categories {
			categories = append(categories, domainvideo.ClassificationCategoryPayload{
				Name:        category.Name,
				Probability: category.Probability,
			})
		}
		out = append(out, domainvideo.ClassificationPayload{
			FrameID:      item.FrameID.String(),
			Confidential: item.Confidential,
			Probability:  item.Probability,
			Categories:   categories,
			Status:       item.Status,
		})
	}
	return out
}
