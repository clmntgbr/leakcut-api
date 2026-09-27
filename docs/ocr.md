# ocr

## Overview

Read text from each retained frame. Inference is a Python AMQP worker (`cmd/ocr`). Persistence stays on the main Go `worker`. There is no HTTP OCR API.

```
segment extract → each frame upserted → outbox video.ocr_frame_requested.v1 (immédiat)
     → queue ocr → video.ocr_frame_completed.v1
     → persist → video.classify_frame_requested.v1 (immédiat)
     → when extract done + all OCR rows → video.frames_ocr_completed.v1
```

OCR is triggered **only** by `video.ocr_frame_requested.v1` (one message per frame). `video.frames_extracted.v1` is extract-complete for realtime/status — it does **not** start OCR.
## Queue

| Setting | Value |
|---------|-------|
| Queue | `OCR_QUEUE` (`ocr`) |
| Routing | `OCR_ROUTING_KEY` (`video.ocr_frame_requested.v1`) |
| Exchange | `domain.events` |
| Binary | `cmd/ocr/worker.py` (RapidOCR + onnxruntime) |

## Jobs

| Job type | Owner | Role |
|----------|-------|------|
| `segment` | `segment` | Split video; one `video.segment_ready.v1` per slice |
| `frame` | `frame` | Extract frames per segment; emits OCR requests |
| `ocr` | main `worker` | Persist OCR results + counters |
| `classify` | `classify` | One message per frame after each OCR persist |

All jobs share the same statuses: `pending` → `processing` → `success` / `failed`. Distinguish stages with `type`.

## Scale

```bash
docker compose -f compose.dev.yaml up --scale ocr=3
```

One message per frame — replicas share work inside a single video.

## Code map

| Piece | Location |
|-------|----------|
| Python worker | `cmd/ocr/worker.py`, `cmd/ocr/engine.py` |
| Fan-out from extract | `extract_frames.go` (`RequestOCRFrames`) |
| Persist | `ocr_frames.go` / `on_ocr_frame_completed.go` |
