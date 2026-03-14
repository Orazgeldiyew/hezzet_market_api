-- Sales: filter by created_by (cashier/operator reports)
CREATE INDEX IF NOT EXISTS idx_sales_created_by
    ON sales(created_by)
    WHERE created_by IS NOT NULL;

-- Sales: status + time ordering (common list query pattern)
CREATE INDEX IF NOT EXISTS idx_sales_status_created_at
    ON sales(status, created_at DESC);

-- Sales: warehouse + status (filter by warehouse and status together)
CREATE INDEX IF NOT EXISTS idx_sales_warehouse_status
    ON sales(warehouse_id, status)
    WHERE warehouse_id IS NOT NULL;

-- Purchase orders: warehouse + status (list by warehouse)
CREATE INDEX IF NOT EXISTS idx_purchase_orders_warehouse_status
    ON purchase_orders(warehouse_id, status)
    WHERE warehouse_id IS NOT NULL;

-- Stock movements: warehouse + type + time (stock report filtering)
CREATE INDEX IF NOT EXISTS idx_wid_warehouse_type_time
    ON warehouse_item_details(warehouse_id, type, created_at DESC);

-- Transactions: created_by (finance reports per user)
CREATE INDEX IF NOT EXISTS idx_transactions_created_by
    ON transactions(created_by)
    WHERE created_by IS NOT NULL;
