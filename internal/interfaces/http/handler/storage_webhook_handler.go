package handler

import (
	"errors"
	"log"
	"strings"

	cmdvideo "go-api/internal/application/command/video"
	domainvideo "go-api/internal/domain/video"
	"go-api/internal/interfaces/http/dto"
	"go-api/internal/interfaces/http/validation"

	"github.com/gofiber/fiber/v3"
)

type StorageWebhookHandler struct {
	mediaBucket          string
	confirmUploadHandler storageConfirmUploadHandler
}

func NewStorageWebhookHandler(
	mediaBucket string,
	confirmUploadHandler storageConfirmUploadHandler,
) *StorageWebhookHandler {
	return &StorageWebhookHandler{
		mediaBucket:          mediaBucket,
		confirmUploadHandler: confirmUploadHandler,
	}
}

func (h *StorageWebhookHandler) ObjectCreated(c fiber.Ctx) error {
	event, ok := c.Locals("payload").(dto.ObjectCreatedEvent)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}
	if err := validation.Struct(c, &event); err != nil {
		return err
	}

	for _, record := range event.Records {
		if err := h.processRecord(c, record); err != nil {
			if errors.Is(err, domainvideo.ErrVideoNotFound) ||
				errors.Is(err, domainvideo.ErrInvalidTransition) ||
				errors.Is(err, domainvideo.ErrVideoTooLarge) {
				continue
			}
			log.Printf("storage webhook: failed to confirm upload: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Failed to confirm upload",
			})
		}
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *StorageWebhookHandler) processRecord(c fiber.Ctx, record dto.ObjectCreatedRecord) error {
	if !strings.HasPrefix(record.EventName, "s3:ObjectCreated") {
		return nil
	}
	if record.S3.Bucket.Name != h.mediaBucket {
		return nil
	}

	decodedKey, err := domainvideo.DecodeObjectKey(record.S3.Object.Key)
	if err != nil {
		log.Printf("storage webhook: invalid object key %q: %v", record.S3.Object.Key, err)
		return nil
	}
	if domainvideo.IsFrameObjectKey(decodedKey) || domainvideo.IsThumbnailObjectKey(decodedKey) {
		return nil
	}

	if _, err := domainvideo.VideoIDFromStorageKey(decodedKey); err != nil {
		log.Printf("storage webhook: ignored object key %q: %v", decodedKey, err)
		return nil
	}

	return h.confirmUploadHandler.Handle(c.Context(), cmdvideo.ConfirmUploadCommand{
		StorageKey:  decodedKey,
		ContentType: record.S3.Object.ContentType,
		SizeBytes:   record.S3.Object.Size,
	})
}
