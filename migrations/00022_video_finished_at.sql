-- +goose Up
-- +goose StatementBegin
ALTER TABLE videos
    ADD COLUMN IF NOT EXISTS finished_at TIMESTAMPTZ;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE videos DROP COLUMN IF EXISTS finished_at;
-- +goose StatementEnd
