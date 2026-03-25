# Roles & Permissions — Frontend Guide

---

## Step 1: Create Role

```
POST /api/roles
Authorization: Bearer <admin_token>
```
```json
{
  "code": "senior_cashier",
  "name": "Senior Cashier",
  "description": "Cashier who can also view reports"
}
```
Response: `{ "data": { "id": 5, "code": "senior_cashier", "name": "Senior Cashier" } }`

---

## Step 2: Set Permissions for Role

After creating role, send ALL permissions at once:

```
PUT /api/permissions/5/bulk
Authorization: Bearer <admin_token>
```
```json
{
  "permissions": [
    { "module": "sales",       "action": "view",   "granted": true  },
    { "module": "sales",       "action": "create", "granted": true  },
    { "module": "sales",       "action": "update", "granted": true  },
    { "module": "sales",       "action": "delete", "granted": false },
    { "module": "products",    "action": "view",   "granted": true  },
    { "module": "products",    "action": "create", "granted": false },
    { "module": "products",    "action": "update", "granted": false },
    { "module": "products",    "action": "delete", "granted": false },
    { "module": "categories",  "action": "view",   "granted": true  },
    { "module": "categories",  "action": "create", "granted": false },
    { "module": "categories",  "action": "update", "granted": false },
    { "module": "categories",  "action": "delete", "granted": false },
    { "module": "customers",   "action": "view",   "granted": true  },
    { "module": "customers",   "action": "create", "granted": true  },
    { "module": "customers",   "action": "update", "granted": false },
    { "module": "customers",   "action": "delete", "granted": false },
    { "module": "stock",       "action": "view",   "granted": false },
    { "module": "stock",       "action": "create", "granted": false },
    { "module": "stock",       "action": "update", "granted": false },
    { "module": "stock",       "action": "delete", "granted": false },
    { "module": "purchases",   "action": "view",   "granted": false },
    { "module": "purchases",   "action": "create", "granted": false },
    { "module": "purchases",   "action": "update", "granted": false },
    { "module": "purchases",   "action": "delete", "granted": false },
    { "module": "suppliers",   "action": "view",   "granted": false },
    { "module": "suppliers",   "action": "create", "granted": false },
    { "module": "suppliers",   "action": "update", "granted": false },
    { "module": "suppliers",   "action": "delete", "granted": false },
    { "module": "warehouses",  "action": "view",   "granted": true  },
    { "module": "warehouses",  "action": "create", "granted": false },
    { "module": "warehouses",  "action": "update", "granted": false },
    { "module": "warehouses",  "action": "delete", "granted": false },
    { "module": "finance",     "action": "view",   "granted": false },
    { "module": "finance",     "action": "create", "granted": false },
    { "module": "finance",     "action": "update", "granted": false },
    { "module": "finance",     "action": "delete", "granted": false },
    { "module": "reports",     "action": "view",   "granted": true  },
    { "module": "reports",     "action": "create", "granted": false },
    { "module": "reports",     "action": "update", "granted": false },
    { "module": "reports",     "action": "delete", "granted": false },
    { "module": "workers",     "action": "view",   "granted": false },
    { "module": "workers",     "action": "create", "granted": false },
    { "module": "workers",     "action": "update", "granted": false },
    { "module": "workers",     "action": "delete", "granted": false },
    { "module": "payroll",     "action": "view",   "granted": false },
    { "module": "payroll",     "action": "create", "granted": false },
    { "module": "payroll",     "action": "update", "granted": false },
    { "module": "payroll",     "action": "delete", "granted": false }
  ]
}
```

Where `5` = role id from Step 1 response.

---

## Step 3: Assign Role to User (multiselect)

When creating user — send `roles` as array (multiselect):
```
POST /auth/register
Authorization: Bearer <admin_token>
```
```json
{
  "username": "mergen",
  "password": "123456",
  "full_name": "Mergen Atayev",
  "phone": "+99361234567",
  "email": "mergen@mail.com",
  "roles": ["senior_cashier"]
}
```

Multiple roles:
```json
{
  "username": "bayram",
  "password": "123456",
  "full_name": "Bayram Orazov",
  "roles": ["senior_cashier", "operator"]
}
```

Update existing user roles:
```
PATCH /auth/users/:id
Authorization: Bearer <admin_token>
```
```json
{
  "roles": ["senior_cashier", "operator"]
}
```

After role change user must re-login.

---
вввввввв
## Step 4: Login — Check User Roles

```
POST /auth/login
```
```json
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

## Step 5: Show/Hide UI Based on Permissions

After login, load permissions for the user's roles:

```
GET /api/permissions/matrix
Authorization: Bearer <token>
```

Response:
```json
[
  { "role_id": 5, "role_code": "senior_cashier", "module": "sales",   "view": true,  "create": true,  "update": true,  "delete": false },
  { "role_id": 5, "role_code": "senior_cashier", "module": "reports", "view": true,  "create": false, "update": false, "delete": false },
  { "role_id": 5, "role_code": "senior_cashier", "module": "stock",   "view": false, "create": false, "update": false, "delete": false }
]
```

Frontend logic:
```javascript
const { roles } = loginResponse.data;

// Admin = show everything
if (roles.includes('admin')) {
  showAllUI();
  return;
}

// Load permission matrix
const matrix = await api.get('/api/permissions/matrix');

// Check permission
function canDo(module, action) {
  const row = matrix.find(m => roles.includes(m.role_code) && m.module === module);
  if (!row) return true; // no row = allowed by default
  return row[action];
}

// Sidebar: show/hide menu items
if (canDo('sales', 'view'))     showMenuItem('Sales');
if (canDo('products', 'view'))  showMenuItem('Products');
if (canDo('stock', 'view'))     showMenuItem('Stock');
if (canDo('reports', 'view'))   showMenuItem('Reports');
if (canDo('customers', 'view')) showMenuItem('Customers');

