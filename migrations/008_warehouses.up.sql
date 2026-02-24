BEGIN;

CREATE TABLE IF NOT EXISTS warehouses (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    address    TEXT,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_warehouses_deleted_at ON warehouses(deleted_at);

CREATE UNIQUE INDEX IF NOT EXISTS warehouses_name_unique ON warehouses(name) WHERE deleted_at IS NULL;

COMMIT;
