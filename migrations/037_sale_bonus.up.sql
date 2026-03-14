ALTER TABLE sales ADD COLUMN bonus_used_cents BIGINT NOT NULL DEFAULT 0 CHECK (bonus_used_cents >= 0);
