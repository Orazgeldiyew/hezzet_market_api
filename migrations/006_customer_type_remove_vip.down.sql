-- 006_customer_type_remove_vip.down.sql
-- Re-add 'vip' to customer_type enum.

BEGIN;

-- Drop the column default (it references the current enum and blocks ALTER TYPE)
ALTER TABLE customers ALTER COLUMN type DROP DEFAULT;

CREATE TYPE customer_type_old AS ENUM ('regular', 'vip', 'wholesale');

ALTER TABLE customers
  ALTER COLUMN type TYPE customer_type_old
  USING (type::text::customer_type_old);

DROP TYPE customer_type;
ALTER TYPE customer_type_old RENAME TO customer_type;

-- Restore the default
ALTER TABLE customers ALTER COLUMN type SET DEFAULT 'regular';

COMMIT;
