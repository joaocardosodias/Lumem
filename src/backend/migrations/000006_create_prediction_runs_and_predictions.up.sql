CREATE TABLE IF NOT EXISTS prediction_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    horizon_days INTEGER NOT NULL DEFAULT 7,
    total_doors_evaluated INTEGER NOT NULL DEFAULT 0,
    total_doors_at_risk INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS predictions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id INTEGER NOT NULL REFERENCES prediction_runs(id) ON DELETE CASCADE,
    station_code TEXT NOT NULL,
    platform TEXT NOT NULL,
    door_number INTEGER NOT NULL,
    will_fail INTEGER NOT NULL CHECK(will_fail IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_predictions_run_id ON predictions(run_id);
CREATE INDEX IF NOT EXISTS idx_predictions_will_fail ON predictions(run_id, will_fail);
