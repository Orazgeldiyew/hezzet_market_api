-- 002_users_auth.up.sql

-- Create role_code enum if it does not already exist
DO $$ BEGIN
  CREATE TYPE role_code AS ENUM ('admin', 'cashier', 'operator', 'manager');
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name     TEXT,
    phone         TEXT,
    email         TEXT,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_is_active  ON users(is_active);

CREATE TABLE IF NOT EXISTS roles (
    id   SERIAL PRIMARY KEY,
    code role_code UNIQUE NOT NULL,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id INT    NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

INSERT INTO roles (code, name) VALUES
    ('admin',    'Administrator'),
    ('cashier',  'Cashier'),
    ('operator', 'Operator'),
    ('manager',  'Manager')
ON CONFLICT DO NOTHING;
