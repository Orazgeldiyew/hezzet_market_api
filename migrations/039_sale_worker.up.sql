-- Add worker_id to sales (worker purchasing on credit)
ALTER TABLE sales ADD COLUMN worker_id BIGINT REFERENCES workers(id);
CREATE INDEX idx_sales_worker_id ON sales(worker_id) WHERE worker_id IS NOT NULL;

-- Extend worker_debts type to include 'purchase'
ALTER TABLE worker_debts DROP CONSTRAINT IF EXISTS worker_debts_type_check;
ALTER TABLE worker_debts ADD CONSTRAINT worker_debts_type_check
    CHECK (type IN ('advance', 'loan', 'purchase'));
