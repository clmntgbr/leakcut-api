-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS video_segments (
    id UUID PRIMARY KEY,
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    segment_index INT NOT NULL,
    offset_ms BIGINT NOT NULL DEFAULT 0,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    storage_key TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    failure_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT video_segments_video_id_segment_index_key UNIQUE (video_id, segment_index)
);

CREATE INDEX IF NOT EXISTS idx_video_segments_video_id ON video_segments (video_id);
CREATE INDEX IF NOT EXISTS idx_video_segments_job_id ON video_segments (job_id);
CREATE INDEX IF NOT EXISTS idx_video_segments_status ON video_segments (status);

ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS expected_segment_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS completed_segment_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS segment_duration_seconds INT NOT NULL DEFAULT 120,
    ADD COLUMN IF NOT EXISTS min_video_duration_for_split_seconds INT NOT NULL DEFAULT 90,
    ADD COLUMN IF NOT EXISTS max_segments INT NOT NULL DEFAULT 40;

ALTER TABLE frames
    ADD COLUMN IF NOT EXISTS segment_index INT NOT NULL DEFAULT 0;

ALTER TABLE frames DROP CONSTRAINT IF EXISTS frames_video_id_index_key;
ALTER TABLE frames
    ADD CONSTRAINT frames_video_id_segment_index_index_key UNIQUE (video_id, segment_index, index);

CREATE INDEX IF NOT EXISTS idx_frames_video_id_segment_index ON frames (video_id, segment_index);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_frames_video_id_segment_index;

ALTER TABLE frames DROP CONSTRAINT IF EXISTS frames_video_id_segment_index_index_key;
ALTER TABLE frames
    ADD CONSTRAINT frames_video_id_index_key UNIQUE (video_id, index);

ALTER TABLE frames DROP COLUMN IF EXISTS segment_index;

ALTER TABLE jobs
    DROP COLUMN IF EXISTS max_segments,
    DROP COLUMN IF EXISTS min_video_duration_for_split_seconds,
    DROP COLUMN IF EXISTS segment_duration_seconds,
    DROP COLUMN IF EXISTS completed_segment_count,
    DROP COLUMN IF EXISTS expected_segment_count;

DROP TABLE IF EXISTS video_segments;
-- +goose StatementEnd
