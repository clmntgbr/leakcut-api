-- +goose Up
-- +goose StatementBegin
ALTER TABLE frames
    ADD COLUMN video_id UUID;

UPDATE frames
SET video_id = jobs.video_id
FROM jobs
WHERE jobs.id = frames.job_id;

ALTER TABLE frames
    ALTER COLUMN video_id SET NOT NULL;

ALTER TABLE frames
    ADD CONSTRAINT frames_video_id_fkey
    FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE;

ALTER TABLE frames
    DROP CONSTRAINT frames_job_id_index_key;

ALTER TABLE frames
    ADD CONSTRAINT frames_video_id_index_key UNIQUE (video_id, index);

DROP INDEX IF EXISTS idx_frames_job_id;

CREATE INDEX idx_frames_video_id ON frames (video_id);

ALTER TABLE frames
    DROP CONSTRAINT frames_job_id_fkey;

ALTER TABLE frames
    DROP COLUMN job_id;

ALTER TABLE ocr_results RENAME TO ocr;

ALTER TABLE ocr RENAME CONSTRAINT ocr_results_pkey TO ocr_pkey;
ALTER TABLE ocr RENAME CONSTRAINT ocr_results_frame_id_key TO ocr_frame_id_key;
ALTER TABLE ocr RENAME CONSTRAINT ocr_results_frame_id_fkey TO ocr_frame_id_fkey;

ALTER INDEX idx_ocr_results_status RENAME TO idx_ocr_status;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER INDEX idx_ocr_status RENAME TO idx_ocr_results_status;

ALTER TABLE ocr RENAME CONSTRAINT ocr_pkey TO ocr_results_pkey;
ALTER TABLE ocr RENAME CONSTRAINT ocr_frame_id_key TO ocr_results_frame_id_key;
ALTER TABLE ocr RENAME CONSTRAINT ocr_frame_id_fkey TO ocr_results_frame_id_fkey;

ALTER TABLE ocr RENAME TO ocr_results;

ALTER TABLE frames
    ADD COLUMN job_id UUID;

UPDATE frames
SET job_id = jobs.id
FROM jobs
WHERE jobs.video_id = frames.video_id
  AND jobs.type = 'extract_frames';

ALTER TABLE frames
    ALTER COLUMN job_id SET NOT NULL;

ALTER TABLE frames
    ADD CONSTRAINT frames_job_id_fkey
    FOREIGN KEY (job_id) REFERENCES jobs(id);

ALTER TABLE frames
    DROP CONSTRAINT frames_video_id_index_key;

ALTER TABLE frames
    ADD CONSTRAINT frames_job_id_index_key UNIQUE (job_id, index);

DROP INDEX IF EXISTS idx_frames_video_id;

CREATE INDEX idx_frames_job_id ON frames (job_id);

ALTER TABLE frames
    DROP CONSTRAINT frames_video_id_fkey;

ALTER TABLE frames
    DROP COLUMN video_id;
-- +goose StatementEnd
