CREATE TABLE IF NOT EXISTS maintenance_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    upload_id INTEGER REFERENCES file_uploads(id) ON DELETE SET NULL,
    sap_order_number TEXT,
    order_type TEXT NOT NULL,
    station_code TEXT,
    installation_location TEXT,
    start_timestamp TEXT NOT NULL,
    end_timestamp TEXT,
    duration_hours REAL NOT NULL DEFAULT 0,
    description TEXT,
    symptom TEXT,
    cause TEXT,
    action_taken TEXT,
    source TEXT NOT NULL DEFAULT 'spreadsheet',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_maintenance_events_order_time ON maintenance_events(order_type, start_timestamp);
CREATE INDEX IF NOT EXISTS idx_maintenance_events_upload_id ON maintenance_events(upload_id);
