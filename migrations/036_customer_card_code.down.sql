DROP INDEX IF EXISTS idx_customers_card_code;
ALTER TABLE customers DROP COLUMN IF EXISTS card_code;
