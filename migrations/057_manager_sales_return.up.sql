-- Grant sales:return permission to manager role
BEGIN;

INSERT INTO role_permissions (role_id, module, action, granted)
SELECT r.id, 'sales', 'return', true
FROM roles r
WHERE r.code = 'manager'
ON CONFLICT (role_id, module, action) DO UPDATE SET granted = true;

COMMIT;
