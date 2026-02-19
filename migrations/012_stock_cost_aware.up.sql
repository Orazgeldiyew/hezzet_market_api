ALTER TABLE warehouse_items
  ADD COLUMN IF NOT EXISTS avg_cost_cents   BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS total_cost_cents BIGINT NOT NULL DEFAULT 0;

-- (опционально) если хочешь — чтобы запросы баланса были быстрее по фильтрам
CREATE INDEX IF NOT EXISTS idx_warehouse_items_wh_prod
  ON warehouse_items (warehouse_id, product_id);
