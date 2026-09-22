package video

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-api/internal/application/messaging"
	domainfinding "go-api/internal/domain/finding"
	domainframe "go-api/internal/domain/frame"
	domainjob "go-api/internal/domain/job"
	domainocr "go-api/internal/domain/ocrresult"
	"go-api/internal/domain/port"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
)

type ClassifyFramesCommand struct {
	VideoID uuid.UUID
}

type ClassifyFramesHandler struct {
	videoRepo   domainvideo.VideoWriteRepository
	jobRepo     domainjob.JobWriteRepository
	frameRepo   domainframe.FrameWriteRepository
	ocrRepo     domainocr.ResultWriteRepository
	findingRepo domainfinding.FindingWriteRepository
	outbox      port.OutboxRepository
	classifier  port.Classifier
	threshold   float64
	timeout     time.Duration
}

func NewClassifyFramesHandler(
	videoRepo domainvideo.VideoWriteRepository,
	jobRepo domainjob.JobWriteRepository,
	frameRepo domainframe.FrameWriteRepository,
	ocrRepo domainocr.ResultWriteRepository,
	findingRepo domainfinding.FindingWriteRepository,
	outbox port.OutboxRepository,
	classifier port.Classifier,
	threshold float64,
	timeout time.Duration,
) *ClassifyFramesHandler {
	if threshold <= 0 {
		threshold = 0.7
	}
	return &ClassifyFramesHandler{
		videoRepo:   videoRepo,
		jobRepo:     jobRepo,
		frameRepo:   frameRepo,
		ocrRepo:     ocrRepo,
		findingRepo: findingRepo,
		outbox:      outbox,
		classifier:  classifier,
		threshold:   threshold,
		timeout:     timeout,
	}
}

func (h *ClassifyFramesHandler) Handle(ctx context.Context, cmd ClassifyFramesCommand) error {
	if h.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.timeout)
		defer cancel()
	}

	video, job, extractJob, err := h.load(ctx, cmd.VideoID)
	if err != nil {
		return err
	}
	if job.Status == domainjob.StatusClassified || video.Status == domainvideo.StatusClassified {
		return nil
	}

	frames, err := h.frameRepo.ListByJobID(ctx, extractJob.ID)
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load frames")
	}

	ocrResults, err := h.ocrRepo.ListByFrameIDs(ctx, frameIDs(frames))
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load ocr results")
	}

	existing, err := h.findingRepo.ListByFrameIDs(ctx, frameIDs(frames))
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load findings")
	}
	known := findingsByFrame(existing)

	if err := h.markProcessing(ctx, video, job, len(frames)); err != nil {
		return err
	}

	pending, skipped := pendingClassifyFrames(frames, ocrResults, existing)
	if err := h.persistFindings(ctx, skipped, known); err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to persist findings")
	}

	if len(pending) > 0 {
		raw, recErr := h.classifier.Classify(ctx, pending, h.threshold)
		if recErr != nil {
			return h.failOrRetry(ctx, video, job, recErr, "classify service unavailable")
		}
		results := h.normalizeResults(pending, raw)
		if err := h.persistFindings(ctx, results, known); err != nil {
			return h.failOrRetry(ctx, video, job, err, "failed to persist findings")
		}
	}

	stored, err := h.findingRepo.ListByFrameIDs(ctx, frameIDs(frames))
	if err != nil {
		return h.failOrRetry(ctx, video, job, err, "failed to load findings")
	}
	if len(stored) < len(frames) {
		return h.failOrRetry(ctx, video, job, errors.New("classify incomplete"), "classify incomplete")
	}
	if err := h.markReady(ctx, video, job, stored); err != nil {
		return messaging.Retryable(err)
	}
	return nil
}

func (h *ClassifyFramesHandler) load(ctx context.Context, videoID uuid.UUID) (*domainvideo.Video, *domainjob.Job, *domainjob.Job, error) {
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

	job, err := h.jobRepo.GetByVideoIDAndType(ctx, video.ID, domainjob.TypeClassify)
	if err != nil {
		return nil, nil, nil, messaging.Retryable(err)
	}
	if job == nil {
		job = domainjob.NewClassifyJob(video.ID)
		if err := h.jobRepo.Save(ctx, job); err != nil {
			return nil, nil, nil, messaging.Retryable(err)
		}
	}
	return video, job, extractJob, nil
}

