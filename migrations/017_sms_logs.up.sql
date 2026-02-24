-- 017_sms_logs.up.sql
-- SMS audit/outbox log table for tracking SMS job lifecycle.

CREATE TABLE IF NOT EXISTS sms_logs (
    id           BIGSERIAL    PRIMARY KEY,
    job_id       TEXT         UNIQUE NOT NULL,
    type         TEXT         NOT NULL,
    to_phone     TEXT         NOT NULL,
    message      TEXT         NOT NULL,
    sms_from     TEXT         NOT NULL DEFAULT '',
    status       TEXT         NOT NULL DEFAULT 'queued'
                 CHECK (status IN ('queued','sending','sent','retrying','rate_limited','dlq','failed')),
    attempt      INT          NOT NULL DEFAULT 0,
    max_attempts INT          NOT NULL DEFAULT 0,
    dedup_key    TEXT,
    provider     TEXT         NOT NULL DEFAULT '',
    last_error   TEXT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    sent_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_sms_logs_status     ON sms_logs (status);
CREATE INDEX IF NOT EXISTS idx_sms_logs_to_phone   ON sms_logs (to_phone);
CREATE INDEX IF NOT EXISTS idx_sms_logs_created_at ON sms_logs (created_at);

-- Auto-update updated_at on every row modification.
CREATE OR REPLACE FUNCTION trg_sms_logs_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS sms_logs_set_updated_at ON sms_logs;
CREATE TRIGGER sms_logs_set_updated_at
    BEFORE UPDATE ON sms_logs
    FOR EACH ROW
    EXECUTE FUNCTION trg_sms_logs_updated_at();
