BEGIN;

-- =========================
-- Customers: numeric -> bigint cents
-- =========================

ALTER TABLE customers ADD COLUMN IF NOT EXISTS total_spent_cents  BIGINT;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS bonus_points_cents BIGINT;

UPDATE customers
SET
  total_spent_cents  = COALESCE(ROUND(total_spent  * 100)::BIGINT, 0),
  bonus_points_cents = COALESCE(ROUND(bonus_points * 100)::BIGINT, 0)
WHERE total_spent_cents IS NULL OR bonus_points_cents IS NULL;

ALTER TABLE customers
  ALTER COLUMN total_spent_cents  SET NOT NULL,
  ALTER COLUMN total_spent_cents  SET DEFAULT 0,
  ALTER COLUMN bonus_points_cents SET NOT NULL,
  ALTER COLUMN bonus_points_cents SET DEFAULT 0;

ALTER TABLE customers
  DROP CONSTRAINT IF EXISTS customers_total_spent_check,
  DROP CONSTRAINT IF EXISTS customers_bonus_points_check;

ALTER TABLE customers
  ADD CONSTRAINT customers_total_spent_check CHECK (total_spent_cents >= 0),
  ADD CONSTRAINT customers_bonus_points_check CHECK (bonus_points_cents >= 0);

ALTER TABLE customers
  DROP COLUMN total_spent,
  DROP COLUMN bonus_points;

ALTER TABLE customers RENAME COLUMN total_spent_cents  TO total_spent;
ALTER TABLE customers RENAME COLUMN bonus_points_cents TO bonus_points;

-- =========================
-- Products: numeric -> bigint cents
-- =========================

ALTER TABLE products ADD COLUMN IF NOT EXISTS purchase_price_cents BIGINT;
ALTER TABLE products ADD COLUMN IF NOT EXISTS sale_price_cents     BIGINT;

UPDATE products
SET
  purchase_price_cents = COALESCE(ROUND(purchase_price * 100)::BIGINT, 0),
  sale_price_cents     = COALESCE(ROUND(sale_price     * 100)::BIGINT, 0)
WHERE purchase_price_cents IS NULL OR sale_price_cents IS NULL;

ALTER TABLE products
  ALTER COLUMN purchase_price_cents SET NOT NULL,
  ALTER COLUMN purchase_price_cents SET DEFAULT 0,
  ALTER COLUMN sale_price_cents     SET NOT NULL,
  ALTER COLUMN sale_price_cents     SET DEFAULT 0;

ALTER TABLE products
  DROP CONSTRAINT IF EXISTS products_purchase_price_check,
  DROP CONSTRAINT IF EXISTS products_sale_price_check;

ALTER TABLE products
  ADD CONSTRAINT products_purchase_price_check CHECK (purchase_price_cents >= 0),
  ADD CONSTRAINT products_sale_price_check CHECK (sale_price_cents >= 0);

ALTER TABLE products
  DROP COLUMN purchase_price,
  DROP COLUMN sale_price;

ALTER TABLE products RENAME COLUMN purchase_price_cents TO purchase_price;
ALTER TABLE products RENAME COLUMN sale_price_cents     TO sale_price;

COMMIT;
