# segment

## Overview

Between upload confirm and frame extraction, the `segment` worker cuts the video into time slices so multiple `frame` replicas can extract in parallel on one video.

```
video.uploaded.v1
  → queue segment
  → probe duration + plan segments
  → ffmpeg -c copy (or reuse original if below threshold)
  → upload segment objects (when split)
  → outbox video.segment_ready.v1 × N
  → queue frame (one message per segment)
  → when CompletedSegmentCount == ExpectedSegmentCount
       → video.frames_extracted.v1 → OCR Start → video.ocr_frame_requested.v1 × N
```

Short videos stay on a **single path**: one `video.segment_ready.v1` with `segmentIndex=0`, `offsetMs=0`, `storageKey` = the original object (no duplicate upload).

## Queue

| Setting | Value |
|---------|-------|
| Queue | `SEGMENT_QUEUE` (`segment`) |
| Routing | `SEGMENT_ROUTING_KEY` (`video.uploaded.v1`) |
| Binary | `cmd/segment` |

Frame workers now bind `FRAME_ROUTING_KEY` = `video.segment_ready.v1` (not `video.uploaded.v1`).

## Planning

| Job param | Default | Role |
|-----------|---------|------|
| `segment_duration_seconds` | 120 | Target slice length |
| `min_video_duration_for_split_seconds` | 90 | Below → no ffmpeg split |
| `max_segments` | 40 | Cap; effective duration rises if needed |

```
count = ceil(duration / segment_duration)
if count > max_segments → duration = ceil(duration / max_segments), count = max_segments
```

## Frames across segments

- Storage key: `frames/{videoId}/{segment:02d}_{local:04d}.jpg`
- `timestampMs = segment.offsetMs + localTimestampMs`
- Unique: `(video_id, segment_index, index)`
- pHash resets per segment (one possible extra frame at borders)

## Jobs

| Job | Role |
|-----|------|
| `segment` | Created on upload confirm. Owns split params + `expected_segment_count`. `success` when slices are published. |
| `frame` | Created when segment succeeds. Tracks `completed_segment_count` until all extracts finish. |

## Counters

- `segment.expected_segment_count` — number of slices planned
- `frame.expected_segment_count` / `frame.completed_segment_count` — extract progress (atomic increment)

Segment MP4 objects under `videos/{id}/segments/` are deleted after each segment extract and again when all segments finish (the original `original.mp4` is never deleted).

## Code map

| Piece | Location |
|-------|----------|
| Plan | `internal/domain/segment/plan.go` |
| Split (ffmpeg) | `internal/infrastructure/video/splitter.go` |
| Command | `internal/application/command/video/segment_video.go` |
| Extract per segment | `internal/application/command/video/extract_frames.go` |
| Schema | `migrations/00021_video_segments.sql` |
