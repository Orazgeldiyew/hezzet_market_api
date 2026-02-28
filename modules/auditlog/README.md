# Audit Log Module

Tracks every successful write operation made by authenticated users and exposes a paginated query API for managers.

---

## Table of Contents

- [Overview](#overview)
- [Database](#database)
- [Module Structure](#module-structure)
- [Middleware](#middleware)
  - [Action Resolution](#action-resolution)
  - [Entity Type Resolution](#entity-type-resolution)
- [API](#api)
- [Integration](#integration)

---

## Overview

The audit log module automatically records who did what and when across all protected API endpoints. It works as a Gin middleware that runs **after** each request completes — if the response is a successful write (POST / PATCH / PUT / DELETE returning 2xx), a row is inserted into `audit_logs` in a background goroutine. This means audit logging never blocks or fails the main request.

Read access is restricted to the `manager` role via `GET /api/audit-logs`.

---

## Database

Migration: `028_audit_logs.up.sql`

```sql
CREATE TABLE audit_logs (
    id          BIGSERIAL    PRIMARY KEY,
    user_id     BIGINT       NOT NULL,
    username    TEXT         NOT NULL DEFAULT '',
    action      TEXT         NOT NULL,   -- CREATE | UPDATE | DELETE | PAY | STOCK_IN | …
    entity_type TEXT         NOT NULL,   -- product | worker | sale | stock | …
    entity_id   TEXT,                    -- value of :id param, if any
    method      TEXT         NOT NULL,   -- HTTP method
    path        TEXT         NOT NULL,   -- actual request path (with real IDs)
    ip_address  TEXT,
    request_id  TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
```

Indexes on `user_id`, `entity_type`, `action`, and `created_at DESC` support all common filter/sort queries efficiently.

---

## Module Structure

```
modules/auditlog/
├── model.go        — AuditLog struct, AuditFilter, AuditListResult
├── repository.go   — Create (fire-and-forget) + List (paginated, filtered)
├── middleware.go   — AuditMiddleware gin.HandlerFunc + action/entity helpers
├── handler.go      — HTTP handler for GET /api/audit-logs
└── routes.go       — Route registration
```

---

## Middleware

`AuditMiddleware(repo *Repository) gin.HandlerFunc` is registered on the `/api` group **before** all route registrations so it applies to every protected endpoint.

```
POST/PATCH/PUT/DELETE + 2xx + authenticated user
        ↓
  resolve action      (from actionOverrides map or HTTP method default)
  resolve entity_type (from route pattern first segment)
  resolve entity_id   (from :id param)
        ↓
  go repo.Create(...)   ← non-blocking
```

Requests that do not match all three conditions (write method, 2xx status, authenticated user) are silently skipped with no DB write.

### Action Resolution

Actions are determined by looking up `"METHOD /full/gin/route/pattern"` in the `actionOverrides` map first. If not found, a method-based default is used:

| HTTP Method | Default Action |
|-------------|----------------|
| POST        | `CREATE`       |
| PATCH / PUT | `UPDATE`       |
| DELETE      | `DELETE`       |

**Override examples:**

| Route                                  | Action            |
|----------------------------------------|-------------------|
| `POST /api/payroll/calculate`          | `CALCULATE`       |
| `POST /api/payroll/:id/pay`            | `PAY`             |
| `POST /api/transactions/:id/cancel`    | `CANCEL`          |
| `POST /api/transactions/:id/payments`  | `PAYMENT_ADD`     |
| `POST /api/stock/in`                   | `STOCK_IN`        |
| `POST /api/stock/in/bulk`              | `STOCK_IN_BULK`   |
| `POST /api/stock/out`                  | `STOCK_OUT`       |
| `POST /api/stock/transfer`             | `STOCK_TRANSFER`  |
| `POST /api/stock/move`                 | `STOCK_MOVE`      |
| `POST /api/workers/:id/fines`          | `FINE_CREATE`     |
| `POST /api/workers/:id/debts`          | `DEBT_CREATE`     |
| `POST /api/workers/compensation`       | `COMPENSATION_SET`|
| `POST /auth/users/:id/block`           | `USER_BLOCK`      |
| `POST /auth/users/:id/unblock`         | `USER_UNBLOCK`    |
| `POST /auth/users/:id/password`        | `PASSWORD_CHANGE` |
| `POST /api/customers/:id/spent`        | `CUSTOMER_SPENT`  |
| `PUT /api/products/:id/categories`     | `CATEGORY_SET`    |
| `DELETE /api/products/:id/categories/:categoryId` | `CATEGORY_REMOVE` |

To add a new override, append to `actionOverrides` in `middleware.go`.

### Entity Type Resolution

The entity type is derived from the Gin route pattern (not the real URL):

1. Strip `/api/` or `/auth/` prefix.
2. Take the first path segment.
3. Apply singular exceptions map (`categories` → `category`).
4. Otherwise, strip a trailing `s` if the segment is longer than 3 characters.

Examples:

| Route pattern             | Entity type |
|---------------------------|-------------|
| `/api/products/:id`       | `product`   |
| `/api/workers/:id/fines`  | `worker`    |
| `/api/stock/in`           | `stock`     |
| `/auth/users/:id`         | `user`      |
| `/api/categories/:id`     | `category`  |

---

## API

### `GET /api/audit-logs`

**Auth:** Bearer JWT — `manager` role required.

**Query parameters:**

| Parameter     | Type    | Description                              |
|---------------|---------|------------------------------------------|
| `user_id`     | integer | Filter by the user who performed the action |
| `entity_type` | string  | Filter by entity type (`product`, `worker`, `sale`, …) |
| `action`      | string  | Filter by action (`CREATE`, `PAY`, `STOCK_IN`, …) |
| `from`        | string  | Start of date range (RFC3339, e.g. `2025-01-01T00:00:00Z`) |
| `to`          | string  | End of date range (RFC3339) |
| `page`        | integer | Page number (default: 1) |
| `limit`       | integer | Items per page (default: 20, max: 100) |

**Response `200`:**

```json
{
  "data": {
    "items": [
      {
        "id": 1,
        "user_id": 42,
        "username": "admin",
        "action": "PAY",
        "entity_type": "payroll",
        "entity_id": "17",
        "method": "POST",
        "path": "/api/payroll/17/pay",
        "ip_address": "192.168.1.10",
        "request_id": "req_abc123",
        "created_at": "2025-06-01T10:22:00Z"
      }
    ],
    "total": 150,
    "page": 1,
    "limit": 20,
    "offset": 0
  }
}
```

---

## Integration

The module is wired in `server/router.go`:

```go
auditRepo := auditlog.NewRepository(deps.DB)
api.Use(auditlog.AuditMiddleware(auditRepo))   // must be before route registrations

// ... all other module route registrations ...

auditlog.RegisterRoutes(api, auditRepo)
```

The middleware must be registered **before** any route `RegisterRoutes` calls so that Gin includes it in the handler chain for all `/api` routes.
