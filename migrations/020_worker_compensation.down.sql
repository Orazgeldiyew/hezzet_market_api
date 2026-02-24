-- Revert transactions CHECK constraint
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_related_table_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_related_table_check
    CHECK (related_table IN ('sale', 'purchase', 'manual', 'adjustment'));

DROP TRIGGER IF EXISTS worker_compensation_set_updated_at ON worker_compensation;
DROP FUNCTION IF EXISTS trg_worker_compensation_updated_at();
DROP TABLE IF EXISTS worker_compensation;
