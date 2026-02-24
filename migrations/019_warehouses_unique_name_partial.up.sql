-- Fix: make unique index on warehouse name soft-delete aware.
-- The old index blocks re-creating a warehouse with a name that was soft-deleted.

DROP INDEX IF EXISTS warehouses_name_unique;

CREATE UNIQUE INDEX IF NOT EXISTS warehouses_name_unique
    ON warehouses(name) WHERE deleted_at IS NULL;
