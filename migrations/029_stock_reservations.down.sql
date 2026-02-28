DROP TABLE IF EXISTS stock_reservations;
ALTER TABLE sales DROP COLUMN IF EXISTS status;
-- NOTE: Postgres cannot remove enum values; 'sale_return' stays in movement_type
