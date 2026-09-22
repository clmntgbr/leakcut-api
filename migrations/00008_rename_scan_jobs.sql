-- +goose Up
-- +goose StatementBegin
ALTER TABLE scan_jobs RENAME TO jobs;
ALTER TABLE jobs RENAME CONSTRAINT scan_jobs_pkey TO jobs_pkey;
ALTER TABLE jobs RENAME CONSTRAINT scan_jobs_video_id_key TO jobs_video_id_key;
ALTER TABLE jobs RENAME CONSTRAINT scan_jobs_video_id_fkey TO jobs_video_id_fkey;
ALTER INDEX idx_scan_jobs_status RENAME TO idx_jobs_status;

ALTER TABLE frames RENAME COLUMN scan_job_id TO job_id;
ALTER TABLE frames RENAME CONSTRAINT frames_scan_job_id_fkey TO frames_job_id_fkey;
ALTER TABLE frames RENAME CONSTRAINT frames_scan_job_id_index_key TO frames_job_id_index_key;
ALTER INDEX idx_frames_scan_job_id RENAME TO idx_frames_job_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER INDEX idx_frames_job_id RENAME TO idx_frames_scan_job_id;
ALTER TABLE frames RENAME CONSTRAINT frames_job_id_index_key TO frames_scan_job_id_index_key;
ALTER TABLE frames RENAME CONSTRAINT frames_job_id_fkey TO frames_scan_job_id_fkey;
ALTER TABLE frames RENAME COLUMN job_id TO scan_job_id;

ALTER INDEX idx_jobs_status RENAME TO idx_scan_jobs_status;
ALTER TABLE jobs RENAME CONSTRAINT jobs_video_id_fkey TO scan_jobs_video_id_fkey;
ALTER TABLE jobs RENAME CONSTRAINT jobs_video_id_key TO scan_jobs_video_id_key;
ALTER TABLE jobs RENAME CONSTRAINT jobs_pkey TO scan_jobs_pkey;
ALTER TABLE jobs RENAME TO scan_jobs;
-- +goose StatementEnd
