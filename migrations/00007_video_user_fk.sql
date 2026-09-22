-- +goose Up
-- +goose StatementBegin
ALTER TABLE videos DROP CONSTRAINT IF EXISTS videos_user_id_fkey;
ALTER TABLE videos
    ADD CONSTRAINT videos_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE videos DROP CONSTRAINT IF EXISTS videos_user_id_fkey;
ALTER TABLE videos
    ADD CONSTRAINT videos_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id);
-- +goose StatementEnd
