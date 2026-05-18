BEGIN;

-- Optional per-line sale price captured at PO time. When > 0, ReceivePO will
-- propagate it to products.sale_price so the cashier sees the new price as
-- soon as the receipt is confirmed. When 0, the product's existing sale_price
-- is left untouched and a manager can update it later via /api/products.
ALTER TABLE purchase_order_items
    ADD COLUMN IF NOT EXISTS sale_price_cents BIGINT NOT NULL DEFAULT 0
        CHECK (sale_price_cents >= 0);

COMMIT;
