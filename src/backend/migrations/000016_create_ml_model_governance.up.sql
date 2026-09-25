ALTER TABLE ml_model_runs
    ADD COLUMN IF NOT EXISTS artifact_uri TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS artifact_sha256 TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS dataset_sha256 TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS target_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS train_end DATE,
    ADD COLUMN IF NOT EXISTS test_start DATE,
    ADD COLUMN IF NOT EXISTS test_end DATE,
    ADD COLUMN IF NOT EXISTS test_rows INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS positive_rows INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS recommended_threshold REAL,
    ADD COLUMN IF NOT EXISTS manifest JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS activated_at TIMESTAMP WITH TIME ZONE;

UPDATE ml_model_runs
SET status = 'production',
    artifact_uri = CASE
        WHEN artifact_path <> '' THEN 'legacy-local://' || artifact_path
        ELSE artifact_uri
    END,
    target_name = 'falha_nos_proximos_7_dias',
    activated_at = COALESCE(activated_at, finished_at)
WHERE status = 'completed';

CREATE UNIQUE INDEX IF NOT EXISTS uq_ml_model_runs_version ON ml_model_runs(version);

CREATE TABLE IF NOT EXISTS ml_threshold_metrics (
    model_run_id TEXT NOT NULL REFERENCES ml_model_runs(id) ON DELETE CASCADE,
    threshold REAL NOT NULL CHECK (threshold >= 0 AND threshold <= 1),
    true_positives INTEGER NOT NULL CHECK (true_positives >= 0),
    false_positives INTEGER NOT NULL CHECK (false_positives >= 0),
    false_negatives INTEGER NOT NULL CHECK (false_negatives >= 0),
    true_negatives INTEGER NOT NULL CHECK (true_negatives >= 0),
    precision_value REAL NOT NULL CHECK (precision_value >= 0 AND precision_value <= 1),
    recall_value REAL NOT NULL CHECK (recall_value >= 0 AND recall_value <= 1),
    f1_score REAL NOT NULL CHECK (f1_score >= 0 AND f1_score <= 1),
    alert_count INTEGER NOT NULL CHECK (alert_count >= 0),
    alert_rate REAL NOT NULL CHECK (alert_rate >= 0 AND alert_rate <= 1),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (model_run_id, threshold)
);

-- Único ponto legado conhecido. A origem fica explícita no manifest; novas curvas
-- são sempre importadas integralmente do treinamento.
INSERT INTO ml_threshold_metrics (
    model_run_id, threshold, true_positives, false_positives, false_negatives,
    true_negatives, precision_value, recall_value, f1_score, alert_count, alert_rate
)
SELECT id, 0.48, 3432, 18132, 2770, 46850,
       3432.0 / (3432 + 18132),
       3432.0 / (3432 + 2770),
       2.0 * (3432.0 / (3432 + 18132)) * (3432.0 / (3432 + 2770)) /
           ((3432.0 / (3432 + 18132)) + (3432.0 / (3432 + 2770))),
       21564, 21564.0 / 71184
FROM ml_model_runs
WHERE id = 'run_initial_v1'
ON CONFLICT DO NOTHING;

UPDATE ml_model_runs
SET test_rows = 71184,
    positive_rows = 6202,
    recommended_threshold = 0.48,
    manifest = jsonb_build_object(
        'schema_version', 0,
        'provenance', 'legacy_import',
        'warning', 'Regere o modelo para obter hashes e a curva integral.'
    )
WHERE id = 'run_initial_v1' AND manifest = '{}'::jsonb;

ALTER TABLE prediction_runs
    ADD COLUMN IF NOT EXISTS validation_precision REAL,
    ADD COLUMN IF NOT EXISTS validation_recall REAL,
    ADD COLUMN IF NOT EXISTS validation_f1 REAL,
    ADD COLUMN IF NOT EXISTS validation_source TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS ml_training_jobs (
    id TEXT PRIMARY KEY,
    requested_version TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'completed', 'failed')),
    requested_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    parameters JSONB NOT NULL DEFAULT '{}'::jsonb,
    manifest JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ml_training_jobs_created_at
    ON ml_training_jobs(created_at DESC);

CREATE TABLE IF NOT EXISTS ml_drift_reports (
    id BIGSERIAL PRIMARY KEY,
    model_run_id TEXT NOT NULL REFERENCES ml_model_runs(id) ON DELETE CASCADE,
    reference_dataset_sha256 TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    data_drift_detected BOOLEAN,
    psi_score REAL,
    ks_statistic REAL,
    evaluated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    features JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS prediction_outcomes (
    prediction_id TEXT PRIMARY KEY REFERENCES predictions(id) ON DELETE CASCADE,
    window_started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    window_ended_at TIMESTAMP WITH TIME ZONE NOT NULL,
    failure_observed BOOLEAN NOT NULL,
    failure_source TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('true_positive', 'false_positive', 'false_negative', 'true_negative')),
    evaluated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_prediction_outcomes_evaluated_at
    ON prediction_outcomes(evaluated_at DESC);

CREATE OR REPLACE VIEW ml_telemetry_v1 AS
SELECT station_code, platform, door_number, alarm_type, duration_seconds, event_timestamp
FROM telemetry_alarms;

CREATE OR REPLACE VIEW ml_maintenance_v1 AS
SELECT order_type, station_code, installation_location, start_timestamp, end_timestamp
FROM maintenance_events;

CREATE OR REPLACE VIEW ml_active_doors_v1 AS
SELECT s.code AS station_code, d.platform, d.door_number
FROM doors d
JOIN stations s ON s.id = d.station_id
WHERE d.status = 'active';

REVOKE INSERT, UPDATE, DELETE ON ml_telemetry_v1, ml_maintenance_v1, ml_active_doors_v1 FROM PUBLIC;
