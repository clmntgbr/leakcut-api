-- +goose Up
-- +goose StatementBegin
ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS type TEXT NOT NULL DEFAULT 'extract_frames';

ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_video_id_key;

ALTER TABLE jobs
    ADD CONSTRAINT jobs_video_id_type_key UNIQUE (video_id, type);

CREATE INDEX IF NOT EXISTS idx_jobs_type ON jobs (type);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM jobs WHERE type = 'ocr';

DROP INDEX IF EXISTS idx_jobs_type;

ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_video_id_type_key;
ALTER TABLE jobs ADD CONSTRAINT jobs_video_id_key UNIQUE (video_id);
ALTER TABLE jobs DROP COLUMN IF EXISTS type;
-- +goose StatementEnd
