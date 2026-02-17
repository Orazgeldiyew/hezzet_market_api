-- 006_customer_type_remove_vip.down.sql
-- Re-add 'vip' to customer_type enum.

BEGIN;

CREATE TYPE customer_type_old AS ENUM ('regular', 'vip', 'wholesale');

ALTER TABLE customers
  ALTER COLUMN type TYPE customer_type_old
  USING (type::text::customer_type_old);

DROP TYPE customer_type;
ALTER TYPE customer_type_old RENAME TO customer_type;

COMMIT;
