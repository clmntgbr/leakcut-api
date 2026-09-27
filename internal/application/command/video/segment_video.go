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
	domainjob "go-api/internal/domain/job"
	"go-api/internal/domain/port"
	domainsegment "go-api/internal/domain/segment"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
)

const segmentContentType = "video/mp4"

type SegmentVideoCommand struct {
	VideoID uuid.UUID
}

type SegmentVideoHandler struct {
	videoRepo   domainvideo.VideoWriteRepository
	jobRepo     domainjob.JobWriteRepository
	segmentRepo domainsegment.WriteRepository
	outbox      port.OutboxRepository
	storage     port.Storage
	splitter    port.SegmentSplitter
	extractor   port.FrameExtractor
	timeout     time.Duration
}

func NewSegmentVideoHandler(
	videoRepo domainvideo.VideoWriteRepository,
	jobRepo domainjob.JobWriteRepository,
	segmentRepo domainsegment.WriteRepository,
	outbox port.OutboxRepository,
	storage port.Storage,
	splitter port.SegmentSplitter,
	extractor port.FrameExtractor,
	timeout time.Duration,
) *SegmentVideoHandler {
	return &SegmentVideoHandler{
		videoRepo:   videoRepo,
		jobRepo:     jobRepo,
		segmentRepo: segmentRepo,
		outbox:      outbox,
		storage:     storage,
		splitter:    splitter,
		extractor:   extractor,
		timeout:     timeout,
	}
}

func (h *SegmentVideoHandler) Handle(ctx context.Context, cmd SegmentVideoCommand) error {
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
	if job.Status == domainjob.StatusSuccess {
		existing, err := h.segmentRepo.ListByVideoID(ctx, video.ID)
		if err != nil {
			return messaging.Retryable(err)
		}
		return h.republishPending(ctx, video, job, existing)
	}

	existing, err := h.segmentRepo.ListByVideoID(ctx, video.ID)
	if err != nil {
		return messaging.Retryable(err)
	}
	if len(existing) > 0 && job.ExpectedSegmentCount > 0 {
		return h.republishPending(ctx, video, job, existing)
	}

	if err := h.markExtracting(ctx, video, job); err != nil {
		return err
	}

	segments, splitErr := h.splitAndStore(ctx, video, job)
	if splitErr != nil {
		_ = h.markFailed(ctx, video, job, publicReason(splitErr))
		if isNonRetryableExtract(splitErr) {
			return messaging.NonRetryable(splitErr)
		}
		return messaging.Retryable(splitErr)
	}

	return h.commitSegments(ctx, video, job, segments)
}

func (h *SegmentVideoHandler) load(ctx context.Context, videoID uuid.UUID) (*domainvideo.Video, *domainjob.Job, error) {
	video, err := h.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		return nil, nil, messaging.Retryable(err)
	}
	if video == nil {
		return nil, nil, messaging.NonRetryable(domainvideo.ErrVideoNotFound)
	}
	job, err := h.jobRepo.GetByVideoIDAndType(ctx, video.ID, domainjob.TypeSegment)
	if err != nil {
		return nil, nil, messaging.Retryable(err)
	}
	if job == nil {
		return nil, nil, messaging.NonRetryable(errors.New("segment job not found"))
	}
	return video, job, nil
}

