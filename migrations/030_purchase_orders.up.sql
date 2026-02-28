-- Outside transaction — Postgres requires ADD VALUE outside a transaction block
ALTER TYPE movement_type ADD VALUE IF NOT EXISTS 'purchase';

BEGIN;

-- ═══════════════════════════════════════════════════════════════
-- A) purchase_orders — header
-- ═══════════════════════════════════════════════════════════════
CREATE TABLE purchase_orders (
    id           BIGSERIAL    PRIMARY KEY,
    supplier_id  BIGINT       NOT NULL REFERENCES suppliers(id),
    warehouse_id BIGINT       NOT NULL REFERENCES warehouses(id),
    status       TEXT         NOT NULL DEFAULT 'draft'
                              CHECK (status IN ('draft', 'received', 'cancelled')),
    total_cents  BIGINT       NOT NULL DEFAULT 0 CHECK (total_cents >= 0),
    items_count  INT          NOT NULL DEFAULT 0,
    note         TEXT,
    created_by   BIGINT       REFERENCES users(id),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    received_at  TIMESTAMPTZ,
    received_by  BIGINT       REFERENCES users(id)
);

CREATE INDEX idx_po_supplier_id ON purchase_orders(supplier_id);
CREATE INDEX idx_po_status      ON purchase_orders(status);
CREATE INDEX idx_po_created_at  ON purchase_orders(created_at DESC);

-- ═══════════════════════════════════════════════════════════════
-- B) purchase_order_items — line items with price snapshot
-- ═══════════════════════════════════════════════════════════════
CREATE TABLE purchase_order_items (
    id               BIGSERIAL    PRIMARY KEY,
    po_id            BIGINT       NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    product_id       BIGINT       NOT NULL REFERENCES products(id),
    qty_milli        BIGINT       NOT NULL CHECK (qty_milli > 0),
    unit_cost_cents  BIGINT       NOT NULL CHECK (unit_cost_cents >= 0),
    line_total_cents BIGINT       NOT NULL CHECK (line_total_cents >= 0),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_poi_po_id      ON purchase_order_items(po_id);
CREATE INDEX idx_poi_product_id ON purchase_order_items(product_id);

COMMIT;
