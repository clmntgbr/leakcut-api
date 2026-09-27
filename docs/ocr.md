# ocr

## Overview

Read text from each retained frame. Inference is a Python AMQP worker (`cmd/ocr`). Persistence stays on the main Go `worker`. There is no HTTP OCR API.

```
extract upserts frames → outbox video.frames_extracted.v1 (one event)
     → main worker Start OCR → video.ocr_processing.v1
                              + video.ocr_frame_requested.v1 × N (one per frame)
     → queue ocr (competing consumers) → download JPEG → RapidOCR
     → video.ocr_batch_completed.v1 (one result per message)
     → main worker persist → job.updated (ocrCompletedCount)
     → when all rows exist → video.frames_ocr_completed.v1 → classify
```

One queue message per **frame**. Scale with `--scale ocr=N` so replicas share the work of a single video.

## Queue

| Setting | Value |
|---------|-------|
| Queue | `OCR_QUEUE` (`ocr`) |
| Routing | `OCR_ROUTING_KEY` (`video.ocr_frame_requested.v1`) |
| Exchange | `domain.events` |
| Binary | `cmd/ocr/worker.py` (RapidOCR + onnxruntime) |

On startup the worker unbinds the legacy `video.frames_extracted.v1` key.

## Jobs

| Job type | Owner | Role |
|----------|-------|------|
| `segment` | `segment` | Split video; one `video.segment_ready.v1` per slice |
| `frame` | `frame` | Extract frames per segment; created when segment succeeds |
| `ocr` | main `worker` | Created on extract; fan-out + counters |
| `classify` | `classify` | Started after `video.frames_ocr_completed.v1` |

All jobs share the same statuses: `pending` → `processing` → `success` / `failed`. Distinguish stages with `type`.

`expectedFrameCount` comes from extract. `ocrCompletedCount` increments on each persist. Realtime `job.updated` is republished after every batch.

`markReady` waits until the `frame` job is `success` **and** every frame has an `ocrs` row.

## Engine

RapidOCR + ONNX Runtime (CPU). Paddle was dropped after aarch64 SIGSEGV.

| Env | Default | Role |
|-----|---------|------|
| `OCR_CONCURRENCY` | `2` | RabbitMQ prefetch (unacked messages buffered per replica) |
| `OCR_INFER_CONCURRENCY` | `2` | Preloaded RapidOCR engines (pool size) |
| `OCR_INTRA_OP_THREADS` | `2` | onnxruntime threads per inference |
| `OCR_BATCH_SIZE` | `8` | Unused (one frame per message) |

Resize happens at extract (`FRAME_MAX_WIDTH_PX`), not here. The worker downloads the stored JPEG as-is.

Empty OCR text is a success with `text=""`. Classify will `skip` that frame.

## Scale

```bash
docker compose -f compose.dev.yaml up --scale ocr=3
```

`ocr` has no `container_name`. The main `worker` must stay a singleton (outbox + expire).

Throughput is mostly `--scale ocr=N`: each replica pulls frame messages from the shared queue. Watch CPU: `replicas × intra_op` should stay near the host core count.

## Schema

`ocrs`: `UNIQUE (frame_id)`. Redelivery overwrites. Status `success` or `failed`. `lines` is JSONB: each line is `{ text, confidence, box: [{x,y}×4] }` in pixels of the stored JPEG. Classify still receives only joined `text`. `GET /videos/:id` returns `ocrText` and `ocrLines`.

## Code map

| Piece | Location |
|-------|----------|
| Python worker | `cmd/ocr/worker.py`, `cmd/ocr/engine.py` |
| Start OCR + fan-out | `internal/application/command/video/ocr_frames.go` (`Start`) |
| Persist batch | same file (`PersistBatch`) |
| Event | `internal/application/event/video/` (`video.ocr_*`) |
| Read | `internal/infrastructure/persistence/read/video_query_repository.go` |
