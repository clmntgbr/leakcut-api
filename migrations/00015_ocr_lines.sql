-- +goose Up
-- +goose StatementBegin
ALTER TABLE ocr_results
    ADD COLUMN IF NOT EXISTS lines JSONB NOT NULL DEFAULT '[]'::jsonb;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ocr_results
    DROP COLUMN IF EXISTS lines;
-- +goose StatementEnd
