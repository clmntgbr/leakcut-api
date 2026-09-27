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
| `video.created` | Video created (`pending`) |
| `video.uploaded` | Upload confirmed (`processing`) |
| `job.updated` | Job moved to `processing`, `success`, or `failed` (also after each OCR frame, with `ocrCompletedCount`). Use `jobType` (`segment` / `frame` / `ocr` / `classify`) to know which stage. `videoStatus` is the coarse video status (`pending` / `processing` / `success` / `failed`). |

Video/job events are published only to the owner (`users:<userId>`). Webhook-ingested videos without a user are not pushed.

```json
{ "type": "video.created", "videoId": "...", "originalFilename": "demo.mp4", "status": "pending", "occurredAt": "..." }
{ "type": "video.uploaded", "videoId": "...", "status": "processing", "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "segment", "status": "processing", "videoStatus": "processing", "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "segment", "status": "success", "videoStatus": "processing", "expectedSegmentCount": 3, "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "frame", "status": "processing", "videoStatus": "processing", "frameCount": 0, "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "frame", "status": "success", "videoStatus": "processing", "frameCount": 12, "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "ocr", "status": "processing", "videoStatus": "processing", "expectedFrameCount": 12, "ocrCompletedCount": 0, "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "ocr", "status": "success", "videoStatus": "processing", "frameCount": 12, "expectedFrameCount": 12, "ocrCompletedCount": 12, "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "ocr", "status": "failed", "videoStatus": "failed", "failureReason": "ocr service unavailable", "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "classify", "status": "processing", "videoStatus": "processing", "expectedFrameCount": 12, "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "classify", "status": "success", "videoStatus": "success", "frameCount": 12, "occurredAt": "..." }
{ "type": "job.updated", "id": "...", "videoId": "...", "jobType": "frame", "status": "failed", "videoStatus": "failed", "failureReason": "unreadable video", "occurredAt": "..." }
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
