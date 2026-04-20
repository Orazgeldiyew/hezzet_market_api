BEGIN;

ALTER TABLE products
    DROP COLUMN IF EXISTS safety_stock_milli,
    DROP COLUMN IF EXISTS lead_time_days;

COMMIT;
