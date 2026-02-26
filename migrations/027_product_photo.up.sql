BEGIN;

CREATE TABLE IF NOT EXISTS product_photos (
    id          BIGSERIAL   PRIMARY KEY,
    product_id  BIGINT      NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    ext         TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_product_photos_product_id ON product_photos (product_id);

ALTER TABLE products ADD COLUMN IF NOT EXISTS photo_path TEXT;

COMMIT;
