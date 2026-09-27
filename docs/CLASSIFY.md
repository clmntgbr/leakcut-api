# classify

## Overview

Decide whether each frame’s OCR text looks confidential. The `classify` worker consumes **one message per frame** (`video.classify_frame_requested.v1`), emitted as each OCR result is persisted. Engine is `CLASSIFY_ENGINE`:

| Value | Engine |
|-------|--------|
| `local` | Regex / heuristics in-process (dev default) |
| `jev` | Vercel AI Gateway `typesafe-ai/jev` |

```
video.ocr_frame_completed.v1 → persist OCR
     → outbox video.classify_frame_requested.v1 × N
     → queue classify (competing consumers) → skip empty text / run engine
     → upsert classification
     → when extract + all OCR + all classifications done
       → video.frames_classified.v1
```

`video.frames_ocr_completed.v1` is also bound for a finalize catch-up (race when the last OCR row lands after the last classify).

Classification runs **only** when trimmed OCR text is non-empty. Empty / whitespace → classification `skipped`, `confidential=false`.

## Queue

| Setting | Value |
|---------|-------|
| Queue | `CLASSIFY_QUEUE` (`classify`) |
| Routing | `CLASSIFY_ROUTING_KEY` (`video.classify_frame_requested.v1,video.frames_ocr_completed.v1`) |
| Binary | `cmd/classify` |
| Threshold | `CLASSIFY_THRESHOLD` (`0.7`) |

`classify` has no `container_name` — `--scale classify=N` shares frame messages across replicas.

The `classify` job uses the shared statuses `pending` → `processing` → `success` / `failed` (`type=classify`).

## Decision

Jev answers a fixed list of boolean questions. `probability` stored on the classification is `max(question probabilities)`. `confidential` is `probability ≥ CLASSIFY_THRESHOLD`.

Questions are atomic booleans (no “or”). Category name stored on the classification:

| Category | Jev question |
|----------|----------------|
| `email` | email address |
| `iban` | IBAN / bank account |
| `api_key` | API key, access token, client secret |
| `password` | password / passphrase |
| `credit_card` | payment card number |
| `phone` | personal phone number |
| `private_key` | private key / certificate |
| `personal_id` | national ID / SSN |
| `connection_string` | DSN / connection string |
| `jwt` | JWT / session token / cookie |
| `wallet_secret` | wallet seed / recovery phrase |
| `webhook_secret` | webhook / HMAC secret |
| `postal_address` | postal / street address |
| `person_name` | first and last name (not a brand) |

A score of `0.01`–`0.02` is a non-hit. Do not treat residual probability as a leak.

## Classification status

| Status | Meaning |
|--------|---------|
| `success` | Jev ran; `confidential` + `categories` set |
| `skipped` | No OCR text; Jev not called |
| `failed` | HTTP / unexpected JSON; `errorReason` set |

`IsFinal` = `success` or `skipped`. Failed rows are retried on redelivery.

## Schema

`classifications`: `UNIQUE (frame_id)`. `categories` is jsonb (`[]` when none). Persistence uses a string `Valuer` so GORM does not send `[]byte` as `bytea`.

## Env

| Variable | Role |
|----------|------|
| `CLASSIFY_ENGINE` | `local` or `jev` |
| `CLASSIFY_THRESHOLD` | Confidential cutoff |
| `CLASSIFY_CONCURRENCY` | Prefetch / in-process parallelism |
| `CLASSIFY_TIMEOUT` | Per-message timeout |
| `AI_GATEWAY_URL` / `AI_GATEWAY_API_KEY` / `JEV_MODEL` | Jev only |

## Code map

| Piece | Location |
|-------|----------|
| Per-frame handler | `internal/application/command/video/classify_frames.go` |
| Event | `on_classify_frame_requested.go` |
| Finalize on OCR done | `on_ocr_completed_classify.go` |
| Engines | `internal/infrastructure/classify/` |
