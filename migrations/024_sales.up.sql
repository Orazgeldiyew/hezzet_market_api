-- 024 up: Add 'sale' to movement_type enum
ALTER TYPE movement_type ADD VALUE IF NOT EXISTS 'sale';
