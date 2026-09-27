-- +goose Up
-- +goose StatementBegin
ALTER TABLE frames
    ADD COLUMN IF NOT EXISTS retained BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS prune_reason TEXT;

ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS retention_ratio_neutral DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    ADD COLUMN IF NOT EXISTS retention_ratio_empty DOUBLE PRECISION NOT NULL DEFAULT 0.1,
    ADD COLUMN IF NOT EXISTS retention_context_window_seconds INT NOT NULL DEFAULT 4;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE jobs
    DROP COLUMN IF EXISTS retention_context_window_seconds,
    DROP COLUMN IF EXISTS retention_ratio_empty,
    DROP COLUMN IF EXISTS retention_ratio_neutral;

ALTER TABLE frames
    DROP COLUMN IF EXISTS prune_reason,
    DROP COLUMN IF EXISTS retained;
-- +goose StatementEnd
