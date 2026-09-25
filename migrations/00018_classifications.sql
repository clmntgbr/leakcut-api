-- +goose Up
-- +goose StatementBegin
ALTER TABLE frame_findings RENAME TO classifications;

ALTER TABLE classifications RENAME CONSTRAINT frame_findings_pkey TO classifications_pkey;
ALTER TABLE classifications RENAME CONSTRAINT frame_findings_frame_id_key TO classifications_frame_id_key;
ALTER TABLE classifications RENAME CONSTRAINT frame_findings_frame_id_fkey TO classifications_frame_id_fkey;

ALTER INDEX idx_frame_findings_confidential RENAME TO idx_classifications_confidential;
ALTER INDEX idx_frame_findings_status RENAME TO idx_classifications_status;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER INDEX idx_classifications_confidential RENAME TO idx_frame_findings_confidential;
ALTER INDEX idx_classifications_status RENAME TO idx_frame_findings_status;

ALTER TABLE classifications RENAME CONSTRAINT classifications_pkey TO frame_findings_pkey;
ALTER TABLE classifications RENAME CONSTRAINT classifications_frame_id_key TO frame_findings_frame_id_key;
ALTER TABLE classifications RENAME CONSTRAINT classifications_frame_id_fkey TO frame_findings_frame_id_fkey;

ALTER TABLE classifications RENAME TO frame_findings;
-- +goose StatementEnd
