-- 006_customer_type_remove_vip.up.sql
-- Remove 'vip' from customer_type enum safely.

BEGIN;

-- 1) Ensure there are no vip rows (convert them if any slipped in)
UPDATE customers
SET type = 'regular'
WHERE type = 'vip';

-- 2) Create new enum without vip
CREATE TYPE customer_type_new AS ENUM ('regular', 'wholesale');

-- 3) Alter column to new type using cast through text
ALTER TABLE customers
  ALTER COLUMN type TYPE customer_type_new
  USING (type::text::customer_type_new);

-- 4) Drop old enum and rename new enum to original name
DROP TYPE customer_type;
ALTER TYPE customer_type_new RENAME TO customer_type;

COMMIT;
