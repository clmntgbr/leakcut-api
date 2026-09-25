-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF to_regclass('public.ocr') IS NOT NULL AND to_regclass('public.ocrs') IS NULL THEN
        ALTER TABLE ocr RENAME TO ocrs;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ocr_pkey' AND conrelid = 'public.ocrs'::regclass
    ) THEN
        ALTER TABLE ocrs RENAME CONSTRAINT ocr_pkey TO ocrs_pkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ocr_frame_id_key' AND conrelid = 'public.ocrs'::regclass
    ) THEN
        ALTER TABLE ocrs RENAME CONSTRAINT ocr_frame_id_key TO ocrs_frame_id_key;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ocr_frame_id_fkey' AND conrelid = 'public.ocrs'::regclass
    ) THEN
        ALTER TABLE ocrs RENAME CONSTRAINT ocr_frame_id_fkey TO ocrs_frame_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = 'public' AND c.relname = 'idx_ocr_status'
    ) THEN
        ALTER INDEX idx_ocr_status RENAME TO idx_ocrs_status;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF to_regclass('public.ocrs') IS NOT NULL AND to_regclass('public.ocr') IS NULL THEN
        ALTER TABLE ocrs RENAME TO ocr;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ocrs_pkey' AND conrelid = 'public.ocr'::regclass
    ) THEN
        ALTER TABLE ocr RENAME CONSTRAINT ocrs_pkey TO ocr_pkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ocrs_frame_id_key' AND conrelid = 'public.ocr'::regclass
    ) THEN
        ALTER TABLE ocr RENAME CONSTRAINT ocrs_frame_id_key TO ocr_frame_id_key;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ocrs_frame_id_fkey' AND conrelid = 'public.ocr'::regclass
    ) THEN
        ALTER TABLE ocr RENAME CONSTRAINT ocrs_frame_id_fkey TO ocr_frame_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = 'public' AND c.relname = 'idx_ocrs_status'
    ) THEN
        ALTER INDEX idx_ocrs_status RENAME TO idx_ocr_status;
    END IF;
END $$;
-- +goose StatementEnd
