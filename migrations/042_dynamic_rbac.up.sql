-- 042_dynamic_rbac.up.sql
-- Convert hardcoded roles to dynamic RBAC with action-level permissions.

BEGIN;

-- 1. Save old role_permissions data
CREATE TEMP TABLE _old_perms AS
  SELECT rp.role, rp.module, rp.enabled
  FROM role_permissions rp;

-- 2. Drop old role_permissions (has hardcoded CHECK constraint)
DROP TABLE role_permissions;

-- 3. Convert roles.code from enum to TEXT
ALTER TABLE roles ALTER COLUMN code TYPE TEXT;
DROP TYPE IF EXISTS role_code;

-- 4. Add metadata columns to roles
ALTER TABLE roles ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE roles ADD COLUMN is_system  BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE roles ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE roles ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- Mark original 4 roles as system (cannot be deleted)
UPDATE roles SET is_system = true
WHERE code IN ('admin', 'cashier', 'operator', 'manager');

-- Add constraint for valid role codes
ALTER TABLE roles ADD CONSTRAINT roles_code_format
  CHECK (code ~ '^[a-z][a-z0-9_]{1,49}$');

-- 5. Create new role_permissions with action-level granularity
CREATE TABLE role_permissions (
    id         BIGSERIAL PRIMARY KEY,
    role_id    INT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    module     TEXT NOT NULL,
    action     TEXT NOT NULL CHECK (action IN ('view','create','update','delete')),
    granted    BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(role_id, module, action)
);

CREATE INDEX idx_role_permissions_role_id ON role_permissions(role_id);

-- 6. Migrate old data: each (role, module, enabled) → 4 rows (view, create, update, delete)
INSERT INTO role_permissions (role_id, module, action, granted)
SELECT r.id, op.module, a.action, op.enabled
FROM _old_perms op
JOIN roles r ON r.code = op.role
CROSS JOIN (VALUES ('view'), ('create'), ('update'), ('delete')) AS a(action);

DROP TABLE _old_perms;

COMMIT;
