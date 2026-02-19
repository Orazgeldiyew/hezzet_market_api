ALTER TABLE warehouse_items
  DROP COLUMN IF EXISTS avg_cost_cents,
  DROP COLUMN IF EXISTS total_cost_cents;

DROP INDEX IF EXISTS idx_warehouse_items_wh_prod;
