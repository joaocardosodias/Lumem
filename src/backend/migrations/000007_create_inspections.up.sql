CREATE TABLE IF NOT EXISTS inspections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    prediction_id INTEGER REFERENCES predictions(id) ON DELETE SET NULL,
    station_code TEXT NOT NULL,
    platform TEXT NOT NULL,
    door_number INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'in_progress', 'completed', 'canceled')),
    priority TEXT NOT NULL DEFAULT 'high' CHECK(priority IN ('low', 'medium', 'high', 'critical')),
    assigned_to TEXT REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT,
    scheduled_for TEXT,
    completed_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_inspections_status ON inspections(status);
CREATE INDEX IF NOT EXISTS idx_inspections_station_door ON inspections(station_code, platform, door_number);
