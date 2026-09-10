ALTER TABLE file_uploads ADD COLUMN IF NOT EXISTS file_hash TEXT;
DROP INDEX IF EXISTS idx_file_uploads_hash;
CREATE UNIQUE INDEX IF NOT EXISTS idx_file_uploads_hash ON file_uploads(file_hash);

