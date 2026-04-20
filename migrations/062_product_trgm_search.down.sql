BEGIN;

DROP INDEX IF EXISTS idx_product_barcodes_barcode_trgm;
DROP INDEX IF EXISTS idx_products_sku_trgm;
DROP INDEX IF EXISTS idx_products_name_trgm;

-- pg_trgm extension itself is left installed — other tables may rely on it.

COMMIT;
