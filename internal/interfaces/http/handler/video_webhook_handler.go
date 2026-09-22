package handler

import (
	"errors"
	"log"
	"path/filepath"

	cmdvideo "go-api/internal/application/command/video"
	domainvideo "go-api/internal/domain/video"
	"go-api/internal/interfaces/http/dto"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/validation"

	"github.com/gofiber/fiber/v3"
)

type VideoWebhookHandler struct {
	requestIngestHandler videoRequestIngestHandler
}

func NewVideoWebhookHandler(requestIngestHandler videoRequestIngestHandler) *VideoWebhookHandler {
	return &VideoWebhookHandler{requestIngestHandler: requestIngestHandler}
}

func (h *VideoWebhookHandler) Ingest(c fiber.Ctx) error {
	req, ok := c.Locals("payload").(dto.VideoIngestRequest)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}
	if err := validation.Struct(c, &req); err != nil {
		return err
	}

	filename := req.Filename
	if filename == "" {
		filename = filepath.Base(req.VideoURL)
	}

	result, err := h.requestIngestHandler.Handle(c.Context(), cmdvideo.RequestIngestCommand{
		RemoteURL:   req.VideoURL,
		Filename:    filename,
		ContentType: req.ContentType,
		SizeBytes:   req.SizeBytes,
	})
	if err != nil {
		if errors.Is(err, domainvideo.ErrInvalidFilename) || errors.Is(err, domainvideo.ErrUnsupportedType) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Unsupported video type",
			})
		}
		log.Printf("failed to ingest remote video: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to ingest video",
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(presenter.NewVideoIngestResponse(result))
}
