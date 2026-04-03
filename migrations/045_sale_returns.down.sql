BEGIN;

DROP TABLE IF EXISTS sale_return_items;
DROP TABLE IF EXISTS sale_returns;

ALTER TABLE sales DROP CONSTRAINT IF EXISTS sales_status_check;
ALTER TABLE sales ADD CONSTRAINT sales_status_check
    CHECK (status IN ('draft','confirmed','cancelled'));

ALTER TABLE role_permissions DROP CONSTRAINT IF EXISTS role_permissions_action_check;
ALTER TABLE role_permissions ADD CONSTRAINT role_permissions_action_check
    CHECK (action IN ('view','create','update','delete','transfer'));

COMMIT;
