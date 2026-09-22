# leakcut — Classify (Jev)

Après l'OCR, l'outbox publie `video.frames_ocr_completed.v1`. Le worker `classify` consomme cette clé sur la queue RabbitMQ `classify` (comme `frame` sur `video.uploaded.v1`) et appelle [Jev](https://vercel.com/ai-gateway/models/jev) en HTTP direct via Vercel AI Gateway.

Pas de sidecar HTTP interne : Jev est déjà request/response. RabbitMQ sert à découpler le job (retry, DLQ, scale) du worker principal.

Chaque frame reçoit un oui/non `confidential` plus des probabilités par catégorie : email, IBAN, API key, password, carte, téléphone, clé privée, identifiant, connection string, JWT, seed wallet, webhook/HMAC secret, adresse postale, nom+prénom.

| Pièce | Emplacement |
|---|---|
| Worker Go | `cmd/classify` — queue `classify`, routing key `video.frames_ocr_completed.v1` |
| Client Jev | `internal/infrastructure/classify/client.go` |
| Commande | `internal/application/command/video/classify_frames.go` |
| Job | `classify` (`UNIQUE (video_id, type)`), créé à la fin de l'OCR |
| Persistance | `frame_findings` (migration `00012`) |
| Événement | `video.frames_classified.v1` (sans le texte OCR) |
| Realtime | `job.updated` via le worker principal |

`AI_GATEWAY_API_KEY` est obligatoire. Seuil `CLASSIFY_THRESHOLD=0.7`. Une frame sans texte OCR est `skipped`.
