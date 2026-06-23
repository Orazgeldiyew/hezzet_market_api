BEGIN;

-- Fills the (role, module, action) holes in role_permissions for manager,
-- operator and cashier. After 077 flipped the "no row" default from allow to
-- deny, every action a role is expected to use must have an explicit grant.
-- The matrix below maps to what each role typically does on the floor.
--
-- Grants are written as (allow, deny) pairs per role; the ON CONFLICT clause
-- updates an existing row so re-runs are safe and granted state is reset to
-- the intended value.

-- ── manager — full operational access, including history + import ──────────
INSERT INTO role_permissions (role_id, module, action, granted)
SELECT r.id, m.module, m.action, true
FROM roles r
CROSS JOIN (VALUES
    -- history: managers see the audit log of every domain object.
    ('categories','history'),('customers','history'),('finance','history'),
    ('payroll','history'),('products','history'),('purchases','history'),
    ('reports','history'),('stock','history'),('suppliers','history'),
    ('warehouses','history'),('worker-cards','history'),('workers','history'),
    -- import: managers can bulk-load most catalogs.
    ('categories','import'),('customers','import'),('products','import'),
    ('purchases','import'),('stock','import'),('suppliers','import'),
    ('warehouses','import'),('workers','import'),('sales','import')
) AS m(module, action)
WHERE r.code = 'manager'
ON CONFLICT (role_id, module, action) DO UPDATE SET granted = true;

-- manager — explicit deny on actions that should stay admin-only.
INSERT INTO role_permissions (role_id, module, action, granted)
SELECT r.id, m.module, m.action, false
FROM roles r
CROSS JOIN (VALUES
    ('finance','import'),         -- bulk-loading money is admin-only
    ('payroll','import'),         -- same reasoning for payroll
    ('reports','import'),
    ('worker-cards','import')
) AS m(module, action)
WHERE r.code = 'manager'
ON CONFLICT (role_id, module, action) DO UPDATE SET granted = false;

-- ── operator — floor lead: catalog/stock/sales, no money/audit/HR import ───
INSERT INTO role_permissions (role_id, module, action, granted)
SELECT r.id, m.module, m.action, true
FROM roles r
CROSS JOIN (VALUES
    -- categories: operator owns the catalog tree.
    ('categories','view'), ('categories','create'),
    ('categories','update'),('categories','delete'),
    -- suppliers: operator can manage suppliers (for receiving stock).
    ('suppliers','view'), ('suppliers','create'),
    ('suppliers','update'),('suppliers','delete'),
    -- warehouses: operator sees warehouses; create/edit reserved to manager.
    ('warehouses','view'),
    -- worker-cards: operator hands out / updates cards at the till.
    ('worker-cards','view'), ('worker-cards','create'), ('worker-cards','update')
) AS m(module, action)
WHERE r.code = 'operator'
ON CONFLICT (role_id, module, action) DO UPDATE SET granted = true;

-- operator — explicit deny on everything else they don't need.
INSERT INTO role_permissions (role_id, module, action, granted)
SELECT r.id, m.module, m.action, false
FROM roles r
CROSS JOIN (VALUES
    -- history & import on every module — operators don't audit or bulk-load.
    ('categories','history'),('categories','import'),
    ('customers','history'),  ('customers','import'),
    ('finance','history'),    ('finance','import'),
    ('payroll','history'),    ('payroll','import'),
    ('products','history'),
    ('purchases','history'),  ('purchases','import'),
    ('reports','history'),    ('reports','import'),
    ('sales','import'),
    ('stock','history'),      ('stock','import'),
    ('suppliers','history'),  ('suppliers','import'),
    ('warehouses','create'),  ('warehouses','update'),
    ('warehouses','delete'),  ('warehouses','history'),('warehouses','import'),
    ('worker-cards','delete'),('worker-cards','history'),('worker-cards','import'),
    ('workers','history'),    ('workers','import')
) AS m(module, action)
WHERE r.code = 'operator'
ON CONFLICT (role_id, module, action) DO UPDATE SET granted = false;

-- ── cashier — till only, no imports of anything. ──────────────────────────
INSERT INTO role_permissions (role_id, module, action, granted)
SELECT r.id, m.module, m.action, false
FROM roles r
CROSS JOIN (VALUES
    ('categories','import'),('customers','import'),('finance','import'),
    ('payroll','import'),   ('purchases','import'),('reports','import'),
    ('sales','import'),     ('stock','import'),    ('suppliers','import'),
    ('warehouses','import'),('worker-cards','import'),('workers','import')
) AS m(module, action)
WHERE r.code = 'cashier'
ON CONFLICT (role_id, module, action) DO UPDATE SET granted = false;

COMMIT;
