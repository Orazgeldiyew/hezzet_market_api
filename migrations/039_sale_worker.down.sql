ALTER TABLE sales DROP COLUMN IF EXISTS worker_id;
ALTER TABLE worker_debts DROP CONSTRAINT IF EXISTS worker_debts_type_check;
ALTER TABLE worker_debts ADD CONSTRAINT worker_debts_type_check
    CHECK (type IN ('advance', 'loan'));
