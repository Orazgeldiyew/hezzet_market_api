-- Phone and email must be unique (when not empty)
CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_phone_unique
    ON customers (phone) WHERE phone != '' AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_email_unique
    ON customers (email) WHERE email != '' AND deleted_at IS NULL;
