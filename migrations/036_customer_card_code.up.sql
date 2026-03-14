ALTER TABLE customers ADD COLUMN card_code TEXT;
UPDATE customers SET card_code = upper(substring(md5(random()::text || id::text), 1, 8)) WHERE card_code IS NULL;
ALTER TABLE customers ALTER COLUMN card_code SET NOT NULL;
CREATE UNIQUE INDEX idx_customers_card_code ON customers(card_code);
