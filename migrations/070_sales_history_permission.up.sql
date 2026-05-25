BEGIN;

-- Backfill sales:history grants for roles that should see /sales-history.
-- Cashier already has granted=false; this migration ensures admin/manager/
-- operator have the row so the /roles UI shows the checkbox to toggle.
INSERT INTO role_permissions (role_id, module, action, granted)
SELECT r.id, 'sales', 'history', (r.code IN ('admin','manager','operator'))
FROM roles r
ON CONFLICT (role_id, module, action) DO NOTHING;

COMMIT;
