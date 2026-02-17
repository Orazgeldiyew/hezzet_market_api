BEGIN;

-- ================
-- 1) Warehouse stock movements (source of truth)
-- ================
CREATE TABLE IF NOT EXISTS stock_movements (
  id            BIGSERIAL PRIMARY KEY,
  product_id    BIGINT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
  movement_type TEXT   NOT NULL CHECK (movement_type IN ('in','out','adjust')),
  qty           NUMERIC(14,3) NOT NULL, -- quantity can be fractional (e.g., kg, liters)
  note          TEXT,
  ref_type      TEXT,   -- e.g. 'sale', 'purchase', 'manual'
  ref_id        BIGINT, -- id of sale/purchase/etc
  created_by    BIGINT, -- later can reference users(id) if you want FK
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_stock_movements_product_id ON stock_movements(product_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_created_at ON stock_movements(created_at);
CREATE INDEX IF NOT EXISTS idx_stock_movements_ref ON stock_movements(ref_type, ref_id);

-- A safe stock view (not a table): stock = sum(in/adjust) - sum(out)
CREATE OR REPLACE VIEW v_product_stock AS
SELECT
  p.id AS product_id,
  COALESCE(SUM(
    CASE
      WHEN sm.movement_type = 'in' THEN sm.qty
      WHEN sm.movement_type = 'out' THEN -sm.qty
      WHEN sm.movement_type = 'adjust' THEN sm.qty
      ELSE 0
    END
  ), 0) AS stock
FROM products p
LEFT JOIN stock_movements sm ON sm.product_id = p.id
GROUP BY p.id;

-- ================
-- 2) Sales (money in cents)
-- ================
CREATE TABLE IF NOT EXISTS sales (
  id               BIGSERIAL PRIMARY KEY,
  customer_id      BIGINT NULL REFERENCES customers(id) ON DELETE SET NULL,
  total_amount     BIGINT NOT NULL DEFAULT 0 CHECK (total_amount >= 0), -- cents
  discount_amount  BIGINT NOT NULL DEFAULT 0 CHECK (discount_amount >= 0), -- cents
  bonus_used       BIGINT NOT NULL DEFAULT 0 CHECK (bonus_used >= 0), -- cents/points (choose meaning)
  net_amount       BIGINT NOT NULL DEFAULT 0 CHECK (net_amount >= 0), -- cents
  status           TEXT   NOT NULL DEFAULT 'paid' CHECK (status IN ('draft','paid','void')),
  note             TEXT,
  created_by       BIGINT,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sales_created_at ON sales(created_at);
CREATE INDEX IF NOT EXISTS idx_sales_customer_id ON sales(customer_id);
CREATE INDEX IF NOT EXISTS idx_sales_status ON sales(status);

CREATE TABLE IF NOT EXISTS sale_items (
  id               BIGSERIAL PRIMARY KEY,
  sale_id          BIGINT NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
  product_id       BIGINT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
  qty              NUMERIC(14,3) NOT NULL CHECK (qty > 0),
  unit_price       BIGINT NOT NULL CHECK (unit_price >= 0), -- cents
  line_total       BIGINT NOT NULL CHECK (line_total >= 0)  -- cents
);

CREATE INDEX IF NOT EXISTS idx_sale_items_sale_id ON sale_items(sale_id);
CREATE INDEX IF NOT EXISTS idx_sale_items_product_id ON sale_items(product_id);

COMMIT;
