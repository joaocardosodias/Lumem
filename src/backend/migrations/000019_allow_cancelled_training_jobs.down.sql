UPDATE ml_training_jobs
SET status = 'failed',
    error_message = COALESCE(NULLIF(error_message, ''), 'Treinamento cancelado antes do rollback da migração.')
WHERE status = 'cancelled';

ALTER TABLE ml_training_jobs
    DROP CONSTRAINT IF EXISTS ml_training_jobs_status_check;

ALTER TABLE ml_training_jobs
    ADD CONSTRAINT ml_training_jobs_status_check
    CHECK (status IN ('queued', 'running', 'completed', 'failed'));
