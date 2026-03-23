# Dynamic RBAC — Roles & Permissions API

## Overview

Admin/manager can create custom roles, assign granular permissions (per module + action), then assign roles to users. Admin always has full access to everything.

---

## Concepts

### Roles
Each user has one or more roles. There are 4 **system roles** (cannot be deleted):
- `admin` — full access, bypasses all permission checks
- `manager` — manages store operations
- `operator` — handles stock/purchases
- `cashier` — handles sales

Admin/manager can create **custom roles** (e.g. `senior_cashier`, `stock_manager`).

### Permissions
Each role has permissions per **module** and **action**:

**Modules:** `sales`, `stock`, `products`, `customers`, `finance`, `reports`, `purchases`, `workers`

**Actions:** `view`, `create`, `update`, `delete`

Example: role `senior_cashier` can have:
- `sales.view` = true
- `sales.create` = true
- `sales.delete` = false
- `reports.view` = true

---

## API Endpoints

### Auth Header
All endpoints require: `Authorization: Bearer <access_token>`

---

### 1. List Roles

```
GET /api/roles
```

**Required role:** manager or admin

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "code": "admin",
      "name": "Administrator",
      "description": "",
      "is_system": true,
      "created_at": "2026-03-18T12:00:00Z",
      "updated_at": "2026-03-18T12:00:00Z"
    },
    {
      "id": 2,
      "code": "cashier",
      "name": "Cashier",
      "description": "",
      "is_system": true
    },
    {
      "id": 5,
      "code": "senior_cashier",
      "name": "Senior Cashier",
      "description": "Can view reports",
      "is_system": false
    }
  ]
}
```

---

### 2. Get Role with Permissions

```
GET /api/roles/:id
```

**Required role:** manager or admin

**Response:**
```json
{
  "success": true,
  "data": {
    "role": {
      "id": 5,
      "code": "senior_cashier",
      "name": "Senior Cashier",
      "description": "Can view reports",
      "is_system": false
    },
    "permissions": [
      { "id": 1, "role_id": 5, "role_code": "senior_cashier", "module": "sales", "action": "view", "granted": true },
      { "id": 2, "role_id": 5, "role_code": "senior_cashier", "module": "sales", "action": "create", "granted": true },
      { "id": 3, "role_id": 5, "role_code": "senior_cashier", "module": "sales", "action": "update", "granted": false },
      { "id": 4, "role_id": 5, "role_code": "senior_cashier", "module": "sales", "action": "delete", "granted": false },
      { "id": 5, "role_id": 5, "role_code": "senior_cashier", "module": "reports", "action": "view", "granted": true }
    ]
  }
}
```

---

### 3. Create Role

```
POST /api/roles
```

**Required role:** manager or admin

**Request body:**
```json
{
  "code": "senior_cashier",
  "name": "Senior Cashier",
  "description": "Cashier who can also view reports"
}
```

**Rules:**
- `code` — lowercase letters, digits, underscores only. 2-50 chars. Must be unique.
- `name` — display name, 1-100 chars.
- `description` — optional.

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": 5,
    "code": "senior_cashier",
    "name": "Senior Cashier",
    "description": "Cashier who can also view reports",
    "is_system": false
  }
}
```

**Errors:**
- `409 DUPLICATE_ROLE` — code already exists

---

### 4. Update Role

```
PATCH /api/roles/:id
```

**Required role:** manager or admin

**Request body (all fields optional):**
```json
{
  "name": "New Name",
  "description": "New description"
}
```

---

### 5. Delete Role

```
DELETE /api/roles/:id
```

**Required role:** admin only

**Rules:**
- Cannot delete system roles (admin, cashier, operator, manager)
- Cannot delete if users are assigned to this role

**Errors:**
- `403 FORBIDDEN` — "cannot delete system role"
- `409 ROLE_IN_USE` — "cannot delete role: users are assigned to it"

---

### 6. List All Permissions

```
GET /api/permissions
```

