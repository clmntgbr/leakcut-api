-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS videos (
    id UUID PRIMARY KEY,
    original_filename TEXT,
    storage_key TEXT NOT NULL,
    size_bytes BIGINT,
    content_type TEXT,
    status TEXT NOT NULL,
    source TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT videos_storage_key_key UNIQUE (storage_key)
);

CREATE INDEX IF NOT EXISTS idx_videos_status ON videos (status);

CREATE TABLE IF NOT EXISTS upload_sessions (
    id UUID PRIMARY KEY,
    video_id UUID NOT NULL REFERENCES videos(id),
    expires_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_upload_sessions_video_id ON upload_sessions (video_id);
CREATE INDEX IF NOT EXISTS idx_upload_sessions_status_expires_at ON upload_sessions (status, expires_at);

CREATE TABLE IF NOT EXISTS scan_jobs (
    id UUID PRIMARY KEY,
    video_id UUID NOT NULL REFERENCES videos(id),
    status TEXT NOT NULL,
    failure_reason TEXT,
    analysis_fps NUMERIC NOT NULL DEFAULT 3,
    diff_threshold NUMERIC NOT NULL DEFAULT 0.08,
    max_interval_seconds INT NOT NULL DEFAULT 10,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT scan_jobs_video_id_key UNIQUE (video_id)
);

CREATE INDEX IF NOT EXISTS idx_scan_jobs_status ON scan_jobs (status);

CREATE TABLE IF NOT EXISTS frames (
    id UUID PRIMARY KEY,
    scan_job_id UUID NOT NULL REFERENCES scan_jobs(id),
    index INT NOT NULL,
    timestamp_ms BIGINT NOT NULL,
    storage_key TEXT NOT NULL,
    selection_reason TEXT NOT NULL,
    diff_score NUMERIC,
    CONSTRAINT frames_scan_job_id_index_key UNIQUE (scan_job_id, index)
);

CREATE INDEX IF NOT EXISTS idx_frames_scan_job_id ON frames (scan_job_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS frames;
DROP TABLE IF EXISTS scan_jobs;
DROP TABLE IF EXISTS upload_sessions;
DROP TABLE IF EXISTS videos;
-- +goose StatementEnd
