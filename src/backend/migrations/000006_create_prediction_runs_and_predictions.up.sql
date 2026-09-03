CREATE TABLE IF NOT EXISTS prediction_runs (
    id TEXT PRIMARY KEY,
    horizon_days INTEGER NOT NULL DEFAULT 7,
    total_doors_evaluated INTEGER NOT NULL DEFAULT 0,
    total_doors_at_risk INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS predictions (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES prediction_runs(id) ON DELETE CASCADE,
    station_code TEXT NOT NULL,
    platform TEXT NOT NULL,
    door_number INTEGER NOT NULL,
    will_fail BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_predictions_run_id ON predictions(run_id);
CREATE INDEX IF NOT EXISTS idx_predictions_will_fail ON predictions(run_id, will_fail);
