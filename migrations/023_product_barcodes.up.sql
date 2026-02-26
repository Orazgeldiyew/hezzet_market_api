-- 023: Move barcode from products column to separate product_barcodes table (multi-barcode support)

CREATE TABLE IF NOT EXISTS product_barcodes (
    id         BIGSERIAL PRIMARY KEY,
    product_id BIGINT  NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    barcode    TEXT    NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (barcode)
);

CREATE INDEX IF NOT EXISTS idx_product_barcodes_product_id ON product_barcodes(product_id);

-- Migrate existing barcodes (only if barcode column still exists on products)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'products' AND column_name = 'barcode'
    ) THEN
        INSERT INTO product_barcodes (product_id, barcode)
        SELECT id, barcode FROM products
        WHERE barcode IS NOT NULL AND barcode != ''
        ON CONFLICT (barcode) DO NOTHING;

        ALTER TABLE products DROP COLUMN barcode;
    END IF;
END
$$;
