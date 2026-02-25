-- 025 up: Sales module tables

CREATE TABLE IF NOT EXISTS sales (
    id           BIGSERIAL   PRIMARY KEY,
    warehouse_id BIGINT      NOT NULL REFERENCES warehouses(id),
    customer_id  BIGINT      REFERENCES customers(id),
    total_cents  BIGINT      NOT NULL CHECK (total_cents >= 0),
    cost_cents   BIGINT      NOT NULL DEFAULT 0,
    items_count  INT         NOT NULL CHECK (items_count > 0),
    note         TEXT,
    created_by   BIGINT      REFERENCES users(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sales_created_at ON sales (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sales_warehouse_id ON sales (warehouse_id);
CREATE INDEX IF NOT EXISTS idx_sales_customer_id ON sales (customer_id) WHERE customer_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS sale_items (
    id               BIGSERIAL   PRIMARY KEY,
    sale_id          BIGINT      NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
    product_id       BIGINT      NOT NULL REFERENCES products(id),
    qty_milli        BIGINT      NOT NULL CHECK (qty_milli > 0),
    unit_price_cents BIGINT      NOT NULL CHECK (unit_price_cents >= 0),
    cost_cents       BIGINT      NOT NULL DEFAULT 0,
    line_total_cents BIGINT      NOT NULL CHECK (line_total_cents >= 0),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sale_items_sale_id ON sale_items (sale_id);
CREATE INDEX IF NOT EXISTS idx_sale_items_product_id ON sale_items (product_id);
