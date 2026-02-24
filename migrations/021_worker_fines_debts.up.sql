BEGIN;

-- ═══════════════════════════════════════════════════════════════
-- A) worker_fines — penalties
-- ═══════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS worker_fines (
    id           BIGSERIAL    PRIMARY KEY,
    worker_id    BIGINT       NOT NULL REFERENCES workers(id),
    amount_cents BIGINT       NOT NULL CHECK (amount_cents > 0),
    reason       TEXT         NOT NULL,
    status       TEXT         NOT NULL DEFAULT 'open'
                 CHECK (status IN ('open', 'deducted', 'canceled')),
    occurred_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by   BIGINT       REFERENCES users(id),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_worker_fines_worker_id ON worker_fines (worker_id);
CREATE INDEX IF NOT EXISTS idx_worker_fines_status    ON worker_fines (status);

CREATE OR REPLACE FUNCTION trg_worker_fines_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = now(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS worker_fines_set_updated_at ON worker_fines;
CREATE TRIGGER worker_fines_set_updated_at
    BEFORE UPDATE ON worker_fines FOR EACH ROW
    EXECUTE FUNCTION trg_worker_fines_updated_at();

-- ═══════════════════════════════════════════════════════════════
-- B) worker_debts — advances / loans given to worker
-- ═══════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS worker_debts (
    id              BIGSERIAL    PRIMARY KEY,
    worker_id       BIGINT       NOT NULL REFERENCES workers(id),
    amount_cents    BIGINT       NOT NULL CHECK (amount_cents > 0),
    remaining_cents BIGINT       NOT NULL CHECK (remaining_cents >= 0),
    type            TEXT         NOT NULL CHECK (type IN ('advance', 'loan')),
    status          TEXT         NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open', 'settled', 'canceled')),
    note            TEXT,
    created_by      BIGINT       REFERENCES users(id),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_worker_debts_worker_id ON worker_debts (worker_id);
CREATE INDEX IF NOT EXISTS idx_worker_debts_status    ON worker_debts (status);

CREATE OR REPLACE FUNCTION trg_worker_debts_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = now(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS worker_debts_set_updated_at ON worker_debts;
CREATE TRIGGER worker_debts_set_updated_at
    BEFORE UPDATE ON worker_debts FOR EACH ROW
    EXECUTE FUNCTION trg_worker_debts_updated_at();

COMMIT;
