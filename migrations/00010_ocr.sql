-- +goose Up
-- +goose StatementBegin
ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS expected_frame_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS ocr_completed_count INT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS ocr_results (
    id UUID PRIMARY KEY,
    frame_id UUID NOT NULL REFERENCES frames(id) ON DELETE CASCADE,
    text TEXT NOT NULL DEFAULT '',
    confidence NUMERIC NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    error_reason TEXT NOT NULL DEFAULT '',
    CONSTRAINT ocr_results_frame_id_key UNIQUE (frame_id)
);

CREATE INDEX IF NOT EXISTS idx_ocr_results_status ON ocr_results (status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ocr_results;
ALTER TABLE jobs
    DROP COLUMN IF EXISTS ocr_completed_count,
    DROP COLUMN IF EXISTS expected_frame_count;
-- +goose StatementEnd
