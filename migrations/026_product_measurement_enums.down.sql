DROP INDEX IF EXISTS idx_products_unit_type;
DROP INDEX IF EXISTS idx_products_unit;

-- Drop defaults first (they reference the enum types and block TYPE conversion)
ALTER TABLE products ALTER COLUMN unit_type DROP DEFAULT;
ALTER TABLE products ALTER COLUMN unit      DROP DEFAULT;

-- Convert back to TEXT
ALTER TABLE products
    ALTER COLUMN unit_type TYPE text,
    ALTER COLUMN unit      TYPE text;

DROP TYPE IF EXISTS unit_type_enum;
DROP TYPE IF EXISTS unit_enum;

-- Restore original unit_type enum
CREATE TYPE unit_type AS ENUM ('piece', 'kg', 'liter', 'meter', 'box');

ALTER TABLE products
    ALTER COLUMN unit_type TYPE unit_type USING unit_type::unit_type,
    ALTER COLUMN unit_type SET DEFAULT 'piece'::unit_type;

CREATE INDEX IF NOT EXISTS idx_products_unit_type ON products(unit_type);
