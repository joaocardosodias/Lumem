-- Reset only the disposable database from compose.yml, before a complete run.
BEGIN;
DO $$ BEGIN
  IF current_database() <> 'rf_browser' OR current_user <> 'rf_browser' THEN
    RAISE EXCEPTION 'This reset is restricted to the isolated browser test database';
  END IF;
END $$;
TRUNCATE telemetry_alarms, maintenance_events, file_uploads,
  inspections, predictions, prediction_runs, prediction_outcomes, river_job CASCADE;
COMMIT;
