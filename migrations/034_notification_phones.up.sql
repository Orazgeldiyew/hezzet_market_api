CREATE TABLE notification_phones (
    id         BIGSERIAL PRIMARY KEY,
    phone      TEXT NOT NULL UNIQUE,
    label      TEXT NOT NULL DEFAULT '',
    enabled    BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
