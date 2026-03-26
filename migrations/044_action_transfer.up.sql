ALTER TABLE role_permissions DROP CONSTRAINT IF EXISTS role_permissions_action_check;
ALTER TABLE role_permissions ADD CONSTRAINT role_permissions_action_check
    CHECK (action IN ('view','create','update','delete','transfer'));
