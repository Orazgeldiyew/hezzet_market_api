BEGIN;

-- Down: revert to the original 2-value CHECK. If any 'purchase' rows have
-- accumulated, this rollback will fail (intentionally — losing that data
-- silently would be worse).
ALTER TABLE worker_debts
    DROP CONSTRAINT IF EXISTS worker_debts_type_check;

ALTER TABLE worker_debts
    ADD CONSTRAINT worker_debts_type_check
    CHECK (type IN ('advance', 'loan'));

COMMIT;
