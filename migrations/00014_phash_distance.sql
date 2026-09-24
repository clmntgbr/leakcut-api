-- +goose Up
-- +goose StatementBegin
ALTER TABLE jobs
    RENAME COLUMN diff_threshold TO phash_distance_threshold;

ALTER TABLE jobs
    ALTER COLUMN phash_distance_threshold TYPE INTEGER USING 14,
    ALTER COLUMN phash_distance_threshold SET DEFAULT 14,
    ALTER COLUMN phash_distance_threshold SET NOT NULL;

ALTER TABLE jobs
    ALTER COLUMN analysis_fps SET DEFAULT 2;

ALTER TABLE jobs
    ALTER COLUMN max_interval_seconds SET DEFAULT 15;

UPDATE jobs
SET phash_distance_threshold = 14
WHERE type = 'extract_frames';

ALTER TABLE frames
    RENAME COLUMN diff_score TO phash_distance;

ALTER TABLE frames
    ALTER COLUMN phash_distance TYPE INTEGER USING ROUND(COALESCE(phash_distance, 0))::integer;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE frames
    ALTER COLUMN phash_distance TYPE NUMERIC USING phash_distance::numeric;

ALTER TABLE frames
    RENAME COLUMN phash_distance TO diff_score;

ALTER TABLE jobs
    ALTER COLUMN analysis_fps SET DEFAULT 2;

ALTER TABLE jobs
    ALTER COLUMN max_interval_seconds SET DEFAULT 15;

ALTER TABLE jobs
    ALTER COLUMN phash_distance_threshold TYPE NUMERIC USING 0.13,
    ALTER COLUMN phash_distance_threshold SET DEFAULT 0.13;

ALTER TABLE jobs
    RENAME COLUMN phash_distance_threshold TO diff_threshold;
-- +goose StatementEnd
