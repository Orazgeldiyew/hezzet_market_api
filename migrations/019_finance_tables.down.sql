DROP TRIGGER IF EXISTS transactions_set_updated_at ON transactions;
DROP FUNCTION IF EXISTS trg_transactions_updated_at();
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS payment_types;
