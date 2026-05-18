BEGIN;

-- Stage C: capture before/after snapshots so reviewers can see WHAT changed,
-- not just who/when/where. JSONB stores arbitrary entity shapes; null on either
-- side means "the entity didn't exist" (create has old=NULL, delete has new=NULL).
ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS old_value JSONB,
    ADD COLUMN IF NOT EXISTS new_value JSONB;

-- No index — these are inspected on a per-row basis, not searched in bulk.
-- Adding GIN would bloat storage on a high-volume audit table.

COMMIT;