func (h *SegmentVideoHandler) markExtracting(ctx context.Context, video *domainvideo.Video, job *domainjob.Job) error {
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

func (h *SegmentVideoHandler) splitAndStore(
	ctx context.Context,
	video *domainvideo.Video,
	job *domainjob.Job,
) ([]*domainsegment.Segment, error) {
	tmp, err := os.CreateTemp("", "video-segment-src-*")
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

	durationMs, err := h.splitter.ProbeDurationMs(ctx, tmp.Name())
	if err != nil {
		return nil, messaging.NonRetryable(err)
	}
	durationSec := float64(durationMs) / 1000.0

	plan := domainsegment.PlanSegments(durationSec, domainsegment.PlanConfig{
		SegmentDurationSeconds:          job.SegmentDurationSeconds,
		MinVideoDurationForSplitSeconds: job.MinVideoDurationForSplitSeconds,
		MaxSegments:                     job.MaxSegments,
	})

	if plan.SkipSplit {
		seg := domainsegment.New(video.ID, job.ID, 0, 0, durationMs, video.StorageKey)
		return []*domainsegment.Segment{seg}, nil
	}

	outDir, err := os.MkdirTemp("", "video-segments-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(outDir)

	files, err := h.splitter.Split(ctx, tmp.Name(), plan.EffectiveDurationSeconds, outDir)
	if err != nil {
		return nil, messaging.NonRetryable(err)
	}

	segments := make([]*domainsegment.Segment, 0, len(files))
	for _, file := range files {
		key := domainvideo.NewSegmentStorageKey(video.ID, file.Index)
		f, err := os.Open(file.Path)
		if err != nil {
			return nil, err
		}
		info, err := f.Stat()
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		if err := h.storage.Put(ctx, key, f, info.Size(), segmentContentType); err != nil {
			_ = f.Close()
			return nil, err
		}
		_ = f.Close()
		segments = append(segments, domainsegment.New(
			video.ID,
			job.ID,
			file.Index,
			file.OffsetMs,
			file.DurationMs,
			key,
		))
	}
	if len(segments) == 0 {
		return nil, messaging.NonRetryable(fmt.Errorf("no segments produced"))
	}
	return segments, nil
}

func (h *SegmentVideoHandler) storeThumbnail(ctx context.Context, video *domainvideo.Video, videoPath string) error {
	if h.extractor == nil {
		return nil
	}
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

func (h *SegmentVideoHandler) commitSegments(
	ctx context.Context,
	video *domainvideo.Video,
	job *domainjob.Job,
	segments []*domainsegment.Segment,
) error {
	job.SetSegmentProgress(len(segments), 0)
	job.MarkSuccess()

	for _, seg := range segments {
		video.RecordSegmentReady(job.ID, seg.ID, seg.SegmentIndex, seg.OffsetMs, seg.DurationMs, seg.StorageKey)
	}

	return h.videoRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.segmentRepo.SaveAll(txCtx, segments); err != nil {
			return err
		}
		if err := h.jobRepo.Update(txCtx, job); err != nil {
			return err
		}
		frameJob, err := h.jobRepo.GetByVideoIDAndType(txCtx, video.ID, domainjob.TypeFrame)
		if err != nil {
			return err
		}
		if frameJob == nil {
			frameJob = domainjob.NewFrameJob(video.ID)
			frameJob.SetSegmentProgress(len(segments), 0)
			if err := h.jobRepo.Save(txCtx, frameJob); err != nil {
				return err
			}
		} else {
			frameJob.SetSegmentProgress(len(segments), frameJob.CompletedSegmentCount)
			if frameJob.Status == domainjob.StatusPending {
				frameJob.MarkProcessing()
			}
			if err := h.jobRepo.Update(txCtx, frameJob); err != nil {
				return err
			}
		}
		if err := h.videoRepo.Update(txCtx, video); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, video.PullEvents())
	})
}

func (h *SegmentVideoHandler) republishPending(
	ctx context.Context,
	video *domainvideo.Video,
	job *domainjob.Job,
	segments []*domainsegment.Segment,
) error {
	pending := 0
	for _, seg := range segments {
		if seg.Status == domainsegment.StatusPending || seg.Status == domainsegment.StatusFailed {
			video.RecordSegmentReady(job.ID, seg.ID, seg.SegmentIndex, seg.OffsetMs, seg.DurationMs, seg.StorageKey)
			pending++
		}
	}
	if pending == 0 {
		return nil
	}
	return h.videoRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		return h.outbox.StoreEvents(txCtx, video.PullEvents())
	})
}

func (h *SegmentVideoHandler) markFailed(
	ctx context.Context,
	video *domainvideo.Video,
	job *domainjob.Job,
	reason string,
) error {
	job.MarkFailed(reason)
	if err := video.MarkExtractionFailed(job.ID, job.Type, job.Status, reason); err != nil {
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