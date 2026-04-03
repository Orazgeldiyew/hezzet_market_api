# Hezzet Market Backend — Complete API Reference

All endpoints require `Authorization: Bearer <access_token>` unless marked as Public.

---

## Auth

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/login` | Public | Login |
| POST | `/auth/register` | Public | Self-register |
| POST | `/auth/refresh` | Public | Refresh tokens |
| POST | `/auth/users` | manager | Create user |
| GET | `/auth/users` | manager | List users |
| GET | `/auth/users/:id` | manager | Get user |
| PATCH | `/auth/users/:id` | manager | Update user |
| DELETE | `/auth/users/:id` | manager | Delete user |
| POST | `/auth/users/:id/password` | admin/self | Change password |
| POST | `/auth/users/:id/block` | manager | Block user |
| POST | `/auth/users/:id/unblock` | manager | Unblock user |

### POST /auth/login
```json
{ "username": "admin", "password": "123456" }
```
Response includes `roles` and `permissions` arrays.

### POST /auth/register
```json
{
  "username": "mergen",
  "password": "123456",
  "full_name": "Mergen Atayev",
  "phone": "+99361234567",
  "email": "mergen@mail.com"
}
```

### POST /auth/users (create with roles)
```json
{
  "username": "mergen",
  "password": "123456",
  "full_name": "Mergen Atayev",
  "phone": "+99361234567",
  "email": "mergen@mail.com",
  "is_active": true,
  "roles": ["cashier"]
}
```

### PATCH /auth/users/:id
```json
{
  "username": "new_name",
  "full_name": "New Name",
  "roles": ["cashier", "operator"]
}
```

---

## Products

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/products` | cashier+ | List (search, pagination) |
| GET | `/api/products/:id` | cashier+ | Get by ID |
| GET | `/api/products/by-barcode/:code` | cashier+ | Find by barcode (scanner) |
| GET | `/api/products/:id/card` | cashier+ | Product card with stock info |
| GET | `/api/products/:id/categories` | cashier+ | Product categories |
| POST | `/api/products` | operator | Create (multipart/form-data) |
| PATCH | `/api/products/:id` | operator | Update |
| DELETE | `/api/products/:id` | operator | Delete |
| PUT | `/api/products/:id/categories` | operator | Set categories |
| DELETE | `/api/products/:id/categories/:categoryId` | operator | Remove category |
| POST | `/api/products/:id/photo` | operator | Upload photo |

### POST /api/products (multipart/form-data)
Field `data` (JSON string):
```json
{
  "name": "Red Bull 0.5L",
  "sku": "RB-05L",
  "barcodes": ["8690504012347"],
  "unit_type": "piece",
  "unit": "piece",
  "purchase_price": 12000,
  "sale_price": 18000,
  "is_active": true,
  "category_ids": [1, 3]
}
```
Field `file`: optional image (jpg/png/webp, max 5MB)

### GET /api/products?search=red+bull&limit=10&page=1
### GET /api/products/by-barcode/8690504012347

---

## Categories

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/categories` | cashier+ | List (paginated) |
| GET | `/api/categories/tree` | cashier+ | Tree structure |
| GET | `/api/categories/:id` | cashier+ | Get by ID |
| POST | `/api/categories` | operator | Create |
| PATCH | `/api/categories/:id` | operator | Update |
| DELETE | `/api/categories/:id` | operator | Delete |

### POST /api/categories
```json
{ "name": "Beverages", "parent_id": null, "is_active": true }
```

---

## Suppliers

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/suppliers` | operator | List |
| GET | `/api/suppliers/:id` | operator | Get |
| POST | `/api/suppliers` | operator | Create |
| PATCH | `/api/suppliers/:id` | operator | Update |
| DELETE | `/api/suppliers/:id` | operator | Delete |

### POST /api/suppliers
```json
{ "name": "Coca-Cola TM", "phone": "+99312345678", "email": "info@coca.tm", "address": "Ashgabat" }
```

---

