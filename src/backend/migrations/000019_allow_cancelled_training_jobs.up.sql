ALTER TABLE ml_training_jobs
    DROP CONSTRAINT IF EXISTS ml_training_jobs_status_check;

ALTER TABLE ml_training_jobs
    ADD CONSTRAINT ml_training_jobs_status_check
    CHECK (status IN ('queued', 'running', 'completed', 'failed', 'cancelled'));
