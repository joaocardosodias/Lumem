CREATE TABLE IF NOT EXISTS inspections (
    id TEXT PRIMARY KEY,
    prediction_id TEXT REFERENCES predictions(id) ON DELETE SET NULL,
    station_code TEXT NOT NULL,
    platform TEXT NOT NULL,
    door_number INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'in_progress', 'completed', 'canceled')),
    notes TEXT,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_inspections_status ON inspections(status);
CREATE INDEX IF NOT EXISTS idx_inspections_station_door ON inspections(station_code, platform, door_number);
