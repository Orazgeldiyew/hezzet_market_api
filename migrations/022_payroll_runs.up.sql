BEGIN;

CREATE TABLE IF NOT EXISTS payroll_runs (
    id                BIGSERIAL    PRIMARY KEY,
    worker_id         BIGINT       NOT NULL REFERENCES workers(id),
    period            TEXT         NOT NULL,
    base_salary_cents BIGINT       NOT NULL,
    fines_cents       BIGINT       NOT NULL DEFAULT 0,
    debts_cents       BIGINT       NOT NULL DEFAULT 0,
    net_salary_cents  BIGINT       NOT NULL,
    status            TEXT         NOT NULL DEFAULT 'calculated'
                      CHECK (status IN ('calculated', 'paid', 'canceled')),
    transaction_id    BIGINT       REFERENCES transactions(id),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_payroll_runs_worker_period
    ON payroll_runs (worker_id, period);

CREATE INDEX IF NOT EXISTS idx_payroll_runs_worker_id ON payroll_runs (worker_id);
CREATE INDEX IF NOT EXISTS idx_payroll_runs_status    ON payroll_runs (status);
CREATE INDEX IF NOT EXISTS idx_payroll_runs_period    ON payroll_runs (period);

CREATE OR REPLACE FUNCTION trg_payroll_runs_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = now(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS payroll_runs_set_updated_at ON payroll_runs;
CREATE TRIGGER payroll_runs_set_updated_at
    BEFORE UPDATE ON payroll_runs FOR EACH ROW
    EXECUTE FUNCTION trg_payroll_runs_updated_at();

COMMIT;
