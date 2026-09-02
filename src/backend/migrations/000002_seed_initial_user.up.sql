INSERT INTO users (id, name, email, password_hash, role, is_active, created_at, updated_at)
VALUES (
    'usr_operator_01',
    'Operador Linha 4',
    'operador@motiva.com.br',
    '$2a$12$KIXe8KkY3SjQp9z7UuYkTeO1X9R2oN6F8qJ0Z3t1W5A7B8C9D0E1F',
    'operator',
    1,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
);
