BEGIN;

-- Restore NOT NULL ip for rows that still have one; USB rows without IP
-- must be deleted or filled before this rollback can succeed.
UPDATE printers
SET ip_address = COALESCE(NULLIF(ip_address, ''), '0.0.0.0')
WHERE ip_address IS NULL OR ip_address = '';

ALTER TABLE printers
    ALTER COLUMN ip_address SET NOT NULL;

ALTER TABLE printers
    DROP CONSTRAINT IF EXISTS printers_connection_type_check;

ALTER TABLE printers
    DROP COLUMN IF EXISTS connection_type;

COMMIT;
