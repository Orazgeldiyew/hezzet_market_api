BEGIN;

-- Enable trigram matching so product search tolerates typos and ranks results
-- by similarity. Critical above ~10k products where ILIKE '%x%' is a full scan.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- GIN indexes for similarity() and ILIKE over the searched columns.
CREATE INDEX IF NOT EXISTS idx_products_name_trgm
    ON products USING GIN (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_products_sku_trgm
    ON products USING GIN (sku gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_product_barcodes_barcode_trgm
    ON product_barcodes USING GIN (barcode gin_trgm_ops);

COMMIT;
