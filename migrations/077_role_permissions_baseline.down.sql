BEGIN;

-- Rolls back the explicit grants added in 077. Removes only the rows we
-- inserted; rows seeded by earlier migrations stay untouched.
DELETE FROM role_permissions rp USING roles r
WHERE rp.role_id = r.id
  AND (
    (r.code = 'manager'  AND ((rp.module, rp.action) IN (
        ('payroll','view'),('payroll','create'),('payroll','update'),
        ('finance','view'),('finance','create'),('finance','update'),
        ('worker-cards','view'),('worker-cards','create'),
        ('worker-cards','update'),('worker-cards','delete'))))
    OR
    (r.code = 'operator' AND ((rp.module, rp.action) IN (
        ('payroll','view'),('payroll','create'),('payroll','update'),('payroll','delete'),
        ('finance','view'),('finance','create'),('finance','update'),('finance','delete'),
        ('reports','view'),('reports','create'),('reports','update'),('reports','delete'),
        ('worker-cards','view'))))
    OR
    (r.code = 'cashier'  AND ((rp.module, rp.action) IN (
        ('payroll','view'),('payroll','create'),('payroll','update'),('payroll','delete'),
        ('finance','view'),('finance','create'),('finance','update'),('finance','delete'),
        ('worker-cards','view'),('worker-cards','create'),
        ('worker-cards','update'),('worker-cards','delete'))))
  );

COMMIT;
