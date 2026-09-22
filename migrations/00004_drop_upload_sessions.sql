-- +goose Up
-- +goose StatementBegin
DROP TABLE IF EXISTS upload_sessions;

CREATE INDEX IF NOT EXISTS idx_videos_pending_created_at ON videos (status, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_videos_pending_created_at;

CREATE TABLE IF NOT EXISTS upload_sessions (
    id UUID PRIMARY KEY,
    video_id UUID NOT NULL REFERENCES videos(id),
    expires_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_upload_sessions_video_id ON upload_sessions (video_id);
CREATE INDEX IF NOT EXISTS idx_upload_sessions_status_expires_at ON upload_sessions (status, expires_at);
-- +goose StatementEnd
