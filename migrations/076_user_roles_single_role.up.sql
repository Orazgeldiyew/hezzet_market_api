BEGIN;

-- Enforce one role per user at the schema level. Application code already
-- treats user_roles as single-valued (employees.Role is a *string), but the
-- table still allows multiple rows per user. A unique index makes the
-- invariant explicit so any future code path that tries to insert a second
-- row fails fast instead of silently producing a "two-role" record.

CREATE UNIQUE INDEX IF NOT EXISTS user_roles_user_id_unique
    ON user_roles(user_id);

COMMIT;
