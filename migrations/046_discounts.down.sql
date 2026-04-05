ALTER TABLE sale_items DROP COLUMN IF EXISTS discount_percent;
ALTER TABLE sales DROP COLUMN IF EXISTS discount_percent;
ALTER TABLE sales DROP COLUMN IF EXISTS discount_cents;
