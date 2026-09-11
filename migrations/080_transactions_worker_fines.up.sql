-- Allow finance transactions linked to worker fines (CreateFine writes related_table='worker_fines').
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_related_table_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_related_table_check
    CHECK (related_table IN (
        'sale', 'purchase', 'manual', 'adjustment',
        'payroll', 'worker_debt', 'worker_fines'
    ));
