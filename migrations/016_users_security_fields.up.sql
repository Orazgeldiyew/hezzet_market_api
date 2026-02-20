ALTER TABLE users ADD COLUMN IF NOT EXISTS blocked_at        timestamptz NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS blocked_reason     text        NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_changed_at timestamptz NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version      integer     NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at      timestamptz NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS created_by         bigint      NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_by         bigint      NULL;

DO $$ BEGIN
    ALTER TABLE users ADD CONSTRAINT fk_users_created_by
        FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE users ADD CONSTRAINT fk_users_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL;
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;