func (h *ClassifyFramesHandler) markProcessing(ctx context.Context, video *domainvideo.Video, job *domainjob.Job, expected int) error {
	job.MarkClassifying(expected)
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

func (h *ClassifyFramesHandler) persistFindings(
	ctx context.Context,
	findings []*domainfinding.Finding,
	known map[uuid.UUID]*domainfinding.Finding,
) error {
	if len(findings) == 0 {
		return nil
	}
	if err := h.findingRepo.UpsertAll(ctx, findings); err != nil {
		return err
	}
	for _, finding := range findings {
		known[finding.FrameID] = finding
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
	job.MarkClassifyFailed(reason)
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

func (h *ClassifyFramesHandler) markReady(ctx context.Context, video *domainvideo.Video, job *domainjob.Job, findings []*domainfinding.Finding) error {
	job.MarkClassified()
	if err := video.MarkClassified(job.ID, job.Type, job.Status, job.ExpectedFrameCount, toFindingPayloads(findings)); err != nil {
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

func (h *ClassifyFramesHandler) normalizeResults(frames []port.ClassifyFrame, raw []port.ClassifyItemResult) []*domainfinding.Finding {
	byFrame := make(map[string]port.ClassifyItemResult, len(raw))
	for _, item := range raw {
		byFrame[item.FrameID] = item
	}

	out := make([]*domainfinding.Finding, 0, len(frames))
	for _, frame := range frames {
		frameID, err := uuid.Parse(frame.FrameID)
		if err != nil {
			continue
		}
		item, ok := byFrame[frame.FrameID]
		if !ok {
			out = append(out, domainfinding.NewFinding(frameID, false, 0, nil, domainfinding.StatusFailed, "missing classify result"))
			continue
		}
		status := item.Status
		if status == "" {
			status = domainfinding.StatusSuccess
		}
		reason := ""
		if status == domainfinding.StatusFailed {
			reason = "classify engine failed"
		}
		out = append(out, domainfinding.NewFinding(
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
	findings []*domainfinding.Finding,
) ([]port.ClassifyFrame, []*domainfinding.Finding) {
	done := make(map[uuid.UUID]struct{}, len(findings))
	for _, finding := range findings {
		if finding.IsFinal() {
			done[finding.FrameID] = struct{}{}
		}
	}

	ocrByFrame := make(map[uuid.UUID]*domainocr.Result, len(ocrResults))
	for _, result := range ocrResults {
		ocrByFrame[result.FrameID] = result
	}

	pending := make([]port.ClassifyFrame, 0)
	skipped := make([]*domainfinding.Finding, 0)
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
			skipped = append(skipped, domainfinding.NewFinding(
				frame.ID,
				false,
				0,
				nil,
				domainfinding.StatusSkipped,
				"",
			))
			continue
		}
		pending = append(pending, port.ClassifyFrame{FrameID: frame.ID.String(), Text: text})
	}
	return pending, skipped
}

func findingsByFrame(findings []*domainfinding.Finding) map[uuid.UUID]*domainfinding.Finding {
	out := make(map[uuid.UUID]*domainfinding.Finding, len(findings))
	for _, finding := range findings {
		out[finding.FrameID] = finding
	}
	return out
}

func toDomainCategories(items []port.ClassifyCategory) []domainfinding.Category {
	out := make([]domainfinding.Category, 0, len(items))
	for _, item := range items {
		out = append(out, domainfinding.Category{Name: item.Name, Probability: item.Probability})
	}
	return out
}

func toFindingPayloads(findings []*domainfinding.Finding) []domainvideo.FrameFindingPayload {
	out := make([]domainvideo.FrameFindingPayload, 0, len(findings))
	for _, finding := range findings {
		categories := make([]domainvideo.FindingCategoryPayload, 0, len(finding.Categories))
		for _, category := range finding.Categories {
			categories = append(categories, domainvideo.FindingCategoryPayload{
				Name:        category.Name,
				Probability: category.Probability,
			})
		}
		out = append(out, domainvideo.FrameFindingPayload{
			FrameID:      finding.FrameID.String(),
			Confidential: finding.Confidential,
			Probability:  finding.Probability,
			Categories:   categories,
			Status:       finding.Status,
		})
	}
	return out
}
