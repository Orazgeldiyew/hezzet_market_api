BEGIN;

-- 1. Add 'return' to valid permission actions
ALTER TABLE role_permissions DROP CONSTRAINT IF EXISTS role_permissions_action_check;
ALTER TABLE role_permissions ADD CONSTRAINT role_permissions_action_check
    CHECK (action IN ('view','create','update','delete','transfer','return'));

-- 2. Add new sale statuses for returns
ALTER TABLE sales DROP CONSTRAINT IF EXISTS sales_status_check;
ALTER TABLE sales ADD CONSTRAINT sales_status_check
    CHECK (status IN ('draft','confirmed','cancelled','returned','partially_returned'));

-- 3. Sale returns header
CREATE TABLE sale_returns (
    id          BIGSERIAL    PRIMARY KEY,
    sale_id     BIGINT       NOT NULL REFERENCES sales(id),
    reason      TEXT,
    total_cents BIGINT       NOT NULL DEFAULT 0,
    created_by  BIGINT       REFERENCES users(id),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_sale_returns_sale_id ON sale_returns(sale_id);

-- 4. Sale return line items
CREATE TABLE sale_return_items (
    id             BIGSERIAL    PRIMARY KEY,
    return_id      BIGINT       NOT NULL REFERENCES sale_returns(id) ON DELETE CASCADE,
    sale_item_id   BIGINT       NOT NULL REFERENCES sale_items(id),
    product_id     BIGINT       NOT NULL REFERENCES products(id),
    qty_milli      BIGINT       NOT NULL CHECK (qty_milli > 0),
    refund_cents   BIGINT       NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_sale_return_items_return_id ON sale_return_items(return_id);

COMMIT;
