BEGIN;

CREATE TABLE inventory_counts (
    id           BIGSERIAL PRIMARY KEY,
    warehouse_id BIGINT NOT NULL REFERENCES warehouses(id),
    status       TEXT NOT NULL DEFAULT 'draft'
                 CHECK (status IN ('draft', 'confirmed', 'cancelled')),
    note         TEXT,
    created_by   BIGINT REFERENCES users(id),
    confirmed_by BIGINT REFERENCES users(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    confirmed_at TIMESTAMPTZ
);

CREATE INDEX idx_inventory_counts_warehouse ON inventory_counts(warehouse_id);
CREATE INDEX idx_inventory_counts_status ON inventory_counts(status);

CREATE TABLE inventory_count_items (
    id               BIGSERIAL PRIMARY KEY,
    count_id         BIGINT NOT NULL REFERENCES inventory_counts(id) ON DELETE CASCADE,
    product_id       BIGINT NOT NULL REFERENCES products(id),
    system_qty_milli BIGINT NOT NULL,
    actual_qty_milli BIGINT NOT NULL DEFAULT 0,
    diff_milli       BIGINT NOT NULL DEFAULT 0,
    UNIQUE(count_id, product_id)
);

CREATE INDEX idx_inventory_count_items_count ON inventory_count_items(count_id);

COMMIT;
