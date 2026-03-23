# Permission List

> `admin` has access to **everything** automatically (bypass in middleware).
> No need to list admin in each row — it always passes.

---

## Roles

| Role       | Description              |
|------------|--------------------------|
| `admin`    | Full access to all APIs  |
| `manager`  | Store management         |
| `operator` | Stock & purchase ops     |
| `cashier`  | Sales & customers        |

---

## Permission Table

### Auth / Users

| Endpoint                          | Method | cashier | operator | manager | admin |
|-----------------------------------|--------|---------|----------|---------|-------|
| `/auth/login`                     | POST   | -       | -        | -       | -     |
| `/auth/register`                  | POST   | -       | -        | manager | auto  |
| `/auth/users`                     | GET    | -       | -        | manager | auto  |
| `/auth/users/:id`                 | GET    | -       | -        | manager | auto  |
| `/auth/users/:id/block`           | POST   | -       | -        | manager | auto  |
| `/auth/users/:id/unblock`         | POST   | -       | -        | manager | auto  |
| `/auth/users/:id/password`        | POST   | -       | -        | manager | auto  |

### Products

| Endpoint                                    | Method | cashier | operator | manager | admin |
|---------------------------------------------|--------|---------|----------|---------|-------|
| `/api/products`                             | GET    | yes     | yes      | yes     | auto  |
| `/api/products/:id`                         | GET    | yes     | yes      | yes     | auto  |
| `/api/products`                             | POST   | -       | yes      | -       | auto  |
| `/api/products/:id`                         | PUT    | -       | yes      | -       | auto  |
| `/api/products/:id`                         | DELETE | -       | yes      | -       | auto  |
| `/api/products/:id/categories`              | PUT    | -       | yes      | -       | auto  |
| `/api/products/:id/categories/:categoryId`  | DELETE | -       | yes      | -       | auto  |

### Categories

| Endpoint                | Method | cashier | operator | manager | admin |
|-------------------------|--------|---------|----------|---------|-------|
| `/api/categories`       | GET    | yes     | yes      | yes     | auto  |
| `/api/categories/:id`   | GET    | yes     | yes      | yes     | auto  |
| `/api/categories`       | POST   | -       | yes      | -       | auto  |
| `/api/categories/:id`   | PUT    | -       | yes      | -       | auto  |
| `/api/categories/:id`   | DELETE | -       | yes      | -       | auto  |

### Sales

| Endpoint                          | Method | cashier | operator | manager | admin |
|-----------------------------------|--------|---------|----------|---------|-------|
| `/api/sales`                      | GET    | yes     | yes      | yes     | auto  |
| `/api/sales/:id`                  | GET    | yes     | yes      | yes     | auto  |
| `/api/sales`                      | POST   | yes     | yes      | yes     | auto  |
| `/api/sales/:id/items`            | POST   | yes     | yes      | yes     | auto  |
| `/api/sales/:id/items/:itemId`    | DELETE | yes     | yes      | yes     | auto  |
| `/api/sales/:id/confirm`          | POST   | yes     | yes      | yes     | auto  |
| `/api/sales/:id/cancel`           | POST   | yes     | yes      | yes     | auto  |

### Customers

| Endpoint                          | Method | cashier | operator | manager | admin |
|-----------------------------------|--------|---------|----------|---------|-------|
| `/api/customers`                  | GET    | yes     | yes      | yes     | auto  |
| `/api/customers/:id`              | GET    | yes     | yes      | yes     | auto  |
| `/api/customers/by-card/:code`    | GET    | yes     | yes      | yes     | auto  |
| `/api/customers`                  | POST   | yes     | -        | yes     | auto  |
| `/api/customers/:id/spent`        | POST   | yes     | -        | yes     | auto  |
| `/api/customers/:id/contact`      | PATCH  | yes     | yes      | yes     | auto  |
| `/api/customers/:id/admin`        | PATCH  | -       | -        | yes     | auto  |
| `/api/customers/:id`              | DELETE | -       | -        | -       | auto  |

### Stock

| Endpoint                          | Method | cashier | operator | manager | admin |
|-----------------------------------|--------|---------|----------|---------|-------|
| `/api/stock/in`                   | POST   | -       | yes      | yes     | auto  |
| `/api/stock/in/bulk`              | POST   | -       | yes      | yes     | auto  |
| `/api/stock/out`                  | POST   | -       | yes      | yes     | auto  |
| `/api/stock/transfer`             | POST   | -       | yes      | yes     | auto  |
| `/api/stock/move`                 | POST   | -       | yes      | yes     | auto  |
| `/api/stock/warehouse/:id`        | GET    | yes     | yes      | yes     | auto  |
| `/api/stock/product/:id`          | GET    | yes     | yes      | yes     | auto  |
| `/api/stock/opening-balance`      | POST   | -       | yes      | yes     | auto  |

### Purchases

| Endpoint                          | Method | cashier | operator | manager | admin |
|-----------------------------------|--------|---------|----------|---------|-------|
| `/api/purchases`                  | GET    | -       | yes      | yes     | auto  |
| `/api/purchases/:id`              | GET    | -       | yes      | yes     | auto  |
| `/api/purchases`                  | POST   | -       | yes      | yes     | auto  |
| `/api/purchases/:id`              | PUT    | -       | yes      | yes     | auto  |
| `/api/purchases/:id/receive`      | POST   | -       | yes      | yes     | auto  |
| `/api/purchases/:id/cancel`       | POST   | -       | yes      | yes     | auto  |
| `/api/purchases/:id/payments`     | POST   | -       | yes      | yes     | auto  |

