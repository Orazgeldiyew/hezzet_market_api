BEGIN;

-- Hard guarantee: a user can have at most one open shift at a time.
-- The application-level COUNT(*) check has a race window; this partial
-- unique index closes it at the database level.
CREATE UNIQUE INDEX IF NOT EXISTS idx_shifts_user_open_unique
    ON shifts (user_id)
    WHERE status = 'open';

COMMIT;
