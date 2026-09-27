-- +goose Up
-- +goose StatementBegin
ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS started_at TIMESTAMPTZ;

ALTER TABLE jobs RENAME COLUMN completed_at TO finished_at;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE jobs RENAME COLUMN finished_at TO completed_at;
ALTER TABLE jobs DROP COLUMN IF EXISTS started_at;
-- +goose StatementEnd
