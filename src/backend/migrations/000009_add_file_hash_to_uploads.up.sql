ALTER TABLE file_uploads ADD COLUMN file_hash TEXT;
CREATE INDEX IF NOT EXISTS idx_file_uploads_hash ON file_uploads(file_hash);
