-- +goose Up
-- +goose StatementBegin
ALTER TABLE jobs ALTER COLUMN analysis_fps SET DEFAULT 2;
ALTER TABLE jobs ALTER COLUMN diff_threshold SET DEFAULT 0.13;
ALTER TABLE jobs ALTER COLUMN max_interval_seconds SET DEFAULT 15;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE jobs ALTER COLUMN analysis_fps SET DEFAULT 3;
ALTER TABLE jobs ALTER COLUMN diff_threshold SET DEFAULT 0.08;
ALTER TABLE jobs ALTER COLUMN max_interval_seconds SET DEFAULT 10;
-- +goose StatementEnd
