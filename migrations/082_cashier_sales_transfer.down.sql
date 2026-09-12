UPDATE role_permissions rp
SET granted = false
FROM roles r
WHERE rp.role_id = r.id
  AND r.code IN ('cashier', 'manager', 'operator')
  AND rp.module = 'sales'
  AND rp.action = 'transfer';
