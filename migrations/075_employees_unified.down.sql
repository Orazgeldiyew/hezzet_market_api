BEGIN;

-- Recreate workers table (minimal — for rollback only).
CREATE TABLE workers (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    position   VARCHAR(100),
    department VARCHAR(100),
    phone      VARCHAR(50),
    email      VARCHAR(255),
    address    TEXT,
    salary     NUMERIC(14,2) NOT NULL DEFAULT 0,
    hire_date  DATE,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    notes      TEXT,
    user_id    BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- Rebind worker FKs back to workers (any data is lost on rollback).
ALTER TABLE payroll_runs        DROP CONSTRAINT IF EXISTS payroll_runs_worker_id_fkey;
ALTER TABLE sales               DROP CONSTRAINT IF EXISTS sales_worker_id_fkey;
ALTER TABLE worker_cards        DROP CONSTRAINT IF EXISTS worker_cards_worker_id_fkey;
ALTER TABLE worker_compensation DROP CONSTRAINT IF EXISTS worker_compensation_worker_id_fkey;
ALTER TABLE worker_debts        DROP CONSTRAINT IF EXISTS worker_debts_worker_id_fkey;
ALTER TABLE worker_fines        DROP CONSTRAINT IF EXISTS worker_fines_worker_id_fkey;

ALTER TABLE payroll_runs        ADD CONSTRAINT payroll_runs_worker_id_fkey        FOREIGN KEY (worker_id) REFERENCES workers(id);
ALTER TABLE sales               ADD CONSTRAINT sales_worker_id_fkey               FOREIGN KEY (worker_id) REFERENCES workers(id);
ALTER TABLE worker_cards        ADD CONSTRAINT worker_cards_worker_id_fkey        FOREIGN KEY (worker_id) REFERENCES workers(id);
ALTER TABLE worker_compensation ADD CONSTRAINT worker_compensation_worker_id_fkey FOREIGN KEY (worker_id) REFERENCES workers(id);
ALTER TABLE worker_debts        ADD CONSTRAINT worker_debts_worker_id_fkey        FOREIGN KEY (worker_id) REFERENCES workers(id);
ALTER TABLE worker_fines        ADD CONSTRAINT worker_fines_worker_id_fkey        FOREIGN KEY (worker_id) REFERENCES workers(id);

ALTER TABLE employees
    DROP COLUMN position,
    DROP COLUMN department,
    DROP COLUMN address,
    DROP COLUMN salary,
    DROP COLUMN hire_date,
    DROP COLUMN notes,
    DROP COLUMN has_account,
    DROP COLUMN is_worker;

ALTER TABLE employees RENAME COLUMN name TO full_name;
ALTER TABLE employees RENAME TO users;
ALTER SEQUENCE employees_id_seq RENAME TO users_id_seq;

COMMIT;
