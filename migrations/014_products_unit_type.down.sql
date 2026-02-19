ALTER TABLE products
  DROP CONSTRAINT IF EXISTS products_unit_scale_chk;

DROP INDEX IF EXISTS idx_products_unit_type;

ALTER TABLE products
  DROP COLUMN IF EXISTS unit_scale,
  DROP COLUMN IF EXISTS unit_type;

DO $$ BEGIN
  DROP TYPE unit_type;
EXCEPTION
  WHEN undefined_object THEN NULL;
END $$;