## Warehouses

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/warehouses` | cashier+ | List |
| GET | `/api/warehouses/:id` | cashier+ | Get |
| POST | `/api/warehouses` | manager | Create |

### POST /api/warehouses
```json
{ "name": "Main Warehouse", "address": "Ashgabat, Oguzhan 15" }
```

---

## Stock

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/stock/in` | operator | Stock in (add) |
| POST | `/api/stock/in/bulk` | operator | Bulk stock in |
| POST | `/api/stock/out` | operator | Stock out (remove) |
| POST | `/api/stock/transfer` | operator | Transfer between warehouses |
| POST | `/api/stock/move` | operator | Generic movement (damaged, adjustment) |
| GET | `/api/stock/balance` | cashier+ | Current balances |
| GET | `/api/stock/details` | cashier+ | Movement ledger |
| GET | `/api/stock/negative` | operator | Negative stock items |

**IMPORTANT:** All POST stock endpoints require `idempotency_key` as valid UUID!

### POST /api/stock/in
```json
{
  "warehouse_id": 1,
  "product_id": 7,
  "qty_milli": 5000,
  "price_cents": 12000,
  "idempotency_key": "550e8400-e29b-41d4-a716-446655440000"
}
```
Frontend generates UUID: `crypto.randomUUID()`

### POST /api/stock/in/bulk
```json
{
  "warehouse_id": 1,
  "items": [
    { "product_id": 7, "qty_milli": 5000, "price_cents": 12000, "idempotency_key": "uuid-1" },
    { "product_id": 8, "qty_milli": 3000, "price_cents": 8000, "idempotency_key": "uuid-2" }
  ]
}
```

### POST /api/stock/out
```json
{
  "warehouse_id": 1,
  "product_id": 7,
  "qty_milli": 2000,
  "idempotency_key": "uuid-3"
}
```

### POST /api/stock/transfer
```json
{
  "from_warehouse_id": 1,
  "to_warehouse_id": 2,
  "product_id": 7,
  "qty_milli": 3000,
  "idempotency_key": "uuid-4"
}
```

### POST /api/stock/move
```json
{
  "warehouse_id": 1,
  "product_id": 7,
  "delta_milli": -1000,
  "type": "damaged",
  "idempotency_key": "uuid-5",
  "note": "broken packaging"
}
```

### GET /api/stock/balance?warehouse_id=1
### GET /api/stock/details?warehouse_id=1&product_id=7&page=1&limit=20

---

## Customers

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/customers` | cashier+ | List |
| GET | `/api/customers/:id` | cashier+ | Get |
| GET | `/api/customers/by-card/:code` | cashier+ | Find by card code |
| POST | `/api/customers` | cashier | Create |
| POST | `/api/customers/:id/spent` | cashier | Add spent (bonus calc) |
| PATCH | `/api/customers/:id/contact` | cashier+ | Update contact |
| PATCH | `/api/customers/:id/admin` | manager | Update type/active |
| DELETE | `/api/customers/:id` | admin | Delete |

### POST /api/customers
```json
{ "name": "Ahmed", "phone": "+99361111111", "type": "regular" }
```

### POST /api/customers/:id/spent
```json
{ "amount_cents": 50000 }
```

---

## Sales

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/sales` | cashier+ | Create draft |
| GET | `/api/sales` | cashier+ | List |
| GET | `/api/sales/:id` | cashier+ | Get with items |
| GET | `/api/sales/:id/receipt` | cashier+ | Get receipt |
| POST | `/api/sales/:id/confirm` | cashier+ | Confirm (deduct stock) |
| POST | `/api/sales/:id/cancel` | cashier+ | Cancel |
| POST | `/api/sales/:id/transfer` | perm: sales.transfer | Transfer draft |
| POST | `/api/sales/:id/return` | perm: sales.return | Return items |
| DELETE | `/api/sales/:id/items/:item_id` | cashier+ | Delete item from draft |

### POST /api/sales
```json
{
  "warehouse_id": 1,
  "customer_id": 5,
  "items": [
    { "product_id": 7, "qty_milli": 2000 },
    { "product_id": 8, "qty_milli": 1000 }
  ],
  "note": "test sale",
  "force": false
}
```

### POST /api/sales/:id/confirm
```json
{
  "payment_type_id": 1,
  "payment_amount": 50000,
  "bonus_used_cents": 0,
  "force": false
}
```

