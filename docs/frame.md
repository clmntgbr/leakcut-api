# upload & frame extraction

## Overview

Ingest a video, then **segment** slices it and **frame** workers extract stills in parallel. OCR and classify are downstream — see [segment](segment.md), [ocr](ocr.md) and [classify](classify.md).

Two ingest paths:

1. **Presigned PUT** — the client uploads straight to MinIO. The API never sees the bytes.
2. **Webhook ingest** — a third party sends a remote URL. The worker downloads it into the internal bucket.

```
client → POST /api/videos/upload-url → MinIO PUT
MinIO → POST /webhooks/minio/object-created → outbox video.uploaded.v1
     → queue segment → plan/split → video.segment_ready.v1 × N
     → queue frame → extract per segment → video.ocr_frame_requested.v1 × frames (immédiat)
                  → when all segments done → video.frames_extracted.v1 (status only)
     → queue ocr → one frame → video.ocr_frame_completed.v1
                  → classify_frame_requested.v1 × 1
     → queue classify → one frame → when all done → video.frames_classified.v1
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

Video and jobs share the same statuses: `pending` → `processing` → `success` / `failed`.

| Video status | When |
|--------------|------|
| `pending` | Presigned URL issued / waiting for upload |
| `processing` | Upload confirmed; pipeline running |
| `success` | Classify finished |
| `failed` | Any job failed, or upload expired |

Each task has its own `Job` (`UNIQUE (video_id, type)`): `segment` → `frame` → `ocr` → `classify`. `GET /api/videos/:id` exposes `jobStatus` / `jobId` for the **first job that is not yet `success`** (pipeline order). Use `jobs[]` + `jobType` for per-stage detail (`createdAt` / `startedAt` / `finishedAt`).

`GET /api/videos/:id` also embeds `frames[]`. Each frame carries `ocrText` / `ocrLines` (`box` in JPEG pixels) / `ocrStatus`, optional `classification`. All frame images are kept. Presigned `videoUrl`, `thumbnailUrl`, and `imageUrl` expire with the storage TTL.

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

ffmpeg streams JPEGs (`image2pipe` / mjpeg). Each retained frame is uploaded (pool of `FRAME_UPLOAD_CONCURRENCY`), upserted, then immediately enqueued as `video.ocr_frame_requested.v1`. When all segments finish, one `video.frames_extracted.v1` marks extract complete (realtime) — it does not trigger OCR.

OCR starts from that video-level event (see [ocr](ocr.md)).

## Storage layout

```
media/
  videos/{video_id}/original.mp4
  videos/{video_id}/thumbnail.jpg
  videos/{video_id}/segments/segment_{ii}.mp4   # temporary when split
  frames/{video_id}/{segment:02d}_{local:04d}.jpg
```

## Errors

- Corrupt / unsupported codec → `extraction_failed`, DLQ after retries.
- Timeout → same, configurable `FRAME_EXTRACTION_TIMEOUT` / `SEGMENT_TIMEOUT`.
- Redelivery → upsert `(video_id, segment_index, index)` and overwrite the JPEG. pHash resets per segment.

## Code map

| Piece | Location |
|-------|----------|
| Segment worker | `cmd/segment` — queue `segment`, routing `video.uploaded.v1` |
| Frame worker | `cmd/frame` — queue `frame`, routing `video.segment_ready.v1` |
| Commands | `segment_video.go`, `extract_frames.go` |
| ffmpeg | `splitter.go`, `extractor.go` |
| HTTP | `internal/interfaces/http/handler/video_handler.go` |
| Webhooks | `video_webhook_handler.go`, MinIO object-created handler |

`frame` / `segment` have no `container_name` — `--scale frame=2` (and segment) is allowed. The main `worker` stays singleton (outbox relay). See [segment](segment.md).
