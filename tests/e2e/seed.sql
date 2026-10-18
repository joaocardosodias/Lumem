-- Only execute against the isolated database defined in compose.yml.
CREATE EXTENSION IF NOT EXISTS pgcrypto;
INSERT INTO users (id, name, email, password_hash, role, is_active, created_at, updated_at)
VALUES
  ('rf_browser_operator', 'Operador de teste', 'operator@rf.test', crypt('BrowserRF!123', gen_salt('bf', 10)), 'operator', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('rf_browser_engineer', 'Engenharia de teste', 'engineer@rf.test', crypt('BrowserRF!123', gen_salt('bf', 10)), 'enginner', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO UPDATE SET password_hash = EXCLUDED.password_hash;
