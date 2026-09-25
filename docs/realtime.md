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
| `video.created` | Video created (`pending_upload`) |
| `video.uploaded` | Upload confirmed (`extraction_queued`) |
| `job.updated` | Job moved to `processing`, `success`, or `failed` (also after each OCR batch, with `ocrCompletedCount`). Use `jobType` (`frame` / `ocr` / `classify`) to know which stage. |

Video/job events are published only to the owner (`users:<userId>`). Webhook-ingested videos without a user are not pushed.

```json
{ "type": "video.created", "videoId": "...", "originalFilename": "demo.mp4", "status": "pending_upload", "occurredAt": "..." }
{ "type": "video.uploaded", "videoId": "...", "status": "extraction_queued", "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "frame", "status": "processing", "videoStatus": "extracting", "frameCount": 0, "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "frame", "status": "success", "videoStatus": "frames_ready", "frameCount": 12, "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "ocr", "status": "processing", "videoStatus": "ocr_processing", "expectedFrameCount": 12, "ocrCompletedCount": 0, "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "ocr", "status": "success", "videoStatus": "ocr_ready", "frameCount": 12, "expectedFrameCount": 12, "ocrCompletedCount": 12, "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "ocr", "status": "failed", "videoStatus": "ocr_failed", "failureReason": "ocr service unavailable", "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "classify", "status": "processing", "videoStatus": "classifying", "expectedFrameCount": 12, "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "classify", "status": "success", "videoStatus": "classified", "frameCount": 12, "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "frame", "status": "failed", "videoStatus": "extraction_failed", "failureReason": "unreadable video", "occurredAt": "..." }
```

Use `occurredAt` to ignore a stale `job.updated` if events arrive out of order.

## Code map

| Layer | Location |
|-------|----------|
| HTTP | `internal/interfaces/http/handler/realtime_handler.go` |
| Ports | `internal/domain/port/realtime.go` |
| Helpers | `internal/application/realtime/` |
| Adapter | `internal/infrastructure/centrifugo/` |
| Event publish | `internal/application/event/user/publish_realtime.go` |
| Event publish (video / job) | `internal/application/event/video/publish_realtime.go` |
