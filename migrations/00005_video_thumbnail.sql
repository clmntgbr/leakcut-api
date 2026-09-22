-- +goose Up
-- +goose StatementBegin
ALTER TABLE videos ADD COLUMN IF NOT EXISTS thumbnail_key TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE videos DROP COLUMN IF EXISTS thumbnail_key;
-- +goose StatementEnd
