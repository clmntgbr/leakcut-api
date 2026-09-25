# classify

## Overview

Decide whether each frame’s OCR text looks confidential. The `classify` worker consumes `video.frames_ocr_completed.v1`. Engine is `CLASSIFY_ENGINE`:

| Value | Engine |
|-------|--------|
| `local` | Regex / heuristics in-process (dev default) |
| `jev` | Vercel AI Gateway `typesafe-ai/jev` |

```
video.frames_ocr_completed.v1 → queue classify
     → load OCR rows → skip empty text
     → local rules or POST ai-gateway /v1/evaluate
     → upsert classifications
     → video.frames_classified.v1
```

Classification runs **only** when trimmed OCR text is non-empty. Empty / whitespace → classification `skipped`, `confidential=false`. That is not a hit.

## Queue

| Setting | Value |
|---------|-------|
| Queue | `CLASSIFY_QUEUE` (`classify`) |
| Routing | `CLASSIFY_ROUTING_KEY` (`video.frames_ocr_completed.v1`) |
| Binary | `cmd/classify` |
| Threshold | `CLASSIFY_THRESHOLD` (`0.7`) |

`classify` has no `container_name` — `--scale classify=2` is allowed. One message per **video** (after all OCR rows exist), so extra replicas help across videos, not inside one video.

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
| `CLASSIFY_ENGINE` | `local` or `jev` (compose.dev defaults to `local`) |
| `AI_GATEWAY_URL` | Default `https://ai-gateway.vercel.sh` (Jev only) |
| `AI_GATEWAY_API_KEY` or `JEV_API_KEY` | Bearer token (Jev only) |
| `CLASSIFY_THRESHOLD` | Confidential cutoff (default `0.7`) |

## Code map

| Piece | Location |
|-------|----------|
| Worker | `cmd/classify` |
| Command | `internal/application/command/video/classify_frames.go` |
| Jev client | `internal/infrastructure/classify/client.go` |
| Domain | `internal/domain/classification/` |
| Event start | `internal/application/event/video/on_ocr_completed_classify.go` |

`GET /api/videos/:id` embeds each frame’s `classification` — see [upload & frame extraction](frame.md).
