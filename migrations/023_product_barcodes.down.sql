-- 023 down: Restore single barcode column on products

ALTER TABLE products ADD COLUMN barcode TEXT UNIQUE;

-- Copy first barcode back per product
UPDATE products p
SET barcode = pb.barcode
FROM (
    SELECT DISTINCT ON (product_id) product_id, barcode
    FROM product_barcodes
    ORDER BY product_id, id
) pb
WHERE p.id = pb.product_id;

DROP TABLE IF EXISTS product_barcodes;
