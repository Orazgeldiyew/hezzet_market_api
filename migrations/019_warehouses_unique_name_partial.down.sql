-- Revert to the original non-partial unique index on warehouse name.

DROP INDEX IF EXISTS warehouses_name_unique;

CREATE UNIQUE INDEX IF NOT EXISTS warehouses_name_unique ON warehouses(name);
