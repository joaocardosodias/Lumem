ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('operator', 'enginner'));

INSERT INTO users (id, name, email, password_hash, role, is_active, created_at, updated_at)
VALUES (
    'usr_engineer_01',
    'Engenheiro Linha 4',
    'engenheiro@motiva.com.br',
    '$2a$10$4zudOMaHpW3tQ/qEH53FeuqrgxNJFy0tWVJbhLt.oU219dUOfsXSe',
    'enginner',
    1,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT (email) DO UPDATE SET role = EXCLUDED.role, password_hash = EXCLUDED.password_hash;
