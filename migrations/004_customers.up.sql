-- 004_customers.up.sql
-- NOTE: 'vip' is included here because migration 006 removes it later.
-- Keep enum values in sync with that migration.

DO $$ BEGIN
  CREATE TYPE customer_type AS ENUM ('regular', 'vip', 'wholesale');
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS customers (
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    phone         VARCHAR(50),
    email         VARCHAR(255),
    type          customer_type NOT NULL DEFAULT 'regular',
    total_spent   NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (total_spent >= 0),
    bonus_points  NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (bonus_points >= 0),
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    notes         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_customers_deleted_at ON customers(deleted_at);
CREATE INDEX IF NOT EXISTS idx_customers_phone      ON customers(phone);
CREATE INDEX IF NOT EXISTS idx_customers_email      ON customers(email);
CREATE INDEX IF NOT EXISTS idx_customers_type       ON customers(type);
