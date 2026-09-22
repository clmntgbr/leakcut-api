package video

import (
	"context"
	"errors"
	"io"
	"os"

	"go-api/internal/application/messaging"
	"go-api/internal/domain/port"
	domainvideo "go-api/internal/domain/video"

	"github.com/google/uuid"
)

type IngestRemoteCommand struct {
	VideoID   uuid.UUID
	RemoteURL string
}

type IngestRemoteHandler struct {
	videoRepo     domainvideo.VideoWriteRepository
	fetcher       port.RemoteFetcher
	storage       port.Storage
	confirmUpload *ConfirmUploadHandler
	maxSize       int64
}

func NewIngestRemoteHandler(
	videoRepo domainvideo.VideoWriteRepository,
	fetcher port.RemoteFetcher,
	storage port.Storage,
	confirmUpload *ConfirmUploadHandler,
	maxSize int64,
) *IngestRemoteHandler {
	return &IngestRemoteHandler{
		videoRepo:     videoRepo,
		fetcher:       fetcher,
		storage:       storage,
		confirmUpload: confirmUpload,
		maxSize:       maxSize,
	}
}

func (h *IngestRemoteHandler) Handle(ctx context.Context, cmd IngestRemoteCommand) error {
	video, err := h.videoRepo.GetByID(ctx, cmd.VideoID)
	if err != nil {
		return messaging.Retryable(err)
	}
	if video == nil {
		return messaging.NonRetryable(domainvideo.ErrVideoNotFound)
	}
	if video.Status != domainvideo.StatusPendingUpload {
		return nil
	}

	remote, err := h.fetcher.Fetch(ctx, cmd.RemoteURL)
	if err != nil {
		if errors.Is(err, domainvideo.ErrRemoteURLForbidden) {
			return messaging.NonRetryable(err)
		}
		return messaging.Retryable(err)
	}
	defer remote.Body.Close()

	tmp, err := os.CreateTemp("", "video-ingest-*")
	if err != nil {
		return messaging.Retryable(err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	limited := io.Reader(remote.Body)
	if h.maxSize > 0 {
		limited = io.LimitReader(remote.Body, h.maxSize+1)
	}
	size, err := io.Copy(tmp, limited)
	if err != nil {
		return messaging.Retryable(err)
	}
	if h.maxSize > 0 && size > h.maxSize {
		return messaging.NonRetryable(domainvideo.ErrVideoTooLarge)
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return messaging.Retryable(err)
	}

	contentType := remote.ContentType
	if contentType == "" {
		contentType = video.ContentType
	}

	if err := h.storage.Put(ctx, video.StorageKey, tmp, size, contentType); err != nil {
		return messaging.Retryable(err)
	}

	if err := h.confirmUpload.Handle(ctx, ConfirmUploadCommand{
		StorageKey:  video.StorageKey,
		ContentType: contentType,
		SizeBytes:   size,
	}); err != nil {
		return messaging.Retryable(err)
	}
	return nil
}
