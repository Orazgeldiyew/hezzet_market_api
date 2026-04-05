BEGIN;

CREATE TABLE customer_debts (
    id              BIGSERIAL PRIMARY KEY,
    customer_id     BIGINT NOT NULL REFERENCES customers(id),
    sale_id         BIGINT REFERENCES sales(id),
    amount_cents    BIGINT NOT NULL CHECK (amount_cents > 0),
    remaining_cents BIGINT NOT NULL CHECK (remaining_cents >= 0),
    status          TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'settled', 'cancelled')),
    note            TEXT,
    created_by      BIGINT REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_customer_debts_customer ON customer_debts(customer_id);
CREATE INDEX idx_customer_debts_status ON customer_debts(status);
CREATE INDEX idx_customer_debts_sale ON customer_debts(sale_id);

-- Payments against customer debts
CREATE TABLE customer_debt_payments (
    id          BIGSERIAL PRIMARY KEY,
    debt_id     BIGINT NOT NULL REFERENCES customer_debts(id),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    payment_type_id BIGINT REFERENCES payment_types(id),
    note        TEXT,
    created_by  BIGINT REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_customer_debt_payments_debt ON customer_debt_payments(debt_id);

COMMIT;
