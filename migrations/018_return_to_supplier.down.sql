-- 018_return_to_supplier.down.sql

-- idx_wid_type_created was created by 010_stock_details_indexes, not by this migration — do not drop it here
DROP INDEX IF EXISTS idx_wid_supplier_id;
ALTER TABLE warehouse_item_details DROP COLUMN IF EXISTS note;
ALTER TABLE warehouse_item_details DROP COLUMN IF EXISTS supplier_id;
-- Cannot remove enum value in Postgres; 'return_to_supplier' remains harmless.
