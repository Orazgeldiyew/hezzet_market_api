BEGIN;

-- Removes the rows inserted by 078. Existing rows from earlier migrations
-- stay; we only delete (role, module, action) tuples that 078 introduced.
DELETE FROM role_permissions rp USING roles r
WHERE rp.role_id = r.id AND (
    (r.code = 'manager' AND ((rp.module, rp.action) IN (
        ('categories','history'),('customers','history'),('finance','history'),
        ('payroll','history'),('products','history'),('purchases','history'),
        ('reports','history'),('stock','history'),('suppliers','history'),
        ('warehouses','history'),('worker-cards','history'),('workers','history'),
        ('categories','import'),('customers','import'),('products','import'),
        ('purchases','import'),('stock','import'),('suppliers','import'),
        ('warehouses','import'),('workers','import'),('sales','import'),
        ('finance','import'),('payroll','import'),('reports','import'),
        ('worker-cards','import')
    )))
    OR
    (r.code = 'operator' AND ((rp.module, rp.action) IN (
        ('categories','view'),('categories','create'),('categories','update'),
        ('categories','delete'),('categories','history'),('categories','import'),
        ('customers','history'),('customers','import'),
        ('finance','history'),('finance','import'),
        ('payroll','history'),('payroll','import'),
        ('products','history'),
        ('purchases','history'),('purchases','import'),
        ('reports','history'),('reports','import'),
        ('sales','import'),
        ('stock','history'),('stock','import'),
        ('suppliers','view'),('suppliers','create'),('suppliers','update'),
        ('suppliers','delete'),('suppliers','history'),('suppliers','import'),
        ('warehouses','view'),('warehouses','create'),('warehouses','update'),
        ('warehouses','delete'),('warehouses','history'),('warehouses','import'),
        ('worker-cards','view'),('worker-cards','create'),('worker-cards','update'),
        ('worker-cards','delete'),('worker-cards','history'),('worker-cards','import'),
        ('workers','history'),('workers','import')
    )))
    OR
    (r.code = 'cashier' AND ((rp.module, rp.action) IN (
        ('categories','import'),('customers','import'),('finance','import'),
        ('payroll','import'),('purchases','import'),('reports','import'),
        ('sales','import'),('stock','import'),('suppliers','import'),
        ('warehouses','import'),('worker-cards','import'),('workers','import')
    )))
);

COMMIT;
