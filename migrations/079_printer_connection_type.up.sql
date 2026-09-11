BEGIN;

-- network = ESC/POS over TCP (ip:port)
-- usb     = local OS/browser print on the cashier PC
ALTER TABLE printers
    ADD COLUMN IF NOT EXISTS connection_type TEXT NOT NULL DEFAULT 'network';

ALTER TABLE printers
    DROP CONSTRAINT IF EXISTS printers_connection_type_check;

ALTER TABLE printers
    ADD CONSTRAINT printers_connection_type_check
    CHECK (connection_type IN ('network', 'usb'));

-- USB printers do not need an IP; keep existing network rows intact.
ALTER TABLE printers
    ALTER COLUMN ip_address DROP NOT NULL;

UPDATE printers
SET connection_type = 'network'
WHERE connection_type IS NULL OR connection_type = '';

COMMIT;
