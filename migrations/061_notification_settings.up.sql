BEGIN;

-- Singleton row (id=1) for runtime-configurable notification settings.
-- Lets admins change e.g. reorder digest hour without redeploy.
CREATE TABLE IF NOT EXISTS notification_settings (
    id                  SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    reorder_digest_hour INT      NOT NULL DEFAULT 9
                        CHECK (reorder_digest_hour BETWEEN 0 AND 23),
    reorder_digest_enabled BOOLEAN NOT NULL DEFAULT true,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO notification_settings (id) VALUES (1)
ON CONFLICT (id) DO NOTHING;

COMMIT;