**Required role:** manager or admin

**Response:**
```json
{
  "success": true,
  "data": [
    { "id": 1, "role_id": 2, "role_code": "cashier", "module": "sales", "action": "view", "granted": true },
    { "id": 2, "role_id": 2, "role_code": "cashier", "module": "sales", "action": "create", "granted": true },
    { "id": 3, "role_id": 2, "role_code": "cashier", "module": "reports", "action": "view", "granted": false }
  ]
}
```

---

### 7. Permission Matrix

```
GET /api/permissions/matrix
```

**Required role:** manager or admin

Returns compact view — one row per role+module with all 4 actions:

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "role_id": 2,
      "role_code": "cashier",
      "role_name": "Cashier",
      "module": "sales",
      "view": true,
      "create": true,
      "update": true,
      "delete": true
    },
    {
      "role_id": 2,
      "role_code": "cashier",
      "role_name": "Cashier",
      "module": "reports",
      "view": false,
      "create": false,
      "update": false,
      "delete": false
    }
  ]
}
```

**UI tip:** Use this endpoint to build a permission table/grid:

```
              | sales                    | reports                  | stock
              | view create update delete| view create update delete| ...
--------------+--------------------------+--------------------------+----
cashier       |  ON    ON     ON    ON   |  OFF   OFF    OFF   OFF | ...
operator      |  ON    ON     ON    ON   |  OFF   OFF    OFF   OFF | ...
manager       |  ON    ON     ON    ON   |  ON    ON     ON    ON  | ...
senior_cashier|  ON    ON     OFF   OFF  |  ON    OFF    OFF   OFF | ...
```

---

### 8. Update Single Permission

```
PUT /api/permissions/:role_id/:module/:action
```

**Required role:** manager or admin (only admin can change manager permissions)

**URL params:**
- `role_id` — integer, role ID (from GET /api/roles)
- `module` — string: `sales`, `stock`, `products`, `customers`, `finance`, `reports`, `purchases`, `workers`
- `action` — string: `view`, `create`, `update`, `delete`

**Request body:**
```json
{
  "granted": true
}
```

**Example — disable cashier (role_id=2) from deleting sales:**
```
PUT /api/permissions/2/sales/delete
{ "granted": false }
```

**Example — enable cashier to view reports:**
```
PUT /api/permissions/2/reports/view
{ "granted": true }
```

---

### 9. Bulk Update Permissions

```
PUT /api/permissions/:role_id/bulk
```

**Required role:** manager or admin

**Request body:**
```json
{
  "permissions": [
    { "module": "sales", "action": "view", "granted": true },
    { "module": "sales", "action": "create", "granted": true },
    { "module": "sales", "action": "update", "granted": false },
    { "module": "sales", "action": "delete", "granted": false },
    { "module": "reports", "action": "view", "granted": true },
    { "module": "reports", "action": "create", "granted": false },
    { "module": "reports", "action": "update", "granted": false },
    { "module": "reports", "action": "delete", "granted": false }
  ]
}
```

**UI tip:** When admin saves the permission grid, collect all changed cells and send them in one bulk request.

---

## Frontend Implementation

### 1. Roles Management Page

```
+------------------------------------------------------+
|  Roles                                    [+ New Role]|
+------------------------------------------------------+
|  LOCK  Administrator (admin)         — system         |
|  LOCK  Manager (manager)             — system         |
|  LOCK  Operator (operator)           — system         |
|  LOCK  Cashier (cashier)             — system         |
|  EDIT  Senior Cashier (senior_cashier) — custom [DEL] |
|  EDIT  Stock Manager (stock_manager)   — custom [DEL] |
+------------------------------------------------------+
```

- System roles show LOCK — cannot delete, only edit permissions
- Custom roles show EDIT and DEL buttons
- Click role -> opens permission grid

### 2. Permission Grid (for selected role)

Load data: `GET /api/permissions/matrix` and filter by selected role_id.

```
+------------------------------------------------+
|  Permissions for: Senior Cashier               |
+------------+------+--------+--------+----------+
| Module     | View | Create | Update | Delete   |
+------------+------+--------+--------+----------+
| Sales      |  ON  |   ON   |   OFF  |   OFF   |
| Stock      |  ON  |   OFF  |   OFF  |   OFF   |
| Products   |  ON  |   OFF  |   OFF  |   OFF   |
| Customers  |  ON  |   ON   |   OFF  |   OFF   |
| Finance    |  OFF |   OFF  |   OFF  |   OFF   |
| Reports    |  ON  |   OFF  |   OFF  |   OFF   |
| Purchases  |  OFF |   OFF  |   OFF  |   OFF   |
| Workers    |  OFF |   OFF  |   OFF  |   OFF   |
+------------+------+--------+--------+----------+
|                              [Cancel] [Save]    |
+------------------------------------------------+
```

- Each cell is a toggle (checkbox/switch)
- On Save -> `PUT /api/permissions/:role_id/bulk` with all permissions

### 3. JavaScript Example

```javascript
// After login, get user roles from response
const { roles } = loginResponse.data;

