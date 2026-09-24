-- Execute como administrador e substitua a senha antes de configurar ML_DATABASE_URL.
-- Este arquivo não faz parte da sequência automática de migrations.
CREATE ROLE g04_ml_reader LOGIN PASSWORD 'SUBSTITUA_POR_UMA_SENHA_FORTE';
GRANT CONNECT ON DATABASE postgres TO g04_ml_reader;
GRANT USAGE ON SCHEMA public TO g04_ml_reader;
GRANT SELECT ON ml_telemetry_v1, ml_maintenance_v1, ml_active_doors_v1 TO g04_ml_reader;
REVOKE INSERT, UPDATE, DELETE, TRUNCATE ON ALL TABLES IN SCHEMA public FROM g04_ml_reader;
