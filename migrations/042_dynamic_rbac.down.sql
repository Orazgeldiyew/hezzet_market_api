-- 042_dynamic_rbac.down.sql
-- Revert to hardcoded enum roles + module-level permissions.

BEGIN;

DROP TABLE role_permissions;

-- Recreate enum
CREATE TYPE role_code AS ENUM ('admin', 'cashier', 'operator', 'manager');

-- Remove new columns from roles
ALTER TABLE roles DROP COLUMN IF EXISTS description;
ALTER TABLE roles DROP COLUMN IF EXISTS is_system;
ALTER TABLE roles DROP COLUMN IF EXISTS created_at;
ALTER TABLE roles DROP COLUMN IF EXISTS updated_at;
ALTER TABLE roles DROP CONSTRAINT IF EXISTS roles_code_format;

-- Convert code back to enum
ALTER TABLE roles ALTER COLUMN code TYPE role_code USING code::role_code;

-- Recreate old role_permissions
CREATE TABLE role_permissions (
    id         BIGSERIAL PRIMARY KEY,
    role       TEXT NOT NULL CHECK (role IN ('cashier', 'operator', 'manager')),
    module     TEXT NOT NULL,
    enabled    BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(role, module)
);

INSERT INTO role_permissions (role, module, enabled) VALUES
  ('cashier',  'sales',     true),
  ('cashier',  'stock',     true),
  ('cashier',  'products',  true),
  ('cashier',  'customers', true),
  ('cashier',  'finance',   true),
  ('cashier',  'reports',   false),
  ('cashier',  'purchases', false),
  ('cashier',  'workers',   false),
  ('operator', 'sales',     true),
  ('operator', 'stock',     true),
  ('operator', 'products',  true),
  ('operator', 'customers', true),
  ('operator', 'finance',   true),
  ('operator', 'reports',   false),
  ('operator', 'purchases', true),
  ('operator', 'workers',   true),
  ('manager',  'sales',     true),
  ('manager',  'stock',     true),
  ('manager',  'products',  true),
  ('manager',  'customers', true),
  ('manager',  'finance',   true),
  ('manager',  'reports',   true),
  ('manager',  'purchases', true),
  ('manager',  'workers',   true);

COMMIT;
