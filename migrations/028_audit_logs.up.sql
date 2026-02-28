CREATE TABLE IF NOT EXISTS audit_logs (
    id          BIGSERIAL    PRIMARY KEY,
    user_id     BIGINT       NOT NULL,
    username    TEXT         NOT NULL DEFAULT '',
    action      TEXT         NOT NULL,
    entity_type TEXT         NOT NULL,
    entity_id   TEXT,
    method      TEXT         NOT NULL,
    path        TEXT         NOT NULL,
    ip_address  TEXT,
    request_id  TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id     ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity_type ON audit_logs(entity_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at  ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action      ON audit_logs(action);
