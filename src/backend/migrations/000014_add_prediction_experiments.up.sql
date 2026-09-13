ALTER TABLE prediction_runs
    ADD COLUMN IF NOT EXISTS is_experiment BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS published_at TIMESTAMP WITH TIME ZONE;

UPDATE prediction_runs
SET published_at = created_at
WHERE is_experiment = FALSE AND published_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_prediction_runs_published
    ON prediction_runs (published_at DESC)
    WHERE is_experiment = FALSE;

CREATE INDEX IF NOT EXISTS idx_prediction_runs_experiments
    ON prediction_runs (created_at DESC)
    WHERE is_experiment = TRUE;
