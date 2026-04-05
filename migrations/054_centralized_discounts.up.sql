BEGIN;

-- A) Product-level discount (admin sets, auto-applies on all cashiers)
ALTER TABLE products ADD COLUMN discount_percent INT NOT NULL DEFAULT 0
    CHECK (discount_percent >= 0 AND discount_percent <= 100);

-- B) Amount threshold discounts (e.g. buy > 500 TMT → 5% off)
CREATE TABLE discount_rules (
    id              BIGSERIAL PRIMARY KEY,
    name            TEXT NOT NULL,
    min_amount_cents BIGINT NOT NULL CHECK (min_amount_cents > 0),
    discount_percent INT NOT NULL CHECK (discount_percent > 0 AND discount_percent <= 100),
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_by      BIGINT REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_discount_rules_active ON discount_rules(is_active, min_amount_cents);

COMMIT;
