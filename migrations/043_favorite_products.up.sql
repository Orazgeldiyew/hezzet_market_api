CREATE TABLE favorite_products (
    id         BIGSERIAL    PRIMARY KEY,
    user_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id BIGINT       NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    position   INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE(user_id, product_id)
);

CREATE INDEX idx_favorite_products_user ON favorite_products(user_id, position);
