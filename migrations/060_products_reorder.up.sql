BEGIN;

-- Dynamic reorder-point inputs per product.
-- lead_time_days     — how long the supplier takes to deliver a new order
-- safety_stock_milli — buffer kept on top of the reorder point

ALTER TABLE products
    ADD COLUMN IF NOT EXISTS lead_time_days     INT    NOT NULL DEFAULT 3
        CHECK (lead_time_days >= 0),
    ADD COLUMN IF NOT EXISTS safety_stock_milli BIGINT NOT NULL DEFAULT 0
        CHECK (safety_stock_milli >= 0);

COMMIT;
