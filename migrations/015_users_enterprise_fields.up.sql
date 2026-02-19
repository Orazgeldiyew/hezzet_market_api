-- 015_users_management.up.sql

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS blocked_at timestamptz,
  ADD COLUMN IF NOT EXISTS blocked_reason text,
  ADD COLUMN IF NOT EXISTS password_changed_at timestamptz,
  ADD COLUMN IF NOT EXISTS token_version integer NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS last_login_at timestamptz,
  ADD COLUMN IF NOT EXISTS created_by bigint REFERENCES users(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS updated_by bigint REFERENCES users(id) ON DELETE SET NULL;

-- indexes (часть у тебя уже есть; IF NOT EXISTS безопасно)
CREATE INDEX IF NOT EXISTS idx_users_blocked_at   ON users(blocked_at);
CREATE INDEX IF NOT EXISTS idx_users_username     ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_is_active    ON users(is_active);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at   ON users(deleted_at);
