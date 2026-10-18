DROP VIEW IF EXISTS ml_active_doors_v1;
DROP VIEW IF EXISTS ml_maintenance_v1;
DROP VIEW IF EXISTS ml_telemetry_v1;
DROP TABLE IF EXISTS prediction_outcomes;
DROP TABLE IF EXISTS ml_drift_reports;
DROP TABLE IF EXISTS ml_training_jobs;
DROP TABLE IF EXISTS ml_threshold_metrics;
DROP INDEX IF EXISTS uq_ml_model_runs_version;

ALTER TABLE prediction_runs
    DROP COLUMN IF EXISTS validation_source,
    DROP COLUMN IF EXISTS validation_f1,
    DROP COLUMN IF EXISTS validation_recall,
    DROP COLUMN IF EXISTS validation_precision;

ALTER TABLE ml_model_runs
    DROP COLUMN IF EXISTS activated_at,
    DROP COLUMN IF EXISTS manifest,
    DROP COLUMN IF EXISTS recommended_threshold,
    DROP COLUMN IF EXISTS positive_rows,
    DROP COLUMN IF EXISTS test_rows,
    DROP COLUMN IF EXISTS test_end,
    DROP COLUMN IF EXISTS test_start,
    DROP COLUMN IF EXISTS train_end,
    DROP COLUMN IF EXISTS target_name,
    DROP COLUMN IF EXISTS dataset_sha256,
    DROP COLUMN IF EXISTS artifact_sha256,
    DROP COLUMN IF EXISTS artifact_uri;
