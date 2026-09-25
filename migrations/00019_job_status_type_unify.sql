-- +goose Up
-- +goose StatementBegin
UPDATE jobs SET type = 'frame' WHERE type = 'extract_frames';

UPDATE jobs SET status = 'processing'
WHERE status IN ('extracting_frames', 'ocr_processing', 'classifying');

UPDATE jobs SET status = 'success'
WHERE status IN ('frames_ready', 'ocr_ready', 'classified');

UPDATE jobs SET status = 'failed'
WHERE status IN ('ocr_failed', 'classify_failed');

ALTER TABLE jobs ALTER COLUMN type SET DEFAULT 'frame';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE jobs ALTER COLUMN type SET DEFAULT 'extract_frames';

UPDATE jobs SET type = 'extract_frames' WHERE type = 'frame';

UPDATE jobs SET status = 'extracting_frames'
WHERE type = 'extract_frames' AND status = 'processing';

UPDATE jobs SET status = 'frames_ready'
WHERE type = 'extract_frames' AND status = 'success';

UPDATE jobs SET status = 'ocr_processing'
WHERE type = 'ocr' AND status = 'processing';

UPDATE jobs SET status = 'ocr_ready'
WHERE type = 'ocr' AND status = 'success';

UPDATE jobs SET status = 'ocr_failed'
WHERE type = 'ocr' AND status = 'failed';

UPDATE jobs SET status = 'classifying'
WHERE type = 'classify' AND status = 'processing';

UPDATE jobs SET status = 'classified'
WHERE type = 'classify' AND status = 'success';

UPDATE jobs SET status = 'classify_failed'
WHERE type = 'classify' AND status = 'failed';
-- +goose StatementEnd
