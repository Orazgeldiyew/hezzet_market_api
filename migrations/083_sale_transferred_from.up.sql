-- Mark drafts that were handed from one cashier to another («Переданные мне»).
ALTER TABLE sales
    ADD COLUMN IF NOT EXISTS transferred_from BIGINT REFERENCES employees(id);

CREATE INDEX IF NOT EXISTS idx_sales_transferred_from
    ON sales (created_by, transferred_from)
    WHERE status = 'draft' AND transferred_from IS NOT NULL;
