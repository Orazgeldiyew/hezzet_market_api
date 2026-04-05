BEGIN;

CREATE TABLE cash_registers (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO cash_registers (name) VALUES ('Касса 1');

CREATE TABLE shifts (
    id               BIGSERIAL PRIMARY KEY,
    register_id      BIGINT NOT NULL REFERENCES cash_registers(id),
    user_id          BIGINT NOT NULL REFERENCES users(id),
    opened_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at        TIMESTAMPTZ,
    opening_cash     BIGINT NOT NULL DEFAULT 0,
    closing_cash     BIGINT,
    expected_cash    BIGINT,
    sales_count      INT NOT NULL DEFAULT 0,
    sales_total      BIGINT NOT NULL DEFAULT 0,
    returns_total    BIGINT NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT 'open'
                     CHECK (status IN ('open', 'closed')),
    note             TEXT,
    closed_by        BIGINT REFERENCES users(id)
);

CREATE INDEX idx_shifts_user_id ON shifts(user_id);
CREATE INDEX idx_shifts_register_id ON shifts(register_id);
CREATE INDEX idx_shifts_status ON shifts(status);
CREATE INDEX idx_shifts_opened_at ON shifts(opened_at DESC);

COMMIT;
