BEGIN;

CREATE TABLE printers (
    id           BIGSERIAL PRIMARY KEY,
    name         TEXT NOT NULL,
    ip_address   TEXT NOT NULL,
    port         INT NOT NULL DEFAULT 9100,
    register_id  BIGINT REFERENCES cash_registers(id),
    is_active    BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_printers_register_id ON printers(register_id);
CREATE INDEX idx_printers_is_active ON printers(is_active);

COMMIT;
