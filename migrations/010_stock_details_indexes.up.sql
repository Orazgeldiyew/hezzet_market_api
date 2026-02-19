-- Index for filtering by warehouse and ordering by date
CREATE INDEX IF NOT EXISTS idx_wid_created_at
ON warehouse_item_details (warehouse_id, created_at DESC);

-- Index for filtering by product and ordering by date
CREATE INDEX IF NOT EXISTS idx_wid_product_created
ON warehouse_item_details (product_id, created_at DESC);

-- Optional: index for type filtering
CREATE INDEX IF NOT EXISTS idx_wid_type_created
ON warehouse_item_details (type, created_at DESC);
