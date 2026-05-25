BEGIN;

-- Snapshot of which specific worker_debts were included in a payroll run.
-- Without this table, Pay's SettleDebts(worker_id) would close every
-- currently-open debt — including ones created AFTER Calculate ran — even
-- though the salary only deducted the debts that existed at Calculate time.
-- The join table lets Pay close exactly the debts the manager picked.
CREATE TABLE payroll_run_debts (
    payroll_run_id BIGINT NOT NULL REFERENCES payroll_runs(id) ON DELETE CASCADE,
    debt_id        BIGINT NOT NULL REFERENCES worker_debts(id),
    amount_cents   BIGINT NOT NULL CHECK (amount_cents > 0),
    PRIMARY KEY (payroll_run_id, debt_id)
);

CREATE INDEX idx_payroll_run_debts_debt ON payroll_run_debts(debt_id);

COMMIT;
