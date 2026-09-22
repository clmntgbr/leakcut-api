# upload & frame extraction

## Overview

Ingest a video, then the `frame` worker cuts it into retained stills. OCR and classify are downstream — see [ocr](ocr.md) and [classify](classify.md).

Two ingest paths:

1. **Presigned PUT** — the client uploads straight to MinIO. The API never sees the bytes.
2. **Webhook ingest** — a third party sends a remote URL. The worker downloads it into the internal bucket.

```
client → POST /api/videos/upload-url → MinIO PUT
MinIO → POST /webhooks/minio/object-created → outbox video.uploaded.v1
     → queue frame → extract → per-frame video.ocr_frame_requested.v1
                  → video.frames_extracted.v1 (counts + realtime)
```

## HTTP routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/videos/upload-url` | JWT | Presigned PUT (`filename`, `contentType`, `sizeBytes`) |
| `GET` | `/api/videos` | JWT | `PaginateResponse` — `id`, `originalFilename`, `thumbnailUrl`, `status`, `createdAt` |
| `GET` | `/api/videos/:id` | JWT | Full detail (jobs, frames, OCR, findings, signed URLs) |
| `POST` | `/webhooks/minio/object-created` | MinIO token | Confirms `original.mp4` landed |
| `POST` | `/webhooks/videos` | HMAC | Remote URL ingest (`202`) |

## Status

`Video.Status` is the pipeline cursor. Each task has its own `Job` (`UNIQUE (video_id, type)`).

| Video status | When |
|--------------|------|
| `pending_upload` | Presigned URL issued |
| `extraction_queued` | Upload confirmed, `extract_frames` job created |
| `extracting` | `frame` worker started |
| `frames_ready` | Retained frames stored; OCR requests already in flight |
| `extraction_failed` | Unreadable video / ffmpeg / timeout |
| `upload_expired` | No PUT before the URL TTL |

Later statuses (`ocr_*`, `classifying`, `classified`) are owned by the [ocr](ocr.md) and [classify](classify.md) docs.

`GET /api/videos/:id` embeds `jobs[]` and `frames[]`. Each frame carries `ocrText` / `ocrStatus` and optional `finding` (`confidential`, `probability`, `categories[]`). Presigned `videoUrl`, `thumbnailUrl`, and `imageUrl` expire with the storage TTL.

## Frame selection

ffmpeg decodes at `analysis_fps` (not 30 fps). A frame is kept if the grayscale mean-abs-diff vs the **last kept** frame is ≥ `diff_threshold`, or if `max_interval_seconds` elapsed (`fixed_interval`).

| Param | Default | Role |
|-------|---------|------|
| `analysis_fps` | `2` | Decode rate for the diff |
| `diff_threshold` | `0.13` | Keep as `scene_change` |
| `max_interval_seconds` | `15` | Safety net on a static screen |
| `FRAME_MAX_WIDTH_PX` | `1280` | ffmpeg `scale='min(1280,iw)':-2` before upload |

Only retained PNGs are stored. Same pixels for MinIO, OCR, and `GET /videos/:id`.

## Per-frame publish

ffmpeg streams PNGs (`image2pipe`). Each retained frame is uploaded and written to the outbox (`video.ocr_frame_requested.v1`) immediately — OCR does not wait for extract to finish. The main `worker` relay publishes it (~`OUTBOX_POLL_INTERVAL`). Replicas compete on that queue.

When extraction finishes, `video.frames_extracted.v1` carries the retained list and `frameCount` so OCR can set `expectedFrameCount`.

## Storage layout

```
media/
  videos/{video_id}/original.mp4
  videos/{video_id}/thumbnail.jpg
  videos/{video_id}/frames/{index}.png
```

## Errors

- Corrupt / unsupported codec → `extraction_failed`, DLQ after retries.
- Timeout → same, configurable `FRAME_EXTRACTION_TIMEOUT`.
- Redelivery → upsert `(job_id, index)` and overwrite the PNG. Diff is always against the last kept frame.

## Code map

| Piece | Location |
|-------|----------|
| Worker | `cmd/frame` — queue `frame`, routing `video.uploaded.v1` |
| Command | `internal/application/command/video/extract_frames.go` |
| ffmpeg | `internal/infrastructure/video/extractor.go` |
| HTTP | `internal/interfaces/http/handler/video_handler.go` |
| Webhooks | `video_webhook_handler.go`, MinIO object-created handler |

`frame` has no `container_name` — `--scale frame=2` is allowed. The main `worker` stays singleton (outbox relay).
