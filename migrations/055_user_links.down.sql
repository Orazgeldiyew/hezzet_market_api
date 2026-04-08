DROP INDEX IF EXISTS idx_workers_user_id;
DROP INDEX IF EXISTS idx_customers_user_id;
DROP INDEX IF EXISTS idx_suppliers_user_id;
ALTER TABLE workers DROP COLUMN IF EXISTS user_id;
ALTER TABLE customers DROP COLUMN IF EXISTS user_id;
ALTER TABLE suppliers DROP COLUMN IF EXISTS user_id;
