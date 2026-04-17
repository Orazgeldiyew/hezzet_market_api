BEGIN;

CREATE TABLE supplier_returns (
    id              BIGSERIAL PRIMARY KEY,
    supplier_id     BIGINT NOT NULL REFERENCES suppliers(id),
    warehouse_id    BIGINT NOT NULL REFERENCES warehouses(id),
    status          TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'confirmed', 'cancelled')),
    total_cents     BIGINT NOT NULL DEFAULT 0,
    items_count     INT NOT NULL DEFAULT 0,
    note            TEXT,
    created_by      BIGINT REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    confirmed_at    TIMESTAMPTZ,
    confirmed_by    BIGINT REFERENCES users(id)
);

CREATE INDEX idx_supplier_returns_supplier ON supplier_returns(supplier_id);
CREATE INDEX idx_supplier_returns_status ON supplier_returns(status);

CREATE TABLE supplier_return_items (
    id              BIGSERIAL PRIMARY KEY,
    return_id       BIGINT NOT NULL REFERENCES supplier_returns(id),
    product_id      BIGINT NOT NULL REFERENCES products(id),
    qty_milli       BIGINT NOT NULL CHECK (qty_milli > 0),
    unit_cost_cents BIGINT NOT NULL CHECK (unit_cost_cents >= 0),
    line_total_cents BIGINT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_supplier_return_items_return ON supplier_return_items(return_id);

COMMIT;
