-- 004_customers.down.sql
-- Drop table first, then drop the type it depends on.

DROP TABLE IF EXISTS customers;
DROP TYPE  IF EXISTS customer_type;
