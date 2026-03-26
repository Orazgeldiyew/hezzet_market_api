# Transfer Permission -- Frontend Guide

## What is this?

The permission system controls what each role can do in each module. There are now **5 actions** per module:

| # | Action     | Description                                      |
|---|------------|--------------------------------------------------|
| 1 | view       | Can see the data (list, detail)                  |
| 2 | create     | Can create new records                           |
| 3 | update     | Can edit existing records                        |
| 4 | delete     | Can delete records                               |
| 5 | **transfer** | Can transfer a draft sale to another cashier   |

The **transfer** action is the 5th column added to the permission grid. A manager or admin can enable or disable it for each role.

Currently, **transfer** only applies to the `sales` module. For other modules, it has no effect but can still be toggled in the matrix.

---

## How It Works

1. An **admin** or **manager** opens the permission settings page.
2. They see a grid of roles vs. modules with checkboxes for each action.
3. They can toggle the **transfer** checkbox on or off for any role.
4. When a cashier logs in, the login response includes `"transfer": true/false` in their permission matrix. The frontend uses this to show or hide the Transfer button on the POS screen.

---

## API Endpoints

All permission endpoints require **manager** or **admin** role.

### 1. Get the Permission Matrix

Returns all roles, all modules, and all 5 action flags.

```
GET /api/permissions/matrix
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "role_id": 1,
      "role_code": "admin",
      "role_name": "Administrator",
      "module": "sales",
      "view": true,
      "create": true,
      "update": true,
      "delete": true,
      "transfer": true
    },
    {
      "role_id": 2,
      "role_code": "manager",
      "role_name": "Manager",
      "module": "sales",
      "view": true,
      "create": true,
      "update": true,
      "delete": true,
      "transfer": true
    },
    {
      "role_id": 3,
      "role_code": "cashier",
      "role_name": "Cashier",
      "module": "sales",
      "view": true,
      "create": true,
      "update": false,
      "delete": false,
      "transfer": false
    },
    {
      "role_id": 3,
      "role_code": "cashier",
      "role_name": "Cashier",
      "module": "products",
      "view": true,
      "create": false,
      "update": false,
      "delete": false,
      "transfer": false
    }
  ]
}
```

---

### 2. Update a Single Permission

Toggle one specific action for one role on one module.

```
PUT /api/permissions/:role_id/:module/:action
Content-Type: application/json
Authorization: Bearer <token>
```

**URL parameters:**

| Parameter | Type   | Description                                         |
|-----------|--------|-----------------------------------------------------|
| role_id   | number | The role to update                                  |
| module    | string | Module name (e.g., "sales", "products")             |
| action    | string | One of: "view", "create", "update", "delete", "transfer" |

**Request body:**
```json
{
  "granted": false
}
```

**Example -- disable transfer for cashier role (role_id = 3):**

```
PUT /api/permissions/3/sales/transfer
```
```json
{
  "granted": false
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "id": 42,
    "role_id": 3,
    "role_code": "cashier",
    "module": "sales",
    "action": "transfer",
    "granted": false,
    "updated_at": "2026-03-26T14:00:00Z"
  }
}
```

**Example -- enable transfer for cashier role:**

```
PUT /api/permissions/3/sales/transfer
```
```json
{
  "granted": true
}
```

---

### 3. Bulk Update Permissions

Update multiple permissions for one role at once. Useful for the "Save All" button on the settings page.

```
PUT /api/permissions/:role_id/bulk
Content-Type: application/json
Authorization: Bearer <token>
```

**Request body:**
```json
{
  "permissions": [
    { "module": "sales", "action": "view",     "granted": true },
    { "module": "sales", "action": "create",   "granted": true },
    { "module": "sales", "action": "update",   "granted": false },
    { "module": "sales", "action": "delete",   "granted": false },
    { "module": "sales", "action": "transfer", "granted": true }
  ]
}
```

**Response (200):**
```json
{
  "success": true,
  "data": [
    { "id": 38, "role_id": 3, "role_code": "cashier", "module": "sales", "action": "view",     "granted": true,  "updated_at": "2026-03-26T14:00:00Z" },
    { "id": 39, "role_id": 3, "role_code": "cashier", "module": "sales", "action": "create",   "granted": true,  "updated_at": "2026-03-26T14:00:00Z" },
    { "id": 40, "role_id": 3, "role_code": "cashier", "module": "sales", "action": "update",   "granted": false, "updated_at": "2026-03-26T14:00:00Z" },
    { "id": 41, "role_id": 3, "role_code": "cashier", "module": "sales", "action": "delete",   "granted": false, "updated_at": "2026-03-26T14:00:00Z" },
    { "id": 42, "role_id": 3, "role_code": "cashier", "module": "sales", "action": "transfer", "granted": true,  "updated_at": "2026-03-26T14:00:00Z" }
  ]
}
```

---

### Error Responses

**400 -- Invalid action name:**
```json
{
  "success": false,
  "error": "action must be view, create, update, or delete"
}
```

**403 -- Non-admin trying to change manager permissions:**
```json
{
  "success": false,
  "error": "only admin can change manager permissions"
}
```

**404 -- Role not found:**
```json
{
  "success": false,
  "error": "role not found"
}
```

---

## UI Mockup: Permission Grid (Settings Page)

The settings page shows a table with roles as rows, modules as groups, and 5 action columns with checkboxes.

