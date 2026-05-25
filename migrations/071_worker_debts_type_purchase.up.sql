BEGIN;

-- The sale module records "worker credit purchases" (товар взял работник в
-- долг по карте) as worker_debts rows with type='purchase'. The original
-- migration 021 only allowed 'advance' (аванс) and 'loan' (ссуда), so every
-- credit sale to a worker was failing the CHECK constraint — sale rolled back
-- mid-transaction. Adding 'purchase' restores the flow and keeps the three
-- types distinguishable in payroll reports.
ALTER TABLE worker_debts
    DROP CONSTRAINT IF EXISTS worker_debts_type_check;

ALTER TABLE worker_debts
    ADD CONSTRAINT worker_debts_type_check
    CHECK (type IN ('advance', 'loan', 'purchase'));

COMMIT;
