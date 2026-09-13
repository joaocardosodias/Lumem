ALTER TABLE prediction_runs
    ADD COLUMN IF NOT EXISTS threshold_used REAL,
    ADD COLUMN IF NOT EXISTS model_version TEXT NOT NULL DEFAULT '';

ALTER TABLE predictions
    ADD COLUMN IF NOT EXISTS probability REAL;

ALTER TABLE prediction_runs
    DROP CONSTRAINT IF EXISTS prediction_runs_threshold_used_check;

ALTER TABLE prediction_runs
    ADD CONSTRAINT prediction_runs_threshold_used_check
    CHECK (threshold_used IS NULL OR (threshold_used >= 0 AND threshold_used <= 1));

ALTER TABLE predictions
    DROP CONSTRAINT IF EXISTS predictions_probability_check;

ALTER TABLE predictions
    ADD CONSTRAINT predictions_probability_check
    CHECK (probability IS NULL OR (probability >= 0 AND probability <= 1));

CREATE INDEX IF NOT EXISTS idx_predictions_door_history
    ON predictions (station_code, platform, door_number, created_at DESC);