```
+----------------------------------------------------------------------+
|  Settings > Permissions                                               |
+----------------------------------------------------------------------+
|                                                                        |
|  Role: [Cashier v]                                                     |
|                                                                        |
|  +------------------------------------------------------------------+ |
|  | Module      | View | Create | Update | Delete | Transfer         | |
|  |-------------|------|--------|--------|--------|------------------| |
|  | sales       | [x]  |  [x]   |  [ ]   |  [ ]   |  [x]            | |
|  | products    | [x]  |  [ ]   |  [ ]   |  [ ]   |  [ ]            | |
|  | customers   | [x]  |  [x]   |  [ ]   |  [ ]   |  [ ]            | |
|  | warehouses  | [x]  |  [ ]   |  [ ]   |  [ ]   |  [ ]            | |
|  | finance     | [ ]  |  [ ]   |  [ ]   |  [ ]   |  [ ]            | |
|  | reports     | [ ]  |  [ ]   |  [ ]   |  [ ]   |  [ ]            | |
|  +------------------------------------------------------------------+ |
|                                                                        |
|  [Save Changes]                                                        |
|                                                                        |
+----------------------------------------------------------------------+
```

### Key points for the grid:
- The **Transfer** column is the 5th and last column.
- Each checkbox makes a `PUT /api/permissions/:role_id/:module/:action` call on change, OR you can collect all changes and use the bulk endpoint on "Save Changes".
- The grid data comes from `GET /api/permissions/matrix`.
- Only managers and admins can see this page.
- Only admins can change the manager role's permissions.

---

## Login Response -- Permission Matrix

When a user logs in, the response includes their merged permissions. The frontend should store this and use it to show/hide UI elements.

The permissions are returned as a list of MatrixEntry objects:

```json
{
  "success": true,
  "data": {
    "token": "eyJhbG...",
    "user": {
      "id": 5,
      "name": "Merdan",
      "roles": ["cashier"]
    },
    "permissions": [
      {
        "module": "sales",
        "view": true,
        "create": true,
        "update": false,
        "delete": false,
        "transfer": true
      },
      {
        "module": "products",
        "view": true,
        "create": false,
        "update": false,
        "delete": false,
        "transfer": false
      }
    ]
  }
}
```

The `transfer` field is now included alongside the other 4 actions. Use it to decide whether to show the Transfer button on the POS screen.

---

## Frontend Implementation

### Rendering the Permission Grid

```javascript
const API_BASE = '/api';
const token = localStorage.getItem('token');

const headers = {
  'Authorization': `Bearer ${token}`,
  'Content-Type': 'application/json',
};

// Load the full permission matrix
async function loadMatrix() {
  const res = await fetch(`${API_BASE}/permissions/matrix`, { headers });
  const json = await res.json();
  return json.data; // array of MatrixEntry
}

// Toggle a single permission
async function togglePermission(roleId, module, action, granted) {
  const res = await fetch(`${API_BASE}/permissions/${roleId}/${module}/${action}`, {
    method: 'PUT',
    headers,
    body: JSON.stringify({ granted }),
  });
  const json = await res.json();
  if (!res.ok) {
    alert(json.error);
    return null;
  }
  return json.data;
}

// Save all permissions for a role at once
async function bulkSave(roleId, permissions) {
  // permissions = [{ module: "sales", action: "transfer", granted: true }, ...]
  const res = await fetch(`${API_BASE}/permissions/${roleId}/bulk`, {
    method: 'PUT',
    headers,
    body: JSON.stringify({ permissions }),
  });
  const json = await res.json();
  if (!res.ok) {
    alert(json.error);
    return null;
  }
  return json.data;
}

// Render the grid
function renderPermissionGrid(matrixEntries, selectedRoleId) {
  const filtered = matrixEntries.filter(e => e.role_id === selectedRoleId);
  const actions = ['view', 'create', 'update', 'delete', 'transfer'];

  const table = document.getElementById('permission-table');
  table.innerHTML = '';

  // Header row
  const headerRow = document.createElement('tr');
  headerRow.innerHTML = `
    <th>Module</th>
    ${actions.map(a => `<th>${a.charAt(0).toUpperCase() + a.slice(1)}</th>`).join('')}
  `;
  table.appendChild(headerRow);

  // Data rows
  filtered.forEach(entry => {
    const row = document.createElement('tr');
    row.innerHTML = `
      <td>${entry.module}</td>
      ${actions.map(action => `
        <td>
          <input type="checkbox"
            ${entry[action] ? 'checked' : ''}
            onchange="togglePermission(${entry.role_id}, '${entry.module}', '${action}', this.checked)"
          />
        </td>
      `).join('')}
    `;
    table.appendChild(row);
  });
}
```

### Checking Transfer Permission in POS

```javascript
// After login, store permissions from the response
const userPermissions = loginResponse.data.permissions;

// Helper function
function hasPermission(module, action) {
  const perm = userPermissions.find(p => p.module === module);
  return perm ? perm[action] === true : false;
}

// Show/hide the transfer button on the POS screen
if (hasPermission('sales', 'transfer')) {
  document.getElementById('transfer-btn').style.display = 'inline-block';
} else {
  document.getElementById('transfer-btn').style.display = 'none';
}
```

---

## Quick Reference

| Action                   | Method | URL                                          | Body                                                |
|--------------------------|--------|----------------------------------------------|-----------------------------------------------------|
| Get permission matrix    | GET    | /api/permissions/matrix                      | --                                                  |
| Toggle one permission    | PUT    | /api/permissions/:role_id/:module/:action    | `{ "granted": true }`                               |
| Bulk update for a role   | PUT    | /api/permissions/:role_id/bulk               | `{ "permissions": [{ "module": "...", "action": "...", "granted": true }] }` |

### Valid actions:
`view`, `create`, `update`, `delete`, `transfer`

### Access control:
| Who              | Can change           |
|------------------|----------------------|
| Admin            | All roles            |
| Manager          | All roles except manager |
| Cashier / others | No access            |
