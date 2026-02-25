# Role-Based Access Control (RBAC)

The system has **4 roles**. The `admin` role always bypasses all role checks.

## Roles

| Role       | Description   |
|------------|---------------|
| admin      | Administrator — full access to everything |
| manager    | Manager — broad access, manages workers, payroll, finance |
| operator   | Operator — handles products, categories, suppliers, stock |
| cashier    | Cashier — read access + customer operations |

---we

## Permissions by Module

### Auth (`/auth`)

| Endpoint | Method | admin | manager | operator | cashier | Public |
|----------|--------|-------|---------|----------|---------|--------|
| `/auth/login` | POST | - | - | - | - | yes |
| `/auth/register` | POST | - | - | - | - | yes |
| `/auth/refresh` | POST | - | - | - | - | yes |
| `/auth/users` | GET | yes | no | no | no | no |
| `/auth/users` | POST | yes | no | no | no | no |
| `/auth/users/:id` | GET | yes | no | no | no | no |
| `/auth/users/:id` | PATCH | yes | no | no | no | no |
| `/auth/users/:id` | DELETE | yes | no | no | no | no |
| `/auth/users/:id/block` | POST | yes | no | no | no | no |
| `/auth/users/:id/unblock` | POST | yes | no | no | no | no |
| `/auth/users/:id/password` | POST | authenticated (admin or self) | authenticated (admin or self) | authenticated (admin or self) | authenticated (admin or self) | no |

### Categories (`/api/categories`)

| Endpoint | Method | admin | manager | operator | cashier |
|----------|--------|-------|---------|----------|---------|
| `/api/categories` | GET | yes | yes | yes | yes |
| `/api/categories/:id` | GET | yes | yes | yes | yes |
| `/api/categories` | POST | yes | no | yes | no |
| `/api/categories/:id` | PATCH | yes | no | yes | no |
| `/api/categories/:id` | DELETE | yes | no | yes | no |

### Products (`/api/products`)

| Endpoint | Method | admin | manager | operator | cashier |
|----------|--------|-------|---------|----------|---------|
| `/api/products` | GET | yes | yes | yes | yes |
| `/api/products/:id` | GET | yes | yes | yes | yes |
| `/api/products` | POST | yes | no | yes | no |
| `/api/products/:id` | PATCH | yes | no | yes | no |
| `/api/products/:id` | DELETE | yes | no | yes | no |

### Suppliers (`/api/suppliers`)

| Endpoint | Method | admin | manager | operator | cashier |
|----------|--------|-------|---------|----------|---------|
| `/api/suppliers` | GET | yes | no | yes | no |
| `/api/suppliers/:id` | GET | yes | no | yes | no |
| `/api/suppliers` | POST | yes | no | yes | no |
| `/api/suppliers/:id` | PATCH | yes | no | yes | no |
| `/api/suppliers/:id` | DELETE | yes | no | yes | no |

### Customers (`/api/customers`)

| Endpoint | Method | admin | manager | operator | cashier |
|----------|--------|-------|---------|----------|---------|
| `/api/customers` | GET | yes | yes | yes | yes |
| `/api/customers/:id` | GET | yes | yes | yes | yes |
| `/api/customers` | POST | yes | yes | no | yes |
| `/api/customers/:id/spent` | POST | yes | yes | no | yes |
| `/api/customers/:id/contact` | PATCH | yes | yes | yes | yes |
| `/api/customers/:id/admin` | PATCH | yes | yes | no | no |
| `/api/customers/:id` | DELETE | yes | no | no | no |

### Workers (`/api/workers`)

| Endpoint | Method | admin | manager | operator | cashier |
|----------|--------|-------|---------|----------|---------|
| `/api/workers` | GET | yes | yes | yes | no |
| `/api/workers/:id` | GET | yes | yes | yes | no |
| `/api/workers` | POST | yes | yes | no | no |
| `/api/workers/:id` | PATCH | yes | yes | no | no |
| `/api/workers/:id` | DELETE | yes | no | no | no |

### Worker Finance (`/api/workers` — compensation, fines, debts)

| Endpoint | Method | admin | manager | operator | cashier |
|----------|--------|-------|---------|----------|---------|
| `/api/workers/compensation` | POST | yes | yes | no | no |
| `/api/workers/:id/compensation` | GET | yes | yes | no | no |
| `/api/workers/:id/fines` | POST | yes | yes | no | no |
| `/api/workers/:id/fines` | GET | yes | yes | no | no |
| `/api/workers/:id/debts` | POST | yes | yes | no | no |
| `/api/workers/:id/debts` | GET | yes | yes | no | no |

### Payroll (`/api/payroll`)

| Endpoint | Method | admin | manager | operator | cashier |
|----------|--------|-------|---------|----------|---------|
| `/api/payroll` | GET | yes | yes | no | no |
| `/api/payroll/calculate` | POST | yes | yes | no | no |
| `/api/payroll/:id/pay` | POST | yes | yes | no | no |

### Warehouse (`/api/warehouses`)

| Endpoint | Method | admin | manager | operator | cashier |
|----------|--------|-------|---------|----------|---------|
| `/api/warehouses` | GET | yes | yes | yes | yes |
| `/api/warehouses/:id` | GET | yes | yes | yes | yes |
| `/api/warehouses` | POST | yes | yes | no | no |

### Stock (`/api/stock`)

| Endpoint | Method | admin | manager | operator | cashier |
|----------|--------|-------|---------|----------|---------|
| `/api/stock/in` | POST | yes | yes | yes | no |
| `/api/stock/out` | POST | yes | yes | yes | no |
| `/api/stock/transfer` | POST | yes | yes | yes | no |
| `/api/stock/move` | POST | yes | yes | yes | no |
| `/api/stock/balance` | GET | yes | yes | yes | yes |
| `/api/stock/details` | GET | yes | yes | yes | yes |

### Finance (`/api`)

| Endpoint | Method | admin | manager | operator | cashier |
|----------|--------|-------|---------|----------|---------|
| `/api/payment-types` | GET | yes | yes | yes | yes |
| `/api/transactions` | GET | yes | yes | yes | yes |
| `/api/transactions/:id` | GET | yes | yes | yes | yes |
| `/api/transactions/manual` | POST | yes | yes | yes | no |
| `/api/transactions/:id/payments` | POST | yes | yes | yes | no |
| `/api/transactions/:id/cancel` | POST | yes | yes | no | no |

### Notifications — SMS Logs (`/api/notifications/sms`)

| Endpoint | Method | admin | manager | operator | cashier |
|----------|--------|-------|---------|----------|---------|
| `/api/notifications/sms` | GET | yes | no | no | no |
| `/api/notifications/sms/:job_id` | GET | yes | no | no | no |
| `/api/notifications/sms/:job_id/requeue` | POST | yes | no | no | no |

---

## Summary by Role

### Admin
Full access to all endpoints. Bypasses all role checks.

### Manager
- Workers: full CRUD (except delete)
- Worker Finance: compensation, fines, debts
- Payroll: list, calculate, pay
- Warehouse: create, read
- Stock: all operations
- Finance: all operations including cancel
- Customers: full CRUD (except delete), including admin updates
- Categories: read only
- Products: read only

### Operator
- Categories: full CRUD
- Products: full CRUD
- Suppliers: full CRUD
- Stock: in, out, transfer, move, balance, details
- Finance: read, create manual, add payment
- Customers: read, contact update
- Workers: read only
- Warehouse: read only

### Cashier
- Categories: read only
- Products: read only
- Customers: read, create, add spent, contact update
- Stock: balance, details (read only)
- Finance: read only (transactions, payment types)
- Warehouse: read only
