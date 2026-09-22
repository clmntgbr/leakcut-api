# performance

## Overview

Keep the pipeline cheap on long videos without dropping screens that might contain text. Defaults live on the extract job and in compose.

| Lever | Default | Effect |
|-------|---------|--------|
| `analysis_fps` | `2` | Decode rate for the scene-change pass |
| `diff_threshold` | `0.13` | Higher → fewer `scene_change` keeps |
| `max_interval_seconds` | `15` | Safety net on a static screen |
| `FRAME_MAX_WIDTH_PX` | `1280` | ffmpeg scale at extract (OCR sees the same PNG) |
| `--scale ocr=N` | `1` | Competing consumers on `video.ocr_frame_requested.v1` |
| `OCR_CONCURRENCY` | `1` | RabbitMQ prefetch per OCR replica |

A previous 5-minute clip at ~1.6 fps / 8% produced ~490 frames. The current defaults target fewer keeps. There is no pre-OCR “has text?” gate — false negatives would skip real leaks.

## Workers

| Process | Scale | Why |
|---------|-------|-----|
| `worker` | **1** (`container_name: worker`) | Outbox relay + expire; no `SKIP LOCKED` |
| `frame` | N | CPU ffmpeg; one video per message |
| `ocr` | N | One message **per frame**; this is the fan-out |
| `classify` | N | One message per video after OCR completes |

```bash
docker compose -f compose.dev.yaml up --scale ocr=2 --scale frame=1
```

## Incremental OCR

ffmpeg streams candidates. Each retained frame is stored and written to the outbox (`video.ocr_frame_requested.v1`) before the next decode. Replicas compete immediately; OCR does not wait for extract to finish.

Outbox poll is `OUTBOX_POLL_INTERVAL` (default `2s`). That is the delay between “frame stored” and “an OCR replica receives it”.

## What not to do

- Scale the main `worker` — two relays double-publish outbox rows.
- Downscale again in the OCR worker — the PNG is already 1280-wide.
- Add a cheap “blank image” skip before OCR without a measured false-negative budget.

## Code map

| Piece | Location |
|-------|----------|
| Extract defaults | `internal/domain/job` (`DefaultAnalysisFPS`, …) |
| ffmpeg scale | `internal/infrastructure/video/extractor.go` |
| Per-frame OCR event | `internal/application/command/video/extract_frames.go` |
| OCR worker | `cmd/ocr/worker.py` |
| Compose scale | `compose.dev.yaml` (`ocr`, `frame`, `classify` have no `container_name`) |
