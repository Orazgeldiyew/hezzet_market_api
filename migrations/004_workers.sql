-- 004_workers.sql
CREATE TABLE IF NOT EXISTS workers (
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    position      VARCHAR(100),
    department    VARCHAR(100),
    phone         VARCHAR(50),
    email         VARCHAR(255),
    address       TEXT,
    salary        NUMERIC NOT NULL DEFAULT 0,
    hire_date     DATE,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    notes         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_workers_name ON workers(name);
CREATE INDEX IF NOT EXISTS idx_workers_is_active ON workers(is_active);
CREATE INDEX IF NOT EXISTS idx_workers_department ON workers(department);
CREATE INDEX IF NOT EXISTS idx_workers_deleted_at ON workers(deleted_at);
