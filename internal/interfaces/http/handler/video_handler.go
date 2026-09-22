package handler

import (
	"errors"
	"log"

	cmdvideo "go-api/internal/application/command/video"
	queryvideo "go-api/internal/application/query/video"
	"go-api/internal/domain/paginate"
	domainvideo "go-api/internal/domain/video"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/dto"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/validation"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type VideoHandler struct {
	requestUploadURLHandler videoRequestUploadURLHandler
	getVideoByIDHandler     videoGetByIDHandler
	listVideosHandler       videoListHandler
}

func NewVideoHandler(
	requestUploadURLHandler videoRequestUploadURLHandler,
	getVideoByIDHandler videoGetByIDHandler,
	listVideosHandler videoListHandler,
) *VideoHandler {
	return &VideoHandler{
		requestUploadURLHandler: requestUploadURLHandler,
		getVideoByIDHandler:     getVideoByIDHandler,
		listVideosHandler:       listVideosHandler,
	}
}

func (h *VideoHandler) RequestUploadURL(c fiber.Ctx) error {
	user, err := httpctx.GetUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Unauthorized",
		})
	}

	var req dto.RequestUploadURLRequest
	if err := validation.BindBody(c, &req); err != nil {
		return err
	}

	result, err := h.requestUploadURLHandler.Handle(c.Context(), cmdvideo.RequestUploadURLCommand{
		UserID:      user.ID,
		Filename:    req.Filename,
		ContentType: req.ContentType,
		SizeBytes:   req.SizeBytes,
	})
	if err != nil {
		if errors.Is(err, domainvideo.ErrInvalidFilename) || errors.Is(err, domainvideo.ErrUnsupportedType) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Unsupported video type",
			})
		}
		log.Printf("failed to request upload url: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to generate upload url",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(presenter.NewRequestUploadURLResponse(result))
}

func (h *VideoHandler) List(c fiber.Ctx) error {
	user, err := httpctx.GetUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Unauthorized",
		})
	}

	var listQuery paginate.PaginateQuery
	if err := c.Bind().Query(&listQuery); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid query parameters",
		})
	}

	sortBy, orderBy := listQuery.SortBy, listQuery.OrderBy
	listQuery.Normalize()
	if sortBy == "" {
		listQuery.SortBy = "created_at"
	}
	if orderBy == "" {
		listQuery.OrderBy = paginate.OrderByDesc
	}

	views, total, err := h.listVideosHandler.Handle(c.Context(), queryvideo.ListVideosQuery{
		UserID: user.ID,
		Query:  listQuery,
	})
	if err != nil {
		log.Printf("failed to list videos: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to list videos",
		})
	}

	return c.Status(fiber.StatusOK).JSON(paginate.NewPaginateResponse(
		presenter.NewVideoListResponseFromViews(views),
		int(total),
		listQuery,
	))
}

func (h *VideoHandler) GetByID(c fiber.Ctx) error {
	user, err := httpctx.GetUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Unauthorized",
		})
	}

	videoID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid video id",
		})
	}

	view, err := h.getVideoByIDHandler.Handle(c.Context(), queryvideo.GetVideoByIDQuery{
		ID:     videoID,
		UserID: user.ID,
	})
	if err != nil {
		if errors.Is(err, domainvideo.ErrVideoNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Video not found",
			})
		}
		log.Printf("failed to get video %s: %v", videoID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to get video",
		})
	}

	return c.Status(fiber.StatusOK).JSON(presenter.NewVideoDetailResponseFromView(*view))
}
