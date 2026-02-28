-- Restore the non-negative constraint.
-- WARNING: will fail if any row already has qty_milli < 0.
ALTER TABLE warehouse_items
    ADD CONSTRAINT warehouse_items_qty_milli_check CHECK (qty_milli >= 0);
