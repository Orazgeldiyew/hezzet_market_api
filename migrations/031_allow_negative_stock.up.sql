-- Allow warehouse_items.qty_milli to go negative (deficit / oversell scenario).
-- The application layer controls forced confirms via the force=true flag (any authenticated role).
ALTER TABLE warehouse_items
    DROP CONSTRAINT IF EXISTS warehouse_items_qty_milli_check;