### Suppliers

| Endpoint                | Method | cashier | operator | manager | admin |
|-------------------------|--------|---------|----------|---------|-------|
| `/api/suppliers`        | GET    | -       | yes      | -       | auto  |
| `/api/suppliers/:id`    | GET    | -       | yes      | -       | auto  |
| `/api/suppliers`        | POST   | -       | yes      | -       | auto  |
| `/api/suppliers/:id`    | PUT    | -       | yes      | -       | auto  |
| `/api/suppliers/:id`    | DELETE | -       | yes      | -       | auto  |

### Warehouses

| Endpoint                | Method | cashier | operator | manager | admin |
|-------------------------|--------|---------|----------|---------|-------|
| `/api/warehouses`       | GET    | yes     | yes      | yes     | auto  |
| `/api/warehouses/:id`   | GET    | yes     | yes      | yes     | auto  |
| `/api/warehouses`       | POST   | -       | -        | yes     | auto  |
| `/api/warehouses/:id`   | PUT    | -       | -        | yes     | auto  |
| `/api/warehouses/:id`   | DELETE | -       | -        | yes     | auto  |

### Finance / Transactions

| Endpoint                              | Method | cashier | operator | manager | admin |
|---------------------------------------|--------|---------|----------|---------|-------|
| `/api/payment-types`                  | GET    | yes     | yes      | yes     | auto  |
| `/api/transactions`                   | GET    | -       | -        | yes     | auto  |
| `/api/transactions/:id`              | GET    | -       | -        | yes     | auto  |
| `/api/transactions/manual`           | POST   | -       | yes      | yes     | auto  |
| `/api/transactions/:id/payments`     | POST   | -       | yes      | yes     | auto  |
| `/api/transactions/:id/cancel`       | POST   | -       | -        | yes     | auto  |

### Workers

| Endpoint                          | Method | cashier | operator | manager | admin |
|-----------------------------------|--------|---------|----------|---------|-------|
| `/api/workers`                    | GET    | -       | yes      | yes     | auto  |
| `/api/workers/:id`                | GET    | -       | yes      | yes     | auto  |
| `/api/workers`                    | POST   | -       | -        | yes     | auto  |
| `/api/workers/:id`                | PUT    | -       | -        | yes     | auto  |
| `/api/workers/:id/toggle`         | PATCH  | -       | -        | -       | auto  |

### Worker Finance (Fines, Debts, Compensation)

| Endpoint                          | Method | cashier | operator | manager | admin |
|-----------------------------------|--------|---------|----------|---------|-------|
| `/api/worker-finance/...`         | ALL    | -       | -        | yes     | auto  |

### Worker Cards

| Endpoint                          | Method | cashier | operator | manager | admin |
|-----------------------------------|--------|---------|----------|---------|-------|
| `/api/worker-cards/by-card/:code` | GET    | yes     | yes      | yes     | auto  |
| `/api/worker-cards`               | GET    | -       | -        | yes     | auto  |
| `/api/worker-cards`               | POST   | -       | -        | yes     | auto  |
| `/api/worker-cards/:id`           | PUT    | -       | -        | yes     | auto  |
| `/api/worker-cards/:id`           | DELETE | -       | -        | yes     | auto  |

### Payroll

| Endpoint                          | Method | cashier | operator | manager | admin |
|-----------------------------------|--------|---------|----------|---------|-------|
| `/api/payroll/...`                | ALL    | -       | -        | yes     | auto  |

### Reports

| Endpoint                          | Method | cashier | operator | manager | admin |
|-----------------------------------|--------|---------|----------|---------|-------|
| `/api/reports/...`                | ALL    | -       | -        | yes     | auto  |

### Receipt Settings

| Endpoint                          | Method | cashier | operator | manager | admin |
|-----------------------------------|--------|---------|----------|---------|-------|
| `/api/receipt-settings`           | GET    | -       | -        | yes     | auto  |
| `/api/receipt-settings`           | PUT    | -       | -        | yes     | auto  |

### Audit Logs

| Endpoint                          | Method | cashier | operator | manager | admin |
|-----------------------------------|--------|---------|----------|---------|-------|
| `/api/audit-logs`                 | GET    | -       | -        | yes     | auto  |

### Notifications (SMS)

| Endpoint                          | Method | cashier | operator | manager | admin |
|-----------------------------------|--------|---------|----------|---------|-------|
| `/api/notifications/...`          | ALL    | -       | -        | -       | auto  |

### Permissions

| Endpoint                          | Method | cashier | operator | manager | admin |
|-----------------------------------|--------|---------|----------|---------|-------|
| `/api/permissions`                | GET    | -       | -        | yes     | auto  |
| `/api/permissions/:role/:module`  | PUT    | -       | -        | yes     | auto  |

---

## Dynamic Permissions (role_permissions table)

Manager/admin can enable or disable module access per role at runtime:

```
PUT /api/permissions/:role/:module
{ "enabled": false }
```

Example — disable sales module for cashier:
```
PUT /api/permissions/cashier/sales
{ "enabled": false }
```

This overrides the static role check above. If no row exists in `role_permissions`, access is **allowed by default**.

### API

```
GET /api/permissions           — list all permission overrides
PUT /api/permissions/:role/:module  — set enabled true/false
```

### Caching

Permissions are cached in Redis for 5 minutes (`perm:v1:<role>:<module>`).
Cache is invalidated on update.
