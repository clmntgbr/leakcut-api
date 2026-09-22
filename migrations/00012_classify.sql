-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS frame_findings (
    id UUID PRIMARY KEY,
    frame_id UUID NOT NULL REFERENCES frames(id) ON DELETE CASCADE,
    confidential BOOLEAN NOT NULL DEFAULT FALSE,
    probability NUMERIC NOT NULL DEFAULT 0,
    categories JSONB NOT NULL DEFAULT '[]'::jsonb,
    status TEXT NOT NULL,
    error_reason TEXT NOT NULL DEFAULT '',
    CONSTRAINT frame_findings_frame_id_key UNIQUE (frame_id)
);

CREATE INDEX IF NOT EXISTS idx_frame_findings_confidential ON frame_findings (confidential);
CREATE INDEX IF NOT EXISTS idx_frame_findings_status ON frame_findings (status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS frame_findings;
-- +goose StatementEnd
