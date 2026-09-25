# upload & frame extraction

## Overview

Ingest a video, then the `frame` worker cuts it into retained stills. OCR and classify are downstream — see [ocr](ocr.md) and [classify](classify.md).

Two ingest paths:

1. **Presigned PUT** — the client uploads straight to MinIO. The API never sees the bytes.
2. **Webhook ingest** — a third party sends a remote URL. The worker downloads it into the internal bucket.

```
client → POST /api/videos/upload-url → MinIO PUT
MinIO → POST /webhooks/minio/object-created → outbox video.uploaded.v1
     → queue frame → extract (upload + upsert frames only)
                  → video.frames_extracted.v1 (one outbox event)
     → queue ocr → OCR all frames → video.ocr_batch_completed.v1
```

## HTTP routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/videos/upload-url` | JWT | Presigned PUT (`filename`, `contentType`, `sizeBytes`) |
| `GET` | `/api/videos` | JWT | `PaginateResponse` — `id`, `originalFilename`, `thumbnailUrl`, `status`, `createdAt` |
| `GET` | `/api/videos/:id` | JWT | Full detail (jobs, frames, OCR, classifications, signed URLs) |
| `POST` | `/webhooks/minio/object-created` | MinIO token | Confirms `original.mp4` landed |
| `POST` | `/webhooks/videos` | HMAC | Remote URL ingest (`202`) |

## Status

`Video.Status` is the pipeline cursor. Each task has its own `Job` (`UNIQUE (video_id, type)`).

| Video status | When |
|--------------|------|
| `pending_upload` | Presigned URL issued |
| `extraction_queued` | Upload confirmed, `frame` job created |
| `extracting` | `frame` worker started |
| `frames_ready` | Retained frames stored; OCR may already be running |
| `extraction_failed` | Unreadable video / ffmpeg / timeout |
| `upload_expired` | No PUT before the URL TTL |

Jobs use shared statuses (`pending` / `processing` / `success` / `failed`) plus `type` (`frame` / `ocr` / `classify`). Video statuses above are the pipeline cursor; later ones (`ocr_*`, `classifying`, `classified`) are in [ocr](ocr.md) and [classify](classify.md).

`GET /api/videos/:id` embeds `jobs[]` and `frames[]`. Each frame carries `ocrText` / `ocrLines` (`box` in JPEG pixels) / `ocrStatus` and optional `classification` (`confidential`, `probability`, `categories[]`). Presigned `videoUrl`, `thumbnailUrl`, and `imageUrl` expire with the storage TTL.

## Frame selection

ffmpeg decodes at `analysis_fps` (not 30 fps). Each candidate gets a perceptual hash (`pHash`). A frame is kept if the Hamming distance vs the **last kept** frame is ≥ `phash_distance_threshold` (`scene_change`), or if `max_interval_seconds` elapsed (`fixed_interval`). Same image bytes always produce the same hash — retries stay deterministic.

| Param | Default | Role |
|-------|---------|------|
| `analysis_fps` | `2` | Decode rate for the pHash pass |
| `phash_distance_threshold` | `14` | Keep as `scene_change` (Hamming) |
| `max_interval_seconds` | `15` | Safety net on a static screen |
| `FRAME_MAX_WIDTH_PX` | `960` | ffmpeg `scale='min(960,iw)':-2` before upload (JPEG) |
| `FRAME_UPLOAD_CONCURRENCY` | `4` | Parallel MinIO puts while ffmpeg keeps decoding |

Only retained JPEGs are stored. Same pixels for MinIO, OCR, and `GET /videos/:id`.

## Per-frame publish

ffmpeg streams JPEGs (`image2pipe` / mjpeg). Each retained frame is uploaded (pool of `FRAME_UPLOAD_CONCURRENCY`) and upserted in Postgres — **no outbox per frame**. When extraction finishes, one `video.frames_extracted.v1` carries the retained list and `frameCount`.

OCR starts from that video-level event (see [ocr](ocr.md)).

## Storage layout

```
media/
  videos/{video_id}/original.mp4
  videos/{video_id}/thumbnail.jpg
  videos/{video_id}/frames/{index}.jpg
```

## Errors

- Corrupt / unsupported codec → `extraction_failed`, DLQ after retries.
- Timeout → same, configurable `FRAME_EXTRACTION_TIMEOUT`.
- Redelivery → upsert `(video_id, index)` and overwrite the JPEG. pHash is always against the last kept frame.

## Code map

| Piece | Location |
|-------|----------|
| Worker | `cmd/frame` — queue `frame`, routing `video.uploaded.v1` |
| Command | `internal/application/command/video/extract_frames.go` |
| ffmpeg | `internal/infrastructure/video/extractor.go` |
| HTTP | `internal/interfaces/http/handler/video_handler.go` |
| Webhooks | `video_webhook_handler.go`, MinIO object-created handler |

`frame` has no `container_name` — `--scale frame=2` is allowed. The main `worker` stays singleton (outbox relay).
