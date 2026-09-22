package main

import (
	"go-api/cmd/api/di"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
)

func setupRoutes(app *fiber.App, container *di.Container) {
	setupHealthChecks(app)
	setupWebhooks(app, container)
	setupAPIRoutes(app, container)
}

func setupWebhooks(app *fiber.App, container *di.Container) {
	webhooks := app.Group("/webhooks")
	webhooks.Post("/clerk", container.UserWebhookMiddleware.Protected(), container.UserWebhookHandler.Execute)
	webhooks.Post("/minio/object-created", container.StorageWebhookMiddleware.Protected(), container.StorageWebhookHandler.ObjectCreated)
	webhooks.Post("/videos", container.VideoWebhookMiddleware.Protected(), container.VideoWebhookHandler.Ingest)
}

func setupHealthChecks(app *fiber.App) {
	app.Get(healthcheck.LivenessEndpoint, healthcheck.New())
	app.Get(healthcheck.ReadinessEndpoint, healthcheck.New())
	app.Get(healthcheck.StartupEndpoint, healthcheck.New())
}

func setupAPIRoutes(app *fiber.App, container *di.Container) {
	public := app.Group("/api")

	protected := public.Group("", container.AuthenticateMiddleware.Protected())
	setupUserRoutes(protected, container)
	setupVideoRoutes(protected, container)
	setupRealtimeRoutes(protected, container)
}

func setupVideoRoutes(api fiber.Router, container *di.Container) {
	api.Post("/videos/upload-url", container.VideoHandler.RequestUploadURL)
	api.Get("/videos", container.VideoHandler.List)
	api.Get("/videos/:id", container.VideoHandler.GetByID)
}

func setupRealtimeRoutes(api fiber.Router, container *di.Container) {
	api.Get("/realtime/connection", container.RealtimeHandler.GetConnection)
}

func setupUserRoutes(api fiber.Router, container *di.Container) {
	api.Get("/users/me", container.UserHandler.GetUser)
}
