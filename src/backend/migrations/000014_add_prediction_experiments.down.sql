DROP INDEX IF EXISTS idx_prediction_runs_experiments;
DROP INDEX IF EXISTS idx_prediction_runs_published;

ALTER TABLE prediction_runs
    DROP COLUMN IF EXISTS published_at,
    DROP COLUMN IF EXISTS is_experiment;
