CREATE TABLE IF NOT EXISTS prediction_schedules (
    id TEXT PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    interval_minutes INTEGER NOT NULL DEFAULT 1440 CHECK (interval_minutes BETWEEN 15 AND 10080),
    next_run_at TIMESTAMP WITH TIME ZONE,
    last_started_at TIMESTAMP WITH TIME ZONE,
    last_completed_at TIMESTAMP WITH TIME ZONE,
    last_run_id TEXT REFERENCES prediction_runs(id) ON DELETE SET NULL,
    last_status TEXT NOT NULL DEFAULT 'idle',
    last_error TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by TEXT REFERENCES users(id) ON DELETE SET NULL
);

INSERT INTO prediction_schedules (id, enabled, interval_minutes, last_status)
VALUES ('official', FALSE, 1440, 'idle')
ON CONFLICT (id) DO NOTHING;
