package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"go-api/internal/interfaces/http/dto"

	"github.com/gofiber/fiber/v3"
)

type VideoWebhookMiddleware struct {
	secret string
}

func NewVideoWebhookMiddleware(secret string) *VideoWebhookMiddleware {
	return &VideoWebhookMiddleware{secret: secret}
}

func (m *VideoWebhookMiddleware) Protected() fiber.Handler {
	return func(c fiber.Ctx) error {
		if m.secret == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "invalid signature",
			})
		}

		signature := c.Get("X-Signature")
		if !validHMAC(m.secret, c.Body(), signature) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "invalid signature",
			})
		}

		var req dto.VideoIngestRequest
		if err := json.Unmarshal(c.Body(), &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Invalid request body",
			})
		}

		c.Locals("payload", req)
		return c.Next()
	}
}

func validHMAC(secret string, body []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
