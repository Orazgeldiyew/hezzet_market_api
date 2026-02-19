DO $$ BEGIN
  CREATE TYPE unit_type AS ENUM ('piece','kg','liter','meter','box');
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

ALTER TABLE products
  ADD COLUMN IF NOT EXISTS unit_type unit_type NOT NULL DEFAULT 'piece',
  ADD COLUMN IF NOT EXISTS unit_scale integer NOT NULL DEFAULT 1000;

DO $$ BEGIN
  ALTER TABLE products
    ADD CONSTRAINT products_unit_scale_chk CHECK (unit_scale > 0);
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_products_unit_type ON products(unit_type);
