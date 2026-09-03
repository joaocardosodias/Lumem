INSERT INTO users (id, name, email, password_hash, role, is_active, created_at, updated_at)
VALUES (
    'usr_operator_01',
    'Operador Linha 4',
    'operador@motiva.com.br',
    '$2a$10$TR89JJNCXFRtOtkD40so9uEtg77fGZSWjX8uqWn.SF46invv0uTTO',
    'operator',
    1,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT (id) DO NOTHING;
