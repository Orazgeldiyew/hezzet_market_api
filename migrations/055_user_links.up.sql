BEGIN;

-- Link workers to users
ALTER TABLE workers ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_workers_user_id ON workers(user_id) WHERE user_id IS NOT NULL AND deleted_at IS NULL;

-- Link customers to users
ALTER TABLE customers ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_user_id ON customers(user_id) WHERE user_id IS NOT NULL AND deleted_at IS NULL;

-- Link suppliers to users
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_suppliers_user_id ON suppliers(user_id) WHERE user_id IS NOT NULL;

COMMIT;
