BEGIN;

-- Add 'import' to the allowed permission actions so we can gate bulk-import
-- endpoints separately from per-row 'create'. A future feature may want
-- "view + create + update + import" granted to operators while reserving
-- the import action to admins/managers.
ALTER TABLE role_permissions
    DROP CONSTRAINT IF EXISTS role_permissions_action_check;
ALTER TABLE role_permissions
    ADD CONSTRAINT role_permissions_action_check
    CHECK (action = ANY (ARRAY['view','create','update','delete','transfer','return','discount','history','import']));

-- Seed default grants for the new action. Mirror create permission for now —
-- if you can create a product manually, by default you can also import a batch.
-- Admins always bypass via middleware, so this row is informational for them.
-- COALESCE handles roles that have NO products.create row at all (LEFT JOIN
-- would leave rp.granted = NULL → NOT NULL violation). Safe default: deny.
INSERT INTO role_permissions (role_id, module, action, granted)
SELECT r.id, 'products', 'import', COALESCE(rp.granted, false)
FROM roles r
LEFT JOIN role_permissions rp
       ON rp.role_id = r.id AND rp.module = 'products' AND rp.action = 'create'
ON CONFLICT (role_id, module, action) DO NOTHING;

COMMIT;
