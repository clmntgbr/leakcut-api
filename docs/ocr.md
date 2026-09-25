# ocr

## Overview

Read text from each retained frame. Inference is a Python AMQP worker (`cmd/ocr`). Persistence stays on the main Go `worker`. There is no HTTP OCR API.

```
extract upserts frames → outbox video.frames_extracted.v1 (one event)
     → queue ocr (one video per message) → download JPEGs → RapidOCR
     → video.ocr_batch_completed.v1 (chunks of results)
     → main worker persist → job.updated (ocrCompletedCount)
     → when all rows exist → video.frames_ocr_completed.v1 → classify
```

One queue message per **video** after extract finishes. Scale OCR across videos; within a video, `OCR_INFER_CONCURRENCY` runs frames in parallel.

## Queue

| Setting | Value |
|---------|-------|
| Queue | `OCR_QUEUE` (`ocr`) |
| Routing | `OCR_ROUTING_KEY` (`video.frames_extracted.v1`) |
| Exchange | `domain.events` |
| Binary | `cmd/ocr/worker.py` (RapidOCR + onnxruntime) |

On startup the worker unbinds the legacy `video.ocr_frame_requested.v1` key.

## Jobs

| Job type | Owner | Role |
|----------|-------|------|
| `frame` | `frame` | Writes frames only; one `video.frames_extracted.v1` at the end |
| `ocr` | main `worker` | Created on `video.frames_extracted.v1`; counters only |
| `classify` | `classify` | Started after `video.frames_ocr_completed.v1` |

All jobs share the same statuses: `pending` → `processing` → `success` / `failed`. Distinguish stages with `type`.

`expectedFrameCount` comes from extract. `ocrCompletedCount` increments on each persist. Realtime `job.updated` is republished after every batch.

`markReady` waits until the `frame` job is `success` **and** every frame has an `ocrs` row. Early persist while extract is still running only upserts rows.

## Engine

RapidOCR + ONNX Runtime (CPU). Paddle was dropped after aarch64 SIGSEGV.

| Env | Default | Role |
|-----|---------|------|
| `OCR_CONCURRENCY` | `1` | RabbitMQ prefetch (unacked messages per replica) |
| `OCR_INFER_CONCURRENCY` | `2` | Preloaded RapidOCR engines (consume callback is one frame at a time) |
| `OCR_INTRA_OP_THREADS` | `2` | onnxruntime threads per inference |
| `OCR_BATCH_SIZE` | `8` | Unused (one frame per message) |

Resize happens at extract (`FRAME_MAX_WIDTH_PX`), not here. The worker downloads the stored JPEG as-is.

Empty OCR text is a success with `text=""`. Classify will `skip` that frame.

## Scale

```bash
docker compose -f compose.dev.yaml up --scale ocr=2
```

`ocr` has no `container_name`. The main `worker` must stay a singleton (outbox + expire).

Throughput is mostly `--scale ocr=N`. `OCR_INFER_CONCURRENCY` only helps if prefetch > 1 **and** inferences run concurrently; the current consume loop is one message at a time. Watch CPU: `replicas × intra_op` should stay near the host core count.

## Schema

`ocrs`: `UNIQUE (frame_id)`. Redelivery overwrites. Status `success` or `failed`. `lines` is JSONB: each line is `{ text, confidence, box: [{x,y}×4] }` in pixels of the stored JPEG. Classify still receives only joined `text`. `GET /videos/:id` returns `ocrText` and `ocrLines`.

## Code map

| Piece | Location |
|-------|----------|
| Python worker | `cmd/ocr/worker.py`, `cmd/ocr/engine.py` |
| Start OCR job | `internal/application/command/video/ocr_frames.go` (`Start`) |
| Persist batch | same file (`PersistBatch`) |
| Event | `internal/application/event/video/` (`video.ocr_*`) |
| Read | `internal/infrastructure/persistence/read/video_query_repository.go` |

See [performance](performance.md) for frame-count / replica sizing.
