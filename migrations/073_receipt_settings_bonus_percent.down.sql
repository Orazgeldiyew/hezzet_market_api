BEGIN;

ALTER TABLE receipt_settings DROP COLUMN IF EXISTS bonus_percent;

COMMIT;
