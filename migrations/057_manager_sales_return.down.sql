BEGIN;

DELETE FROM role_permissions
WHERE module = 'sales' AND action = 'return'
  AND role_id IN (SELECT id FROM roles WHERE code = 'manager');

COMMIT;
