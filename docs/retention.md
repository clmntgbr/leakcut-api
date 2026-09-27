# Frame retention

## Overview

After classification finishes, leakcut decides which frame **images** to keep in object storage. The pipeline (extract → OCR → classify) is unchanged — retention only runs **after** every frame has a classification row.

```
classify complete
  → decide retained / prune_reason per frame
  → DELETE pruned objects in S3
  → UPDATE frames.retained + prune_reason
  → mark classify job success + publish video.frames_classified.v1
```

Only the S3 object is removed. The `frames` row stays (timestamp, OCR, classification) so timelines and risk charts keep full temporal coverage. `GET /videos/:id` returns `retained` / `pruneReason`; `imageUrl` is null when not retained.

## Policy (B + A)

1. **Confidential** (`classification.confidential = true`) → always kept (`kept_confidential`).
2. **Context window** around any confidential frame (default ±4s) → kept (`kept_context`), all categories.
3. Remaining **empty OCR** (`ocr.status = empty`) → regular temporal sample at `retention_ratio_empty` (default 10%).
4. Remaining **neutral** (text, not confidential) → sample at `retention_ratio_neutral` (default 50%).

| Category | Default ratio | Reasons |
|----------|---------------|---------|
| Confidential | 100% | `kept_confidential` |
| Context | 100% inside window | `kept_context` |
| Empty | 10% | `kept_sampled_empty` / `pruned_empty` |
| Neutral | 50% | `kept_sampled_neutral` / `pruned_redundant_neutral` |

Sampling is deterministic: among candidates of that category sorted by timestamp, keep every `round(1/ratio)`-th frame (index 0, N, 2N, …).

## Job parameters

Stored on the `frame` job (per video):

| Column | Default |
|--------|---------|
| `retention_ratio_neutral` | `0.5` |
| `retention_ratio_empty` | `0.1` |
| `retention_context_window_seconds` | `4` |

Set both ratios to `1.0` to keep every image.

## Code map

| Piece | Location |
|-------|----------|
| Algorithm | `internal/domain/frame/retention.go` |
| Apply (S3 + DB) | `ClassifyFramesHandler.applyRetention` |
| Schema | `migrations/00020_frame_retention.sql` |

## Notes

- Irreversible for images: recovering a pruned JPEG requires re-extract from the original video.
- Must run only after classification is complete for the video.
- A video with no confidential frames never activates the context window — only per-category sampling applies.
