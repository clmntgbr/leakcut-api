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
| `job.updated` | Job moved to `extracting_frames`, `frames_ready`, `ocr_processing`, `ocr_ready`, `ocr_failed`, or `failed` |

Video/job events are published only to the owner (`users:<userId>`). Webhook-ingested videos without a user are not pushed.

```json
{ "type": "video.created", "videoId": "...", "originalFilename": "demo.mp4", "status": "pending_upload", "occurredAt": "..." }
{ "type": "video.uploaded", "videoId": "...", "status": "extraction_queued", "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "extract_frames", "status": "extracting_frames", "videoStatus": "extracting", "frameCount": 0, "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "extract_frames", "status": "frames_ready", "videoStatus": "frames_ready", "frameCount": 12, "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "ocr", "status": "ocr_processing", "videoStatus": "ocr_processing", "expectedFrameCount": 12, "ocrCompletedCount": 0, "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "ocr", "status": "ocr_ready", "videoStatus": "ocr_ready", "frameCount": 12, "expectedFrameCount": 12, "ocrCompletedCount": 12, "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "ocr", "status": "ocr_failed", "videoStatus": "ocr_failed", "failureReason": "paddleocr unavailable", "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "extract_frames", "status": "failed", "videoStatus": "extraction_failed", "failureReason": "unreadable video", "occurredAt": "..." }
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
