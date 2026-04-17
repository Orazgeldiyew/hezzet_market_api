-- Grant purchases:return permission to manager role (used for supplier returns)
BEGIN;

INSERT INTO role_permissions (role_id, module, action, granted)
SELECT r.id, 'purchases', 'return', true
FROM roles r
WHERE r.code = 'manager'
ON CONFLICT (role_id, module, action) DO UPDATE SET granted = true;

COMMIT;
