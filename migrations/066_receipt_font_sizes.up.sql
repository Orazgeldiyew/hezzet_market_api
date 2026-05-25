BEGIN;

-- Configurable font sizes for the thermal receipt. Defaults match the
-- previously hard-coded CSS in modules/printer/html_render.go so existing
-- printers keep printing the same shape unless an admin tweaks them via
-- /receipt-settings.
ALTER TABLE receipt_settings
    ADD COLUMN IF NOT EXISTS font_size_header INT NOT NULL DEFAULT 32 CHECK (font_size_header BETWEEN 10 AND 80),
    ADD COLUMN IF NOT EXISTS font_size_items  INT NOT NULL DEFAULT 25 CHECK (font_size_items  BETWEEN 10 AND 80),
    ADD COLUMN IF NOT EXISTS font_size_meta   INT NOT NULL DEFAULT 23 CHECK (font_size_meta   BETWEEN 10 AND 80),
    ADD COLUMN IF NOT EXISTS font_size_total  INT NOT NULL DEFAULT 30 CHECK (font_size_total  BETWEEN 10 AND 80);

COMMIT;
