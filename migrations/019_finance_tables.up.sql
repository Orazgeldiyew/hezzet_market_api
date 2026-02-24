BEGIN;

-- ═══════════════════════════════════════════════════════════════
-- A) payment_types — lookup table for payment methods
-- ═══════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS payment_types (
    id         BIGSERIAL    PRIMARY KEY,
    code       TEXT         UNIQUE NOT NULL,
    name       TEXT         NOT NULL,
    is_active  BOOLEAN      NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Seed default payment types (idempotent)
INSERT INTO payment_types (code, name) VALUES
    ('cash',          'Cash'),
    ('card',          'Card'),
    ('bank_transfer', 'Bank Transfer'),
    ('debt',          'Debt'),
    ('other',         'Other')
ON CONFLICT (code) DO NOTHING;

-- ═══════════════════════════════════════════════════════════════
-- B) transactions — the financial ledger
-- ═══════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS transactions (
    id                        BIGSERIAL    PRIMARY KEY,
    payment_type_id           BIGINT       REFERENCES payment_types(id),
    reason                    TEXT,
    status                    TEXT         NOT NULL DEFAULT 'pending'
                              CHECK (status IN ('pending', 'partial', 'paid', 'canceled')),
    amount_cents              BIGINT       NOT NULL,
    type                      TEXT         NOT NULL
                              CHECK (type IN ('income', 'expense')),
    table_payment_id          TEXT,
    warehouse_item_detail_id  BIGINT       REFERENCES warehouse_item_details(id),
    related_table             TEXT         NOT NULL
                              CHECK (related_table IN ('sale', 'purchase', 'manual', 'adjustment')),
    related_id                BIGINT,
    created_by                BIGINT       REFERENCES users(id),
    created_at                TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_transactions_created_at
    ON transactions (created_at);

CREATE INDEX IF NOT EXISTS idx_transactions_type_created_at
    ON transactions (type, created_at);

CREATE INDEX IF NOT EXISTS idx_transactions_status
    ON transactions (status);

CREATE INDEX IF NOT EXISTS idx_transactions_related
    ON transactions (related_table, related_id);

CREATE INDEX IF NOT EXISTS idx_transactions_wid
    ON transactions (warehouse_item_detail_id)
    WHERE warehouse_item_detail_id IS NOT NULL;

-- Auto-update updated_at on every row modification (pattern from 017_sms_logs).
CREATE OR REPLACE FUNCTION trg_transactions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS transactions_set_updated_at ON transactions;
CREATE TRIGGER transactions_set_updated_at
    BEFORE UPDATE ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION trg_transactions_updated_at();

-- ═══════════════════════════════════════════════════════════════
-- C) payments — individual payment events against a transaction
-- ═══════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS payments (
    id              BIGSERIAL    PRIMARY KEY,
    transaction_id  BIGINT       NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    payment_type_id BIGINT       NOT NULL REFERENCES payment_types(id),
    amount_cents    BIGINT       NOT NULL CHECK (amount_cents > 0),
    note            TEXT,
    created_by      BIGINT       REFERENCES users(id),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payments_transaction_id
    ON payments (transaction_id);

COMMIT;
