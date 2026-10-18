CREATE TABLE IF NOT EXISTS file_uploads (
    id TEXT PRIMARY KEY,
    file_name TEXT NOT NULL,
    file_type TEXT NOT NULL,
    file_hash TEXT,
    rows_processed INTEGER NOT NULL DEFAULT 0,
    rows_inserted INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'processing',
    error_log TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_file_uploads_hash ON file_uploads(file_hash);

CREATE TABLE IF NOT EXISTS telemetry_alarms (
    id TEXT PRIMARY KEY,
    upload_id TEXT REFERENCES file_uploads(id) ON DELETE SET NULL,
    tag TEXT NOT NULL,
    station_code TEXT NOT NULL,
    platform TEXT,
    door_number INTEGER,
    alarm_type TEXT NOT NULL,
    duration_seconds REAL NOT NULL DEFAULT 0,
    event_timestamp TEXT NOT NULL,
    normalization_timestamp TEXT,
    source TEXT NOT NULL DEFAULT 'spreadsheet',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_telemetry_alarms_station_time ON telemetry_alarms(station_code, event_timestamp);
CREATE INDEX IF NOT EXISTS idx_telemetry_alarms_upload_id ON telemetry_alarms(upload_id);
