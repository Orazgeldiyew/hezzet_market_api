-- Revert: remove 'bonus_payment' from transactions.type CHECK constraint
-- NOTE: will fail if any rows have type='bonus_payment'
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_type_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_type_check
    CHECK (type IN ('income', 'expense'));
