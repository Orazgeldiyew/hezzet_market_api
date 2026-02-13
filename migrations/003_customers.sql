-- 003_customers.sql
CREATE TABLE IF NOT EXISTS customers (
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    phone         VARCHAR(50),
    email         VARCHAR(255),
    type          VARCHAR(20) NOT NULL DEFAULT 'regular',
    total_spent   NUMERIC NOT NULL DEFAULT 0,
    bonus_points  NUMERIC NOT NULL DEFAULT 0,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    notes         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);
