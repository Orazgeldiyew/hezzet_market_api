BEGIN;

-- Unify users + workers into a single employees table.
--
-- The previous split made every staff member exist twice — once as a
-- "worker" (payroll/HR side) and once as a "user" (login side) — with a
-- nullable workers.user_id link tying them. Most fields (name, phone,
-- email, is_active) were duplicated.
--
-- Strategy: rename users → employees (preserves all FKs by OID), add the
-- HR columns from workers, drop the workers table (empty after the dev
-- wipe), and redirect the 6 worker FKs to point at employees.id. No data
-- migration is needed because the dev wipe left workers empty and the
-- only surviving user is the admin (id=1, password unchanged).
--
-- Two new flags split the formerly-implicit notion of "who am I":
--   has_account = true  → can log in (was every users row)
--   is_worker   = true  → market staff with payroll/fines/debts
-- An employee can be one, the other, both, or neither.

ALTER TABLE users RENAME TO employees;
ALTER SEQUENCE users_id_seq RENAME TO employees_id_seq;

-- Rename user-flavoured columns to the merged naming.
ALTER TABLE employees RENAME COLUMN full_name TO name;

-- Add HR columns from workers.
ALTER TABLE employees
    ADD COLUMN position    VARCHAR(100),
    ADD COLUMN department  VARCHAR(100),
    ADD COLUMN address     TEXT,
    ADD COLUMN salary      NUMERIC(14,2) NOT NULL DEFAULT 0,
    ADD COLUMN hire_date   DATE,
    ADD COLUMN notes       TEXT;

-- Flags that replace the implicit "user vs worker" distinction.
ALTER TABLE employees
    ADD COLUMN has_account BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN is_worker   BOOLEAN NOT NULL DEFAULT false;

-- Every surviving row in the merged table came from users, so all of
-- them can log in. Admin (id=1) is a system account, not market staff.
UPDATE employees SET has_account = true;

-- Workers don't carry login credentials. Allow username/password to be NULL,
-- but enforce them whenever has_account=true via a CHECK constraint so we
-- can never have a login-enabled row missing the fields auth needs.
ALTER TABLE employees ALTER COLUMN username DROP NOT NULL;
ALTER TABLE employees ALTER COLUMN password_hash DROP NOT NULL;
ALTER TABLE employees ADD CONSTRAINT employees_account_requires_credentials
    CHECK (
        (has_account = false) OR
        (has_account = true AND username IS NOT NULL AND password_hash IS NOT NULL)
    );

-- Make name NOT NULL — existing admin already has it, and any new
-- employee must have a display name.
UPDATE employees SET name = COALESCE(name, username) WHERE name IS NULL;
ALTER TABLE employees ALTER COLUMN name SET NOT NULL;

-- Redirect the FKs that pointed to workers(id) to now point at employees(id).
-- All six tables are empty on dev (and on prod), so the constraint swap is
-- a clean rebind with nothing to remap.
ALTER TABLE payroll_runs        DROP CONSTRAINT IF EXISTS payroll_runs_worker_id_fkey;
ALTER TABLE sales               DROP CONSTRAINT IF EXISTS sales_worker_id_fkey;
ALTER TABLE worker_cards        DROP CONSTRAINT IF EXISTS worker_cards_worker_id_fkey;
ALTER TABLE worker_compensation DROP CONSTRAINT IF EXISTS worker_compensation_worker_id_fkey;
ALTER TABLE worker_debts        DROP CONSTRAINT IF EXISTS worker_debts_worker_id_fkey;
ALTER TABLE worker_fines        DROP CONSTRAINT IF EXISTS worker_fines_worker_id_fkey;

-- workers is now an orphan — drop it.
DROP TABLE workers;

-- Re-add the FKs against employees.
ALTER TABLE payroll_runs
    ADD CONSTRAINT payroll_runs_worker_id_fkey
    FOREIGN KEY (worker_id) REFERENCES employees(id);
ALTER TABLE sales
    ADD CONSTRAINT sales_worker_id_fkey
    FOREIGN KEY (worker_id) REFERENCES employees(id);
ALTER TABLE worker_cards
    ADD CONSTRAINT worker_cards_worker_id_fkey
    FOREIGN KEY (worker_id) REFERENCES employees(id);
ALTER TABLE worker_compensation
    ADD CONSTRAINT worker_compensation_worker_id_fkey
    FOREIGN KEY (worker_id) REFERENCES employees(id);
ALTER TABLE worker_debts
    ADD CONSTRAINT worker_debts_worker_id_fkey
    FOREIGN KEY (worker_id) REFERENCES employees(id);
ALTER TABLE worker_fines
    ADD CONSTRAINT worker_fines_worker_id_fkey
    FOREIGN KEY (worker_id) REFERENCES employees(id);

COMMIT;
