-- +goose Up
ALTER TABLE media_file ADD COLUMN acoustic_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_media_file_acoustic_id ON media_file(acoustic_id) WHERE acoustic_id != '';

-- +goose Down
DROP INDEX IF EXISTS idx_media_file_acoustic_id;
ALTER TABLE media_file DROP COLUMN acoustic_id;
