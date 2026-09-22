-- +goose Up
-- +goose StatementBegin
ALTER TABLE videos ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id);
CREATE INDEX IF NOT EXISTS idx_videos_user_id_created_at ON videos (user_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_videos_user_id_created_at;
ALTER TABLE videos DROP COLUMN IF EXISTS user_id;
-- +goose StatementEnd
