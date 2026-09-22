-- +goose Up
-- +goose StatementBegin
ALTER TABLE videos DROP COLUMN IF EXISTS source;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE videos ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd
