BEGIN;
ALTER TABLE purchase_order_items DROP COLUMN IF EXISTS sale_price_cents;
COMMIT;
