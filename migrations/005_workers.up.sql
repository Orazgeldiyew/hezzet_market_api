CREATE TABLE workers (
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    position      VARCHAR(100),
    department    VARCHAR(100),
    phone         VARCHAR(50),
    email         VARCHAR(255),
    address       TEXT,
    salary        NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (salary >= 0),
    hire_date     DATE,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    notes         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_workers_deleted_at ON workers(deleted_at);
CREATE INDEX idx_workers_is_active ON workers(is_active);
CREATE INDEX idx_workers_department ON workers(department);
