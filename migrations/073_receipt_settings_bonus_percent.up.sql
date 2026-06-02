BEGIN;

-- Loyalty bonus percent (0–100) applied to sales for customers with cards.
-- The column exists on some local dev DBs from a hand-applied ALTER that
-- was never captured as a migration — production was missing it, which
-- broke GET /api/settings/receipt with "column bonus_percent does not exist".
-- IF NOT EXISTS makes this safe to re-run on machines where it's already
-- present.
ALTER TABLE receipt_settings
    ADD COLUMN IF NOT EXISTS bonus_percent INT NOT NULL DEFAULT 1
    CHECK (bonus_percent >= 0 AND bonus_percent <= 100);

COMMIT;
