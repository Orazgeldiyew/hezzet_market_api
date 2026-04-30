BEGIN;

-- Per-user read state for in-app notification inbox.
-- The source of truth for notifications remains sms_logs — this table just
-- records which logs each user has acknowledged in the UI, so the bell icon
-- can show a per-user unread badge.
CREATE TABLE IF NOT EXISTS notification_reads (
    user_id    BIGINT      NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    sms_log_id BIGINT      NOT NULL REFERENCES sms_logs(id) ON DELETE CASCADE,
    read_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, sms_log_id)
);

CREATE INDEX IF NOT EXISTS idx_notification_reads_user_read_at
    ON notification_reads (user_id, read_at DESC);

COMMIT;
