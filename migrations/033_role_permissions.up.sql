
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
