-- Cashiers must be able to hand a draft cart to another till («Переданные мне»).
-- sales:transfer was added as an action in 044 but never seeded for cashier;
-- fail-closed RBAC therefore hid the transfer button / rejected the API.
INSERT INTO role_permissions (role_id, module, action, granted)
SELECT r.id, 'sales', 'transfer', true
FROM roles r
WHERE r.code IN ('cashier', 'manager', 'operator')
ON CONFLICT (role_id, module, action) DO UPDATE SET granted = true;