### POST /api/sales/:id/transfer
```json
{ "cashier_id": 5 }
```

### POST /api/sales/:id/return (partial)
```json
{
  "items": [
    { "sale_item_id": 10, "qty_milli": 1000 }
  ],
  "reason": "defective"
}
```

### POST /api/sales/:id/return (full — empty items)
```json
{ "reason": "customer changed mind" }
```

### DELETE /api/sales/:id/items/:item_id
```json
{ "delete_code": "0000" }
```

---

## Purchases

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/purchases` | operator | Create PO |
| GET | `/api/purchases` | operator | List |
| GET | `/api/purchases/:id` | operator | Get |
| GET | `/api/purchases/debt` | operator | Supplier debt summary |
| POST | `/api/purchases/:id/receive` | operator | Receive goods |
| POST | `/api/purchases/:id/cancel` | operator | Cancel PO |
| POST | `/api/purchases/:id/payments` | operator | Add payment |

### POST /api/purchases
```json
{
  "supplier_id": 1,
  "warehouse_id": 1,
  "items": [
    { "product_id": 7, "qty_milli": 10000, "unit_cost_cents": 12000 }
  ],
  "note": "weekly order"
}
```

### POST /api/purchases/:id/payments
```json
{ "payment_type_id": 1, "amount_cents": 120000, "note": "cash payment" }
```

---

## Finance

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/payment-types` | cashier+ | List payment types |
| GET | `/api/transactions` | manager | List transactions |
| GET | `/api/transactions/:id` | manager | Get transaction detail |
| POST | `/api/transactions/manual` | operator | Create manual transaction |
| POST | `/api/transactions/:id/payments` | operator | Add payment |
| POST | `/api/transactions/:id/cancel` | manager | Cancel transaction |

### POST /api/transactions/manual
```json
{
  "type": "expense",
  "amount_cents": 50000,
  "reason": "office supplies",
  "payment_type_id": 1,
  "payment_amount": 50000
}
```

---

## Workers

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/workers` | manager+ | List |
| GET | `/api/workers/:id` | manager+ | Get |
| POST | `/api/workers` | manager | Create |
| PATCH | `/api/workers/:id` | manager | Update |
| DELETE | `/api/workers/:id` | admin | Delete |
| POST | `/api/workers/compensation` | manager | Set salary |
| GET | `/api/workers/:id/compensation` | manager | Get salary |
| POST | `/api/workers/:id/fines` | manager | Add fine |
| GET | `/api/workers/:id/fines` | manager | List fines |
| POST | `/api/workers/:id/debts` | manager | Add debt |
| GET | `/api/workers/:id/debts` | manager | List debts |

### POST /api/workers
```json
{
  "name": "Bayram Atayev",
  "position": "cashier",
  "department": "sales",
  "phone": "+99362222222",
  "hire_date": "2026-01-15"
}
```

### POST /api/workers/compensation
```json
{ "worker_id": 1, "base_salary_cents": 500000, "pay_day": 25 }
```

### POST /api/workers/:id/fines
```json
{ "amount_cents": 10000, "reason": "late to work" }
```

### POST /api/workers/:id/debts
```json
{ "amount_cents": 50000, "type": "advance", "note": "salary advance" }
```

---

## Worker Cards

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/worker-cards` | manager | List |
| GET | `/api/worker-cards/by-card/:code` | cashier+ | Find by QR code |
| POST | `/api/worker-cards` | manager | Create |
| PUT | `/api/worker-cards/:id` | manager | Update |
| DELETE | `/api/worker-cards/:id` | manager | Delete |

### POST /api/worker-cards
```json
{ "worker_id": 1, "label": "Main card" }
```

---

