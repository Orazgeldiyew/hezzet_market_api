-- Price history: tracks every purchase_price / sale_price change
CREATE TABLE price_history (
    id               BIGSERIAL PRIMARY KEY,
    product_id       BIGINT NOT NULL REFERENCES products(id),
    field            TEXT NOT NULL CHECK (field IN ('purchase_price', 'sale_price')),
    old_value        BIGINT NOT NULL,
    new_value        BIGINT NOT NULL,
    changed_by       BIGINT REFERENCES users(id),
    changed_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_price_history_product ON price_history(product_id);
CREATE INDEX idx_price_history_changed_at ON price_history(changed_at DESC);
