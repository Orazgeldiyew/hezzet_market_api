-- Add 'discount' to valid permission actions
ALTER TABLE role_permissions DROP CONSTRAINT IF EXISTS role_permissions_action_check;
ALTER TABLE role_permissions ADD CONSTRAINT role_permissions_action_check
    CHECK (action IN ('view','create','update','delete','transfer','return','discount'));

-- sale_items: per-item discount
ALTER TABLE sale_items ADD COLUMN discount_percent INT NOT NULL DEFAULT 0
    CHECK (discount_percent >= 0 AND discount_percent <= 100);

-- sales: per-sale discount
ALTER TABLE sales ADD COLUMN discount_percent INT NOT NULL DEFAULT 0
    CHECK (discount_percent >= 0 AND discount_percent <= 100);
ALTER TABLE sales ADD COLUMN discount_cents BIGINT NOT NULL DEFAULT 0
    CHECK (discount_cents >= 0);
