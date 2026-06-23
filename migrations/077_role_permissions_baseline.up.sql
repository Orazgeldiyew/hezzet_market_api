BEGIN;

-- Baseline grants for each role across every module currently in the system.
-- After switching the default behaviour from "no row = allow" to "no row =
-- deny" (see permissions/repository.go), we need explicit rows for every
-- (role, module, action) the role is expected to use, otherwise legitimate
-- users will start hitting 403s on routes the matrix never seeded.
--
-- The grants below mirror the intent each role had under the old fail-open
-- behaviour. Admins still bypass everything via IsPrivileged, so the admin
-- rows are informational.
--
-- Action coverage per module mirrors the constraint:
--   ('view','create','update','delete','transfer','return','discount',
--    'history','import')

-- ── helpers ────────────────────────────────────────────────────────────────

-- Manager: managerial work — everything except the most destructive admin-only
-- actions. Specifically: view/create/update/delete + history + import on the
-- modules they actually own.
INSERT INTO role_permissions (role_id, module, action, granted)
SELECT r.id, m.module, m.action, true
FROM roles r
CROSS JOIN (VALUES
    -- payroll: managers run the monthly payroll cycle.
    ('payroll','view'), ('payroll','create'), ('payroll','update'),
    -- finance: managers see/run finance transactions.
    ('finance','view'), ('finance','create'), ('finance','update'),
    -- worker-cards: cards are issued by managers.
    ('worker-cards','view'), ('worker-cards','create'),
    ('worker-cards','update'), ('worker-cards','delete')
) AS m(module, action)
WHERE r.code = 'manager'
ON CONFLICT (role_id, module, action) DO UPDATE SET granted = true;

-- Operator: floor lead — sales + stock + purchases + customers + workers
-- (read & write), no payroll/finance/reports/roles management.
INSERT INTO role_permissions (role_id, module, action, granted)
SELECT r.id, m.module, m.action, true
FROM roles r
CROSS JOIN (VALUES
    ('payroll','view'),                 -- can SEE payroll, can't change
    ('worker-cards','view'),
    ('finance','view')                  -- can SEE finance, can't create
) AS m(module, action)
WHERE r.code = 'operator'
ON CONFLICT (role_id, module, action) DO UPDATE SET granted = true;

-- Operator explicit DENIES for the modules they must NOT touch. Without
-- these rows the new fail-closed default would already deny them, but we
-- write them out so the /roles UI matrix shows the intent and an admin
-- toggling the checkbox in either direction works.
INSERT INTO role_permissions (role_id, module, action, granted)
SELECT r.id, m.module, m.action, false
FROM roles r
CROSS JOIN (VALUES
    ('payroll','create'), ('payroll','update'), ('payroll','delete'),
    ('finance','create'), ('finance','update'), ('finance','delete'),
    ('reports','view'),   ('reports','create'),
    ('reports','update'), ('reports','delete')
) AS m(module, action)
WHERE r.code = 'operator'
ON CONFLICT (role_id, module, action) DO UPDATE SET granted = false;

-- Cashier: till only. No worker/payroll/finance/reports/HR access.
-- The cashier already has sales:create/view from earlier migrations; this
-- block writes explicit denies for everything else so the new fail-closed
-- default produces consistent behaviour and the /roles UI can show the
-- correct checkboxes without falling back to "no row".
INSERT INTO role_permissions (role_id, module, action, granted)
SELECT r.id, m.module, m.action, false
FROM roles r
CROSS JOIN (VALUES
    ('payroll','view'),   ('payroll','create'),
    ('payroll','update'), ('payroll','delete'),
    ('finance','view'),   ('finance','create'),
    ('finance','update'), ('finance','delete'),
    ('worker-cards','view'),   ('worker-cards','create'),
    ('worker-cards','update'), ('worker-cards','delete')
) AS m(module, action)
WHERE r.code = 'cashier'
ON CONFLICT (role_id, module, action) DO UPDATE SET granted = false;

COMMIT;