// If admin — show everything
if (roles.includes('admin')) {
  showAllModules();
  return;
}

// Otherwise, load permission matrix
const { data: matrix } = await api.get('/api/permissions/matrix');

// Build permission map for current user's roles
const permMap = {};
matrix.forEach(entry => {
  if (roles.includes(entry.role_code)) {
    const key = entry.module;
    if (!permMap[key]) {
      permMap[key] = { view: false, create: false, update: false, delete: false };
    }
    // OR logic: if ANY role grants it, it's allowed
    if (entry.view)   permMap[key].view = true;
    if (entry.create) permMap[key].create = true;
    if (entry.update) permMap[key].update = true;
    if (entry.delete) permMap[key].delete = true;
  }
});

// Check permissions
function canDo(module, action) {
  if (!permMap[module]) return true; // no explicit row = allowed
  return permMap[module][action];
}

// Show/hide sidebar
if (canDo('sales', 'view'))   showMenuItem('Sales');
if (canDo('reports', 'view')) showMenuItem('Reports');

// Show/hide buttons
if (canDo('sales', 'create')) showButton('New Sale');
if (canDo('sales', 'delete')) showButton('Delete Sale');
```

### 4. Assign Role to User

When creating/editing user:
```
POST /auth/users
{
  "username": "mergen",
  "password": "123456",
  "roles": ["senior_cashier"]
}
```

Or update existing user roles:
```
PATCH /auth/users/:id
{
  "roles": ["senior_cashier", "operator"]
}
```

After role change, user must re-login (token_version is bumped automatically).

### 5. Login Response

```
POST /auth/login
{ "username": "mergen", "password": "123456" }
```

Response:
```json
{
  "data": {
    "access_token": "eyJ...",
    "user": { "id": 6, "username": "mergen" },
    "roles": ["senior_cashier"]
  }
}
```

---

## Permission Check Logic (Backend)

1. Backend reads `roles` from JWT token
2. If `admin` is in roles -> **allow everything**
3. Otherwise -> check `role_permissions` table for `(role, module, action)`
4. If **any** of the user's roles has `granted=true` -> allow
5. If no permission row exists -> **allow by default**
6. If all roles have `granted=false` -> **403 Forbidden**

---

## Important Notes

- **Admin bypass:** admin role always has full access. No permission rows needed for admin.
- **Default allow:** if no permission row exists for a role+module+action, access is **allowed**. To deny, explicitly set `granted: false`.
- **Cache:** permissions are cached in Redis for 5 minutes. Changes take effect immediately (cache invalidated on update).
- **Token version:** when user roles are changed via PATCH /auth/users/:id, the user's token is invalidated and they must re-login.
- **Manager permissions:** only admin can change permissions for the manager role.
- **System roles:** admin, manager, operator, cashier cannot be deleted. Only their permissions can be changed.
