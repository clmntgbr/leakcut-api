# Realtime

## Overview

**Centrifugo** delivers WebSocket updates. The API does not publish realtime events from HTTP handlers — the preferred path is:

```
outbox → RabbitMQ → worker handler → Centrifugo publisher
```

Realtime event `type` uses `entity.action` (e.g. `user.created`), not the versioned domain type (`user.created.v1`).

## HTTP routes

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/realtime/connection` | Connection token / channel / WS URL |

Requires authentication.

## Event examples

| Realtime type | When |
|---------------|------|
| `user.created` | User created (Clerk webhook / signup) |
| `user.updated` | User updated |
| `user.deleted` | User deleted |

Video realtime types (`video.created`, `video.updated`) are reserved. Handlers are registered on the worker but **do not publish to Centrifugo yet** — the frontend must poll `GET /api/videos/:id`.

## Code map

| Layer | Location |
|-------|----------|
| HTTP | `internal/interfaces/http/handler/realtime_handler.go` |
| Ports | `internal/domain/port/realtime.go` |
| Helpers | `internal/application/realtime/` |
| Adapter | `internal/infrastructure/centrifugo/` |
| Event publish | `internal/application/event/user/publish_realtime.go` |
| Event publish (video, no-op) | `internal/application/event/video/publish_realtime.go` |
