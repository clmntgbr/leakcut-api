package middleware

import (
	"crypto/subtle"
	"encoding/json"

	"go-api/internal/interfaces/http/dto"

	"github.com/gofiber/fiber/v3"
)

type StorageWebhookMiddleware struct {
	secret string
}

func NewStorageWebhookMiddleware(secret string) *StorageWebhookMiddleware {
	return &StorageWebhookMiddleware{secret: secret}
}

func (m *StorageWebhookMiddleware) Protected() fiber.Handler {
	return func(c fiber.Ctx) error {
		if m.secret == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "invalid authorization",
			})
		}

		authHeader := c.Get("Authorization")
		expectedBearer := "Bearer " + m.secret
		if subtle.ConstantTimeCompare([]byte(authHeader), []byte(expectedBearer)) != 1 &&
			subtle.ConstantTimeCompare([]byte(authHeader), []byte(m.secret)) != 1 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "invalid authorization",
			})
		}

		var event dto.ObjectCreatedEvent
		if err := json.Unmarshal(c.Body(), &event); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Invalid request body",
			})
		}

		c.Locals("payload", event)
		return c.Next()
	}
}