## Payroll

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/payroll` | manager | List payroll runs |
| GET | `/api/payroll/export` | manager | Export Excel |
| POST | `/api/payroll/calculate` | manager | Calculate monthly payroll |
| POST | `/api/payroll/:id/pay` | manager | Pay employee |

### POST /api/payroll/calculate
```json
{ "worker_id": 1, "period": "2026-03" }
```

### POST /api/payroll/:id/pay
```json
{ "payment_type_code": "cash" }
```

---

## Roles & Permissions

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/roles` | manager | List roles |
| GET | `/api/roles/:id` | manager | Get role + permissions |
| POST | `/api/roles` | admin | Create role |
| PATCH | `/api/roles/:id` | admin | Update role |
| DELETE | `/api/roles/:id` | admin | Delete role |
| GET | `/api/permissions` | manager | List all permissions |
| GET | `/api/permissions/matrix` | manager | Permission matrix |
| PUT | `/api/permissions/:role_id/:module/:action` | manager | Update single permission |
| PUT | `/api/permissions/:role_id/bulk` | manager | Bulk update permissions |

### POST /api/roles
```json
{ "code": "senior_cashier", "name": "Senior Cashier", "description": "Can view reports" }
```

### PUT /api/permissions/2/sales/return
```json
{ "granted": false }
```

### PUT /api/permissions/2/bulk
```json
{
  "permissions": [
    { "module": "sales", "action": "view", "granted": true },
    { "module": "sales", "action": "create", "granted": true },
    { "module": "sales", "action": "return", "granted": false }
  ]
}
```

**Actions:** `view`, `create`, `update`, `delete`, `transfer`, `return`

---

## Favorites (Hot Keys)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/favorites` | any | List my favorites |
| POST | `/api/favorites` | any | Add favorite |
| DELETE | `/api/favorites/:product_id` | any | Remove favorite |
| PUT | `/api/favorites/reorder` | any | Reorder positions |

### POST /api/favorites
```json
{ "product_id": 7 }
```

### PUT /api/favorites/reorder
```json
{
  "items": [
    { "product_id": 7, "position": 0 },
    { "product_id": 8, "position": 1 }
  ]
}
```

---

## Reports

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/reports/dashboard` | manager | Dashboard stats |
| GET | `/api/reports/sales` | manager | Sales by period |
| GET | `/api/reports/sales/products` | manager | Sales by product |
| GET | `/api/reports/sales/export` | manager | Excel export |
| GET | `/api/reports/stock/export` | manager | Stock Excel export |

### GET /api/reports/sales?group_by=month&from=2026-01-01&to=2026-03-31
### GET /api/reports/sales/products?from=2026-01-01&to=2026-03-31&warehouse_id=1

---

## Receipt Settings

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/settings/receipt` | manager | Get settings |
| PUT | `/api/settings/receipt` | manager | Update settings |

### PUT /api/settings/receipt
```json
{
  "shop_name": "Hezzet Market",
  "shop_address": "Ashgabat",
  "shop_phone": "+993 12 345678",
  "footer": "Thank you!",
  "delete_code": "1234"
}
```

---

## Audit Logs

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/audit-logs` | manager | List logs |

### GET /api/audit-logs?user_id=1&entity_type=product&action=CREATE&from=2026-01-01T00:00:00Z&to=2026-03-31T23:59:59Z

---

## Notifications (admin only)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/notifications/sms` | admin | List SMS logs |
| GET | `/api/notifications/sms/:job_id` | admin | Get SMS log |
| POST | `/api/notifications/sms/:job_id/requeue` | admin | Retry SMS |
| GET | `/api/notifications/phones` | admin | List phones |
| POST | `/api/notifications/phones` | admin | Add phone |
| PUT | `/api/notifications/phones/:id` | admin | Update phone |
| DELETE | `/api/notifications/phones/:id` | admin | Delete phone |

### POST /api/notifications/phones
```json
{ "phone": "+99361234567", "label": "Manager phone" }
```

---

## Public Routes (no auth)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/receipt/:id?token=JWT` | Public receipt view |
| GET | `/swagger/*any` | API documentation |
| GET | `/uploads/*filepath` | Static files |

---

## Notes

- All money values in **cents** (int64): 50000 = 500.00 TMT
- All quantities in **milli-units** (int64): 1000 = 1 piece/kg, 2500 = 2.5 kg
- Pagination: `?page=1&limit=10`
- Dates: RFC3339 format `2026-03-14T00:00:00Z`
- Stock idempotency_key: must be valid UUID (`crypto.randomUUID()`)
- Admin role always bypasses all permission checks
