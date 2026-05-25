BEGIN;

-- Legal-document fields for purchase invoice (A4 printable form).
-- legal_name is the official entity name (e.g. "OOO Molokо-plyus"), distinct
-- from the friendlier display `name`. tax_id is the local registration/tax
-- identifier (Turkmenistan uses different IDs depending on entity type — kept
-- as TEXT to stay generic).

ALTER TABLE suppliers
    ADD COLUMN IF NOT EXISTS legal_name TEXT,
    ADD COLUMN IF NOT EXISTS tax_id     TEXT;

ALTER TABLE receipt_settings
    ADD COLUMN IF NOT EXISTS legal_name TEXT,
    ADD COLUMN IF NOT EXISTS tax_id     TEXT;

COMMIT;
