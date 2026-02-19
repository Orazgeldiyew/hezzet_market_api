CREATE INDEX IF NOT EXISTS idx_wid_wh_prod_type_created_id
ON warehouse_item_details (warehouse_id, product_id, type, created_at DESC, id DESC);
