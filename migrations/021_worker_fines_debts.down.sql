DROP TRIGGER IF EXISTS worker_debts_set_updated_at ON worker_debts;
DROP FUNCTION IF EXISTS trg_worker_debts_updated_at();
DROP TABLE IF EXISTS worker_debts;

DROP TRIGGER IF EXISTS worker_fines_set_updated_at ON worker_fines;
DROP FUNCTION IF EXISTS trg_worker_fines_updated_at();
DROP TABLE IF EXISTS worker_fines;
