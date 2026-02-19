-- add new enum value (Postgres allows this)
ALTER TYPE movement_type ADD VALUE IF NOT EXISTS 'opening_balance';
