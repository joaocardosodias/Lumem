CREATE TABLE IF NOT EXISTS ml_configs (
    id TEXT PRIMARY KEY,
    threshold REAL NOT NULL DEFAULT 0.27,
    default_horizon_days INTEGER NOT NULL DEFAULT 7,
    auto_retrain_enabled INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by TEXT REFERENCES users(id) ON DELETE SET NULL
);

INSERT INTO ml_configs (id, threshold, default_horizon_days, auto_retrain_enabled, updated_at, updated_by)
VALUES ('active', 0.48, 7, 0, CURRENT_TIMESTAMP, 'usr_engineer_01')
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS ml_model_runs (
    id TEXT PRIMARY KEY,
    version TEXT NOT NULL,
    algorithm TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'completed',
    train_metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
    test_metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
    feature_importance JSONB NOT NULL DEFAULT '[]'::jsonb,
    artifact_path TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT REFERENCES users(id) ON DELETE SET NULL
);

INSERT INTO ml_model_runs (id, version, algorithm, status, train_metrics, test_metrics, feature_importance, artifact_path, started_at, finished_at, created_by)
VALUES (
    'run_initial_v1',
    'v1.0.0',
    'RandomForestClassifier (n_estimators=400, max_depth=14)',
    'completed',
    '{"recall": 0.603, "precision": 0.179, "f1": 0.276, "roc_auc": 0.718}'::jsonb,
    '{"recall": 0.553, "precision": 0.159, "f1": 0.247, "roc_auc": 0.697, "pr_auc": 0.185, "confusion_matrix": {"tp": 3432, "fp": 18132, "fn": 2770, "tn": 46850}}'::jsonb,
    '[{"feature": "n_corretivas_historico_total", "importance": 0.0386, "group": "Degradação Acumulada"}, {"feature": "dur_max_30d", "importance": 0.0330, "group": "Duração de Alarmes"}, {"feature": "dur_max_7d", "importance": 0.0175, "group": "Duração Recente"}, {"feature": "intervalo_medio_zm01", "importance": 0.0140, "group": "Ciclo ZM01"}, {"feature": "dias_desde_zm01", "importance": 0.0126, "group": "Histórico de Falhas"}, {"feature": "dias_desde_ultima_corretiva", "importance": 0.0124, "group": "Recência Corretiva"}, {"feature": "zm01_90d", "importance": 0.0098, "group": "Volume Trimestral"}, {"feature": "frac_tecnicos_30d", "importance": 0.0094, "group": "Severidade Técnica"}]'::jsonb,
    'artifacts/modelo_5.joblib',
    CURRENT_TIMESTAMP - INTERVAL '7 days',
    CURRENT_TIMESTAMP - INTERVAL '7 days' + INTERVAL '4 minutes',
    'usr_engineer_01'
)
ON CONFLICT (id) DO NOTHING;
