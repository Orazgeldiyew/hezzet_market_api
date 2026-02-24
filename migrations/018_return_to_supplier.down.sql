-- 018_return_to_supplier.down.sql

DROP INDEX IF EXISTS idx_wid_type_created;
DROP INDEX IF EXISTS idx_wid_supplier_id;
ALTER TABLE warehouse_item_details DROP COLUMN IF EXISTS note;
ALTER TABLE warehouse_item_details DROP COLUMN IF EXISTS supplier_id;
-- Cannot remove enum value in Postgres; 'return_to_supplier' remains harmless.
