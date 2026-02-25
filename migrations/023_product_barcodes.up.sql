-- 023: Move barcode from products column to separate product_barcodes table (multi-barcode support)

CREATE TABLE product_barcodes (
    id         BIGSERIAL PRIMARY KEY,
    product_id BIGINT  NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    barcode    TEXT    NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (barcode)
);

CREATE INDEX idx_product_barcodes_product_id ON product_barcodes(product_id);

-- Migrate existing barcodes
INSERT INTO product_barcodes (product_id, barcode)
SELECT id, barcode FROM products
WHERE barcode IS NOT NULL AND barcode != '';

-- Drop old column
ALTER TABLE products DROP COLUMN barcode;
