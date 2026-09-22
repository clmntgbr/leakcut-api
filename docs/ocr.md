# ocr

## Overview

Read text from each retained frame. Inference is a Python AMQP worker (`cmd/ocr`). Persistence stays on the main Go `worker`. There is no HTTP OCR API.

```
extract upserts frame → outbox video.ocr_frame_requested.v1
     → queue ocr (any replica) → download PNG → RapidOCR
     → video.ocr_batch_completed.v1 (one result)
     → main worker persist → job.updated (ocrCompletedCount)
     → when extract is frames_ready and all rows exist
       → video.frames_ocr_completed.v1 → classify
```

One queue message per frame. Replicas compete; a 5-minute video fans out as soon as frames land, not after extract finishes.

## Queue

| Setting | Value |
|---------|-------|
| Queue | `OCR_QUEUE` (`ocr`) |
| Routing | `OCR_ROUTING_KEY` (`video.ocr_frame_requested.v1`) |
| Exchange | `domain.events` |
| Binary | `cmd/ocr/worker.py` (RapidOCR + onnxruntime) |

On startup the worker unbinds the legacy `video.frames_extracted.v1` key so old per-video messages are not consumed twice.

## Jobs

| Job type | Owner | Role |
|----------|-------|------|
| `extract_frames` | `frame` | Writes frames + per-frame OCR requests |
| `ocr` | main `worker` | Created on `video.frames_extracted.v1`; counters only |
| `classify` | `classify` | Started after `video.frames_ocr_completed.v1` |

`ocr` job statuses: `queued` → `ocr_processing` → `ocr_ready` / `ocr_failed`.

`expectedFrameCount` comes from extract. `ocrCompletedCount` increments on each persist. Realtime `job.updated` is republished after every batch.

`markReady` waits until the extract job is `frames_ready` **and** every frame has an `ocr_results` row. Early persist while extract is still running only upserts rows.

## Engine

RapidOCR + ONNX Runtime (CPU). Paddle was dropped after aarch64 SIGSEGV.

| Env | Default | Role |
|-----|---------|------|
| `OCR_CONCURRENCY` | `1` | RabbitMQ prefetch (unacked messages per replica) |
| `OCR_INFER_CONCURRENCY` | `2` | Preloaded RapidOCR engines (consume callback is one frame at a time) |
| `OCR_INTRA_OP_THREADS` | `2` | onnxruntime threads per inference |
| `OCR_BATCH_SIZE` | `8` | Unused (one frame per message) |

Resize happens at extract (`FRAME_MAX_WIDTH_PX`), not here. The worker downloads the stored PNG as-is.

Empty OCR text is a success with `text=""`. Classify will `skip` that frame.

## Scale

```bash
docker compose -f compose.dev.yaml up --scale ocr=2
```

`ocr` has no `container_name`. The main `worker` must stay a singleton (outbox + expire).

Throughput is mostly `--scale ocr=N`. `OCR_INFER_CONCURRENCY` only helps if prefetch > 1 **and** inferences run concurrently; the current consume loop is one message at a time. Watch CPU: `replicas × intra_op` should stay near the host core count.

## Schema

`ocr_results`: `UNIQUE (frame_id)`. Redelivery overwrites. Status `success` or `failed`.

## Code map

| Piece | Location |
|-------|----------|
| Python worker | `cmd/ocr/worker.py`, `cmd/ocr/engine.py` |
| Start OCR job | `internal/application/command/video/ocr_frames.go` (`Start`) |
| Persist batch | same file (`PersistBatch`) |
| Event | `internal/application/event/video/` (`video.ocr_*`) |
| Read | `internal/infrastructure/persistence/read/video_query_repository.go` |

See [performance](performance.md) for frame-count / replica sizing.
