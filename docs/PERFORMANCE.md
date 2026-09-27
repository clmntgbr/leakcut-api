# performance

## Overview

Keep the pipeline cheap on long videos without dropping screens that might contain text. Defaults live on the extract job and in compose.

| Lever | Default | Effect |
|-------|---------|--------|
| `analysis_fps` | `2` | Decode rate for the pHash pass |
| `phash_distance_threshold` | `14` | Higher → fewer `scene_change` keeps (Hamming vs last kept) |
| `max_interval_seconds` | `15` | Safety net on a static screen |
| `FRAME_MAX_WIDTH_PX` | `960` | ffmpeg scale at extract (OCR sees the same JPEG) |
| `FRAME_UPLOAD_CONCURRENCY` | `4` | Parallel MinIO puts during extract |
| `--scale frame=N` | `2` | Parallel segment extract on one video |
| `--scale ocr=N` | `2` | Competing consumers on `video.ocr_frame_requested.v1` (one frame each) |
| `--scale classify=N` | `2` | Competing consumers on `video.classify_frame_requested.v1` (one frame each) |
| `OCR_CONCURRENCY` | `2` | RabbitMQ prefetch per OCR replica |

A previous 5-minute clip at ~1.6 fps / 8% produced ~490 frames. Pixel luminance also missed same-dark-theme scene changes. pHash (default 14) targets ~40–60 keeps on a 5-minute clip. If volume is still high, raise the threshold (16, 18, 20); if a scene is missed, lower it (12, 10). There is no pre-OCR “has text?” gate — false negatives would skip real leaks.

## Workers

| Process | Scale | Why |
|---------|-------|-----|
| `worker` | **1** (`container_name: worker`) | Outbox relay + expire; no `SKIP LOCKED` |
| `frame` | N | CPU ffmpeg; one **segment** per message |
| `ocr` | N | One message **per frame** |
| `classify` | N | One message **per frame** after OCR |

```bash
docker compose -f compose.dev.yaml up --scale ocr=2 --scale frame=2 --scale classify=2
```

## Pipeline fan-out

```
segment_ready × N  →  frame workers
                   →  ocr_frame_requested × frames
                   →  classify_frame_requested × frames
```

ffmpeg streams candidates **per segment**. Each retained frame is uploaded, upserted, then **immediately** enqueued as `video.ocr_frame_requested.v1` (pipeline does not wait for the segment to finish). OCR persist emits `classify_frame_requested` the same way. See [segment](segment.md), [ocr](ocr.md), [classify](classify.md).

Outbox poll is `OUTBOX_POLL_INTERVAL` (default `2s`).

## What not to do

- Scale the main `worker` — two relays double-publish outbox rows.
- Downscale again in the OCR worker — the JPEG is already 960-wide.
- Add a cheap “blank image” skip before OCR without a measured false-negative budget.

## Code map

| Piece | Location |
|-------|----------|
| Extract defaults | `internal/domain/job` (`DefaultAnalysisFPS`, …) |
| ffmpeg scale | `internal/infrastructure/video/extractor.go` |
| Per-frame OCR event | `internal/application/command/video/extract_frames.go` |
| OCR worker | `cmd/ocr/worker.py` |
| Compose scale | `compose.dev.yaml` (`ocr`, `frame`, `classify` have no `container_name`) |
