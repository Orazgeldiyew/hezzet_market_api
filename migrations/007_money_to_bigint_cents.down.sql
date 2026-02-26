BEGIN;

-- Customers: bigint cents -> numeric(14,2)
ALTER TABLE customers
  ADD COLUMN IF NOT EXISTS total_spent_num  NUMERIC(14,2),
  ADD COLUMN IF NOT EXISTS bonus_points_num NUMERIC(14,2);

UPDATE customers
SET
  total_spent_num  = (total_spent  / 100.0),
  bonus_points_num = (bonus_points / 100.0)
WHERE total_spent_num IS NULL OR bonus_points_num IS NULL;

ALTER TABLE customers
  DROP CONSTRAINT IF EXISTS customers_total_spent_check,
  DROP CONSTRAINT IF EXISTS customers_bonus_points_check;

ALTER TABLE customers
  DROP COLUMN total_spent,
  DROP COLUMN bonus_points;

-- PostgreSQL requires one RENAME COLUMN per ALTER TABLE statement
ALTER TABLE customers RENAME COLUMN total_spent_num  TO total_spent;
ALTER TABLE customers RENAME COLUMN bonus_points_num TO bonus_points;

ALTER TABLE customers
  ADD CONSTRAINT customers_total_spent_check  CHECK (total_spent  >= 0),
  ADD CONSTRAINT customers_bonus_points_check CHECK (bonus_points >= 0);


-- Products: bigint cents -> numeric(14,2)
ALTER TABLE products
  ADD COLUMN IF NOT EXISTS purchase_price_num NUMERIC(14,2),
  ADD COLUMN IF NOT EXISTS sale_price_num     NUMERIC(14,2);

UPDATE products
SET
  purchase_price_num = (purchase_price / 100.0),
  sale_price_num     = (sale_price     / 100.0)
WHERE purchase_price_num IS NULL OR sale_price_num IS NULL;

ALTER TABLE products
  DROP CONSTRAINT IF EXISTS products_purchase_price_check,
  DROP CONSTRAINT IF EXISTS products_sale_price_check;

ALTER TABLE products
  DROP COLUMN purchase_price,
  DROP COLUMN sale_price;

-- PostgreSQL requires one RENAME COLUMN per ALTER TABLE statement
ALTER TABLE products RENAME COLUMN purchase_price_num TO purchase_price;
ALTER TABLE products RENAME COLUMN sale_price_num     TO sale_price;

ALTER TABLE products
  ADD CONSTRAINT products_purchase_price_check CHECK (purchase_price >= 0),
  ADD CONSTRAINT products_sale_price_check     CHECK (sale_price     >= 0);

COMMIT;
