-- Auth tables: users, roles, user_roles

CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    full_name     TEXT,
    phone         TEXT,
    email         TEXT,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS roles (
    id   SERIAL PRIMARY KEY,
    code TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id BIGINT  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id INT     NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    UNIQUE(user_id, role_id)
);

-- Seed default roles
INSERT INTO roles (code, name) VALUES
    ('admin',    'Administrator'),
    ('cashier',  'Cashier'),
    ('operator', 'Operator'),
    ('manager',  'Manager')
ON CONFLICT (code) DO NOTHING;
