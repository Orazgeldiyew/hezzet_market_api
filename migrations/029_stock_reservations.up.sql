-- Outside transaction — Postgres requires ADD VALUE outside a transaction block
ALTER TYPE movement_type ADD VALUE IF NOT EXISTS 'sale_return';

BEGIN;

-- 1. status on sales (existing rows default to 'confirmed' — already fulfilled)
ALTER TABLE sales
  ADD COLUMN status TEXT NOT NULL DEFAULT 'confirmed'
                   CHECK (status IN ('draft', 'confirmed', 'cancelled'));

CREATE INDEX idx_sales_status ON sales(status);

-- 2. Reservations ledger
CREATE TABLE stock_reservations (
    id           BIGSERIAL    PRIMARY KEY,
    sale_id      BIGINT       NOT NULL REFERENCES sales(id),
    warehouse_id BIGINT       NOT NULL REFERENCES warehouses(id),
    product_id   BIGINT       NOT NULL REFERENCES products(id),
    qty_milli    BIGINT       NOT NULL CHECK (qty_milli > 0),
    status       TEXT         NOT NULL DEFAULT 'active'
                              CHECK (status IN ('active', 'released', 'fulfilled')),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    released_at  TIMESTAMPTZ
);

CREATE INDEX idx_stock_res_sale_id ON stock_reservations(sale_id);
-- Partial index — only active rows matter for availability calculations
CREATE INDEX idx_stock_res_active ON stock_reservations(warehouse_id, product_id)
  WHERE status = 'active';

COMMIT;
