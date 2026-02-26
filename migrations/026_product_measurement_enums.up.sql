-- Drop indexes (IF EXISTS handles re-runs)
DROP INDEX IF EXISTS idx_products_unit_type;
DROP INDEX IF EXISTS idx_products_unit;

-- Drop defaults on both columns (they may hold references to old enum types)
ALTER TABLE products ALTER COLUMN unit_type DROP DEFAULT;
ALTER TABLE products ALTER COLUMN unit      DROP DEFAULT;

-- Convert both columns to TEXT so all old enum types can be dropped
ALTER TABLE products ALTER COLUMN unit_type TYPE text;
ALTER TABLE products ALTER COLUMN unit      TYPE text;

-- Drop old enum types (IF EXISTS handles re-runs and partial states)
DROP TYPE IF EXISTS unit_type;
DROP TYPE IF EXISTS unit_type_enum;
DROP TYPE IF EXISTS unit_enum;

-- Create new enum types
CREATE TYPE unit_type_enum AS ENUM ('piece', 'weight', 'volume');
CREATE TYPE unit_enum      AS ENUM ('piece', 'kg', 'g', 'l', 'ml');

-- Normalize existing data
UPDATE products SET unit_type = 'piece' WHERE unit_type NOT IN ('piece', 'weight', 'volume');
UPDATE products SET unit = 'kg'    WHERE LOWER(unit) IN ('kg', 'kilogram', 'kilograms');
UPDATE products SET unit = 'g'     WHERE LOWER(unit) IN ('g', 'gram', 'grams');
UPDATE products SET unit = 'l'     WHERE LOWER(unit) IN ('l', 'liter', 'litre', 'liters', 'litres');
UPDATE products SET unit = 'ml'    WHERE LOWER(unit) IN ('ml', 'milliliter', 'millilitre');
UPDATE products SET unit = 'piece' WHERE unit NOT IN ('piece', 'kg', 'g', 'l', 'ml');

-- Apply unit_type_enum to unit_type column
ALTER TABLE products
    ALTER COLUMN unit_type TYPE unit_type_enum USING unit_type::unit_type_enum,
    ALTER COLUMN unit_type SET DEFAULT 'piece'::unit_type_enum;

-- Apply unit_enum to unit column (separate statement to avoid cast conflicts)
ALTER TABLE products
    ALTER COLUMN unit TYPE unit_enum USING unit::unit_enum,
    ALTER COLUMN unit SET DEFAULT 'piece'::unit_enum;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_products_unit_type ON products(unit_type);
CREATE INDEX IF NOT EXISTS idx_products_unit      ON products(unit);
