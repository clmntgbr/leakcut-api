-- +goose Up
-- +goose StatementBegin
UPDATE videos
SET status = CASE status
	WHEN 'pending_upload' THEN 'pending'
	WHEN 'uploaded' THEN 'processing'
	WHEN 'extraction_queued' THEN 'processing'
	WHEN 'extracting' THEN 'processing'
	WHEN 'frames_ready' THEN 'processing'
	WHEN 'ocr_processing' THEN 'processing'
	WHEN 'ocr_ready' THEN 'processing'
	WHEN 'classifying' THEN 'processing'
	WHEN 'classified' THEN 'success'
	WHEN 'extraction_failed' THEN 'failed'
	WHEN 'ocr_failed' THEN 'failed'
	WHEN 'classify_failed' THEN 'failed'
	WHEN 'upload_expired' THEN 'failed'
	ELSE status
END
WHERE status NOT IN ('pending', 'processing', 'success', 'failed');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE videos
SET status = CASE status
	WHEN 'pending' THEN 'pending_upload'
	WHEN 'success' THEN 'classified'
	WHEN 'failed' THEN 'extraction_failed'
	WHEN 'processing' THEN 'extracting'
	ELSE status
END
WHERE status IN ('pending', 'processing', 'success', 'failed');
-- +goose StatementEnd
