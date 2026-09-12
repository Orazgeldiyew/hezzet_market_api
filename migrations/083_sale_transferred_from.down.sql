DROP INDEX IF EXISTS idx_sales_transferred_from;
ALTER TABLE sales DROP COLUMN IF EXISTS transferred_from;
