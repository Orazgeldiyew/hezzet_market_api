-- 018_return_to_supplier.up.sql
-- Add return_to_supplier movement type, supplier_id and note columns.

ALTER TYPE movement_type ADD VALUE IF NOT EXISTS 'return_to_supplier';

ALTER TABLE warehouse_item_details
  ADD COLUMN IF NOT EXISTS supplier_id BIGINT REFERENCES suppliers(id);

ALTER TABLE warehouse_item_details
  ADD COLUMN IF NOT EXISTS note TEXT;

CREATE INDEX IF NOT EXISTS idx_wid_supplier_id
  ON warehouse_item_details (supplier_id) WHERE supplier_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_wid_type_created
  ON warehouse_item_details (type, created_at DESC);
