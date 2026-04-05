BEGIN;

CREATE TABLE supplier_debts (
    id              BIGSERIAL PRIMARY KEY,
    supplier_id     BIGINT NOT NULL REFERENCES suppliers(id),
    purchase_id     BIGINT REFERENCES purchase_orders(id),
    amount_cents    BIGINT NOT NULL CHECK (amount_cents > 0),
    remaining_cents BIGINT NOT NULL CHECK (remaining_cents >= 0),
    status          TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'settled', 'cancelled')),
    note            TEXT,
    created_by      BIGINT REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_supplier_debts_supplier ON supplier_debts(supplier_id);
CREATE INDEX idx_supplier_debts_status ON supplier_debts(status);

CREATE TABLE supplier_debt_payments (
    id              BIGSERIAL PRIMARY KEY,
    debt_id         BIGINT NOT NULL REFERENCES supplier_debts(id),
    amount_cents    BIGINT NOT NULL CHECK (amount_cents > 0),
    payment_type_id BIGINT REFERENCES payment_types(id),
    note            TEXT,
    created_by      BIGINT REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_supplier_debt_payments_debt ON supplier_debt_payments(debt_id);

COMMIT;