// Buttons: show/hide based on action
if (canDo('sales', 'create'))   showButton('New Sale');
if (canDo('sales', 'delete'))   showButton('Delete Sale');
if (canDo('products', 'create')) showButton('Add Product');
```

---

## Full Permission List (all modules x all actions)

This is the complete list of modules and actions available:

| Module         | view | create | update | delete | Affects endpoints |
|----------------|------|--------|--------|--------|-------------------|
| `sales`        | CRUD | CRUD   | CRUD   | CRUD   | `/api/sales/*` |
| `products`     | CRUD | CRUD   | CRUD   | CRUD   | `/api/products/*` |
| `categories`   | CRUD | CRUD   | CRUD   | CRUD   | `/api/categories/*` |
| `customers`    | CRUD | CRUD   | CRUD   | CRUD   | `/api/customers/*` |
| `stock`        | CRUD | CRUD   | CRUD   | CRUD   | `/api/stock/*` |
| `purchases`    | CRUD | CRUD   | CRUD   | CRUD   | `/api/purchases/*` |
| `suppliers`    | CRUD | CRUD   | CRUD   | CRUD   | `/api/suppliers/*` |
| `warehouses`   | CRUD | CRUD   | CRUD   | CRUD   | `/api/warehouses/*` |
| `finance`      | CRUD | CRUD   | CRUD   | CRUD   | `/api/transactions/*` |
| `reports`      | CRUD | CRUD   | CRUD   | CRUD   | `/api/reports/*` |
| `workers`      | CRUD | CRUD   | CRUD   | CRUD   | `/api/workers/*` |
| `worker-cards` | CRUD | CRUD   | CRUD   | CRUD   | `/api/worker-cards/*` |
| `payroll`      | CRUD | CRUD   | CRUD   | CRUD   | `/api/payroll/*` |

Total: **13 modules x 4 actions = 52 permissions per role**

---

## UI Mockups

### Create Role Page (admin only)

```
+----------------------------------------------------------+
|  Create New Role                                          |
+----------------------------------------------------------+
|  Code:        [senior_cashier    ]                        |
|  Name:        [Senior Cashier    ]                        |
|  Description: [Can view reports  ]                        |
|                                                           |
|  Permissions:                                             |
|  +------------+-------+--------+--------+--------+        |
|  | Module     | View  | Create | Update | Delete |        |
|  +------------+-------+--------+--------+--------+        |
|  | Sales      | [x]   | [x]    | [x]    | [ ]    |        |
|  | Products   | [x]   | [ ]    | [ ]    | [ ]    |        |
|  | Categories | [x]   | [ ]    | [ ]    | [ ]    |        |
|  | Customers  | [x]   | [x]    | [ ]    | [ ]    |        |
|  | Stock      | [ ]   | [ ]    | [ ]    | [ ]    |        |
|  | Purchases  | [ ]   | [ ]    | [ ]    | [ ]    |        |
|  | Suppliers  | [ ]   | [ ]    | [ ]    | [ ]    |        |
|  | Warehouses | [x]   | [ ]    | [ ]    | [ ]    |        |
|  | Finance    | [ ]   | [ ]    | [ ]    | [ ]    |        |
|  | Reports    | [x]   | [ ]    | [ ]    | [ ]    |        |
|  | Workers    | [ ]   | [ ]    | [ ]    | [ ]    |        |
|  | WorkerCards| [ ]   | [ ]    | [ ]    | [ ]    |        |
|  | Payroll    | [ ]   | [ ]    | [ ]    | [ ]    |        |
|  +------------+-------+--------+--------+--------+        |
|                                                           |
|                              [Cancel]  [Create Role]      |
+----------------------------------------------------------+
```

On "Create Role" click:
1. `POST /api/roles` with code, name, description
2. Get `role.id` from response
3. `PUT /api/permissions/{role.id}/bulk` with all checkboxes

### Assign Role to User (multiselect dropdown)

```
+----------------------------------------------------------+
|  Create User                                              |
+----------------------------------------------------------+
|  Username:  [mergen          ]                            |
|  Password:  [******          ]                            |
|  Full Name: [Mergen Atayev   ]                            |
|  Phone:     [+99361234567    ]                            |
|                                                           |
|  Roles:     [ senior_cashier      v ]  <-- multiselect    |
|             [x] Senior Cashier                            |
|             [ ] Operator                                  |
|             [ ] Cashier                                   |
|             [ ] Manager                                   |
|                                                           |
|                              [Cancel]  [Create User]      |
+----------------------------------------------------------+
```

Load role list for dropdown: `GET /api/roles`

---

## Summary Flow

```
ADMIN creates role  -->  POST /api/roles
         |
         v
ADMIN sets permissions  -->  PUT /api/permissions/{role_id}/bulk
         |
         v
ADMIN assigns role to user  -->  POST /auth/register  (roles: ["senior_cashier"])
         |                       or PATCH /auth/users/:id  (roles: ["senior_cashier"])
         v
USER logs in  -->  POST /auth/login  -->  response has "roles"
         |
         v
FRONTEND loads matrix  -->  GET /api/permissions/matrix
         |
         v
FRONTEND shows/hides UI  -->  canDo("sales", "view") ? show : hide
```

---

## Rules

- **admin** = full access always, no permission check needed
- **manager** permissions can only be changed by admin
- **System roles** (admin, manager, operator, cashier) cannot be deleted
- **Custom roles** can be deleted only if no users are assigned
- Frontend always sends **full permission table** on save (all 52 rows)
- After role change, user must **re-login** (old token becomes invalid)
