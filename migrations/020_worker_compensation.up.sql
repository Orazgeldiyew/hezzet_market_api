BEGIN;

-- ═══════════════════════════════════════════════════════════════
-- A) worker_compensation — base salary configuration per worker
-- ═══════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS worker_compensation (
    id               BIGSERIAL    PRIMARY KEY,
    worker_id        BIGINT       UNIQUE NOT NULL REFERENCES workers(id),
    base_salary_cents BIGINT      NOT NULL CHECK (base_salary_cents > 0),
    pay_day          INT          NOT NULL CHECK (pay_day BETWEEN 1 AND 31),
    is_active        BOOLEAN      NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_worker_compensation_worker_id
    ON worker_compensation (worker_id);

-- updated_at trigger
CREATE OR REPLACE FUNCTION trg_worker_compensation_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = now(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS worker_compensation_set_updated_at ON worker_compensation;
CREATE TRIGGER worker_compensation_set_updated_at
    BEFORE UPDATE ON worker_compensation FOR EACH ROW
    EXECUTE FUNCTION trg_worker_compensation_updated_at();

-- ═══════════════════════════════════════════════════════════════
-- B) Extend transactions.related_table CHECK to allow payroll + worker_debt
-- ═══════════════════════════════════════════════════════════════
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_related_table_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_related_table_check
    CHECK (related_table IN ('sale', 'purchase', 'manual', 'adjustment', 'payroll', 'worker_debt'));

COMMIT;
