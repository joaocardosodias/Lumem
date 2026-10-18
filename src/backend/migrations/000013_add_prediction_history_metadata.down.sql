DROP INDEX IF EXISTS idx_predictions_door_history;

ALTER TABLE predictions
    DROP COLUMN IF EXISTS probability;

ALTER TABLE prediction_runs
    DROP COLUMN IF EXISTS model_version,
    DROP COLUMN IF EXISTS threshold_used;
