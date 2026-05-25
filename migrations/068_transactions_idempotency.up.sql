BEGIN;

-- Idempotency for manual transactions (POST /api/transactions/manual).
-- Prevents double-click and network-retry duplicates: the frontend supplies
-- a UUID generated when the form opens; the backend rejects a second insert
-- with the same key via the partial unique index.
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS idempotency_key TEXT;

-- Partial unique index: existing rows with NULL key (created before this
-- migration) don't need to satisfy uniqueness.
CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_idempotency_key
    ON transactions (idempotency_key)
    WHERE idempotency_key IS NOT NULL;

COMMIT;
