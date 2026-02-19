-- 015_users_management.down.sql

DROP INDEX IF EXISTS idx_users_blocked_at;
DROP INDEX IF EXISTS idx_users_username;
-- idx_users_is_active и idx_users_deleted_at могут быть созданы ранее — можно не удалять,
-- но если хочешь откатывать "чисто", оставь:
-- DROP INDEX IF EXISTS idx_users_is_active;
-- DROP INDEX IF EXISTS idx_users_deleted_at;

ALTER TABLE users
  DROP COLUMN IF EXISTS updated_by,
  DROP COLUMN IF EXISTS created_by,
  DROP COLUMN IF EXISTS last_login_at,
  DROP COLUMN IF EXISTS token_version,
  DROP COLUMN IF EXISTS password_changed_at,
  DROP COLUMN IF EXISTS blocked_reason,
  DROP COLUMN IF EXISTS blocked_at;
