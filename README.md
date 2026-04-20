# Hezzet Market Backend

REST API backend for market management — products, stock, customers, sales, and notifications.

Built with **Go + Gin + PostgreSQL (pgx) + Redis**.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.25 |
| HTTP | Gin |
| Database | PostgreSQL (pgxpool, no ORM) |
| Queue | Redis (async SMS jobs) |
| Auth | JWT (access + refresh tokens) |
| Docs | Swagger / OpenAPI via swag |
| Migrations | golang-migrate |

## Project Structure

```
.
├── main.go                        # Entry point, graceful shutdown
├── config/config.go               # Env var loader (godotenv)
├── server/
│   ├── router.go                  # Route registration, Deps struct
│   └── middleware.go              # ErrorMiddleware (error classification)
├── middleware/
│   ├── jwt_middleware.go          # JWT auth + token_version validation
│   ├── rbac_middleware.go         # Role-based access (admin bypass)
│   ├── pagination_middleware.go   # ?page=&limit= → Pagination struct
│   ├── request_id.go             # X-Request-Id header
│   └── swagger_guard.go          # IP allowlist + basic auth for Swagger
├── pkg/
│   ├── database/database.go      # pgxpool connection setup
│   ├── errors/errors.go          # AppError domain errors
│   └── response/response.go      # Standard JSON envelope
├── modules/
│   ├── auth/                     # Authentication & user management
│   ├── category/                 # Product categories (tree support)
│   ├── product/                  # Product catalog
│   ├── supplier/                 # Supplier management
│   ├── customer/                 # Customer management
│   ├── workers/                  # Employee management
│   ├── warehouse/                # Warehouse management
│   ├── stock/                    # Stock ledger (in/out/transfer/move)
│   └── notification/             # SMS queue + workers + audit log
├── migrations/                   # 001..017 sequential SQL migrations
├── cmd/
│   ├── migrate/                  # Migration CLI tool
│   └── seed/                     # Database seeding
└── docs/                         # Generated Swagger files
```

Each module follows: `model.go` → `repository.go` → `service.go` → `handler.go` → `routes.go`

## Quick Start

```bash
# 1. Clone & install deps
git clone <repo-url> && cd hezzet_market_backend
go mod download

# 2. Configure environment
cp .env.example .env
# Edit .env with your DB_DSN, JWT secrets, etc.

# 3. Run migrations
go run ./cmd/migrate -cmd up

# 4. Start server
go run .
```

Server starts on `HTTP_ADDR` (default `:8080`).

Swagger UI available at `/swagger/index.html` (dev mode: open; prod: IP allowlist + basic auth).

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `dev` | Environment: `dev`, `prod`, `stage` |
| `HTTP_ADDR` | `:8080` | Listen address |
| `DB_DSN` | — | PostgreSQL connection string |
| `JWT_SECRET` | — | Fallback JWT secret |
| `ACCESS_TOKEN_SECRET` | — | Access token signing key |
| `REFRESH_TOKEN_SECRET` | — | Refresh token signing key |
| `ACCESS_TOKEN_EXPIRES_IN` | `15m` | Access token TTL |
| `REFRESH_TOKEN_EXPIRES_IN` | `168h` | Refresh token TTL |
| `CORS_ALLOWED_ORIGINS` | — | Comma-separated origins |
| `REDIS_URL` | — | Redis URL (enables notifications) |
| `SMS_PROVIDER` | `log` | `log` or `twilio` |
| `SMS_FROM` | `HezzetMarket` | SMS sender ID |
| `SMS_WORKERS` | `1` | Worker goroutines |
| `ADMIN_PHONES` | — | Comma-separated admin phones |
| `LOW_STOCK_DEFAULT` | `10` | Low-stock alert threshold |
| `LOW_STOCK_DEDUP_TTL` | `6h` | Alert dedup window |
| `SMS_RATE_LIMIT_PER_HOUR` | `3` | Per-phone SMS rate limit |
| `SWAGGER_USER` | — | Swagger basic auth user (prod) |
| `SWAGGER_PASS` | — | Swagger basic auth password (prod) |
| `SWAGGER_ALLOW_IPS` | — | CIDR allowlist for Swagger (prod) |
| `SWAGGER_HOST` | — | Override Swagger host header |
| `SWAGGER_SCHEMES` | — | `http`, `https` |

## Authentication & Authorization

All `/api/*` routes require a valid JWT Bearer token.

**Token flow:**
1. `POST /auth/login` → returns `access_token` + `refresh_token`
2. Use `Authorization: Bearer <access_token>` on all API calls
3. `POST /auth/refresh` → exchange refresh token for new pair

**RBAC roles:** `admin`, `manager`, `operator`, `cashier`

- `admin` bypasses all role checks
- `RequireRoles("operator")` → allows `operator` + `admin`
- `RequireRoles()` (empty) → admin-only

Token revocation via `token_version` — changing password or blocking a user invalidates all existing tokens.

---

## API Endpoints

### Health

```
GET /health
```

No authentication required. Returns server health status.

---

### Auth

#### Register

```
POST /auth/register
```

Public. Create a new account (no roles assigned by default).

**Request:**
```json
{
  "username": "johndoe",
  "password": "secret123",
  "full_name": "John Doe",
  "phone": "+99361234567",
  "email": "john@example.com"
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `username` | string | yes | min=3, max=50 |
| `password` | string | yes | min=6 |
| `full_name` | string | no | |
| `phone` | string | no | |
| `email` | string | no | |

**Response (201):**
```json
{
  "success": true,
  "status_code": 201,
  "data": {
    "id": 1,
    "username": "johndoe",
    "full_name": "John Doe",
    "phone": "+99361234567",
    "email": "john@example.com",
    "is_active": true,
    "created_at": "2026-02-24T10:00:00Z",
    "updated_at": "2026-02-24T10:00:00Z",
    "roles": []
  }
}
```

---

#### Login

```
POST /auth/login
```

Public. Authenticate with username and password.

**Request:**
```json
{
  "username": "johndoe",
  "password": "secret123"
}
```

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900,
    "user": {
      "id": 1,
      "username": "johndoe",
      "full_name": "John Doe",
      "phone": "+99361234567",
      "email": "john@example.com",
      "is_active": true,
      "last_login_at": "2026-02-24T10:05:00Z",
      "created_at": "2026-02-24T10:00:00Z",
      "updated_at": "2026-02-24T10:05:00Z"
    },
    "roles": ["operator"]
  }
}
```

---

#### Refresh Token

```
POST /auth/refresh
```

Public. Exchange a valid refresh token for a new token pair.

**Request:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

---

#### Create User (admin only)

```
POST /auth/users
Authorization: Bearer <token>
```

Create a new user with roles. If no roles provided, defaults to `"operator"`. Admin role must be exclusive (cannot combine with other roles).

**Request:**
```json
{
  "username": "cashier1",
  "password": "secure456",
  "full_name": "Jane Smith",
  "phone": "+99367654321",
  "email": "jane@example.com",
  "is_active": true,
  "roles": ["cashier"]
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `username` | string | yes | min=3, max=50 |
| `password` | string | yes | min=6 |
| `full_name` | string | no | |
| `phone` | string | no | |
| `email` | string | no | |
| `is_active` | bool | no | default: true |
| `roles` | string[] | no | default: ["operator"] |

**Response (201):**
```json
{
  "success": true,
  "status_code": 201,
  "data": {
    "id": 2,
    "username": "cashier1",
    "full_name": "Jane Smith",
    "phone": "+99367654321",
    "email": "jane@example.com",
    "is_active": true,
    "created_at": "2026-02-24T11:00:00Z",
    "updated_at": "2026-02-24T11:00:00Z",
    "roles": ["cashier"]
  }
}
```

---

#### List Users (admin only)

```
GET /auth/users?page=1&limit=10&search=john&order_by=created_at&order_direction=desc
Authorization: Bearer <token>
```

| Query Param | Type | Default | Description |
|-------------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `limit` | int | 10 | Items per page (max 100) |
| `skip` | int | 0 | Legacy offset, overrides page |
| `search` | string | — | Search username or full_name |
| `order_by` | string | `created_at` | `username` or `created_at` |
| `order_direction` | string | `desc` | `asc` or `desc` |

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": {
    "items": [
      {
        "id": 1,
        "username": "johndoe",
        "full_name": "John Doe",
        "phone": "+99361234567",
        "email": "john@example.com",
        "is_active": true,
        "created_at": "2026-02-24T10:00:00Z",
        "updated_at": "2026-02-24T10:00:00Z",
        "roles": ["operator"]
      }
    ],
    "total": 1,
    "limit": 10,
    "offset": 0
  },
  "meta": {
    "pagination": {
      "page": 1,
      "limit": 10,
      "offset": 0,
      "total": 1,
      "has_next": false,
      "has_prev": false,
      "total_pages": 1
    }
  }
}
```

---

#### Get User (admin only)

```
GET /auth/users/:id
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": {
    "id": 1,
    "username": "johndoe",
    "full_name": "John Doe",
    "phone": "+99361234567",
    "email": "john@example.com",
    "is_active": true,
    "last_login_at": "2026-02-24T10:05:00Z",
    "created_at": "2026-02-24T10:00:00Z",
    "updated_at": "2026-02-24T10:00:00Z",
    "roles": ["operator"]
  }
}
```

---

#### Update User (admin only)

```
PATCH /auth/users/:id
Authorization: Bearer <token>
```

All fields optional. Admin role must be exclusive.

**Request:**
```json
{
  "full_name": "John D. Updated",
  "roles": ["manager"]
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `username` | string | no | min=3, max=50 |
| `full_name` | string | no | |
| `phone` | string | no | |
| `email` | string | no | |
| `is_active` | bool | no | |
| `roles` | string[] | no | cannot be empty if provided |

**Response (200):** Same shape as Get User.

---

#### Delete User (admin only)

```
DELETE /auth/users/:id
Authorization: Bearer <token>
```

Soft-deletes user. Admin cannot delete themselves.

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": { "deleted": true }
}
```

---

#### Change Password (admin or self)

```
POST /auth/users/:id/password
Authorization: Bearer <token>
```

Non-admin users can only change their own password. Increments `token_version` to invalidate all existing tokens.

**Request:**
```json
{
  "password": "newSecure789"
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `password` | string | yes | min=6 |

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": { "changed": true }
}
```

---

#### Block User (admin only)

```
POST /auth/users/:id/block
Authorization: Bearer <token>
```

Block a user account. Invalidates all tokens. Admin cannot block themselves.

**Request:**
```json
{
  "reason": "Suspicious activity detected"
}
```

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": {
    "id": 2,
    "username": "cashier1",
    "is_active": true,
    "blocked_at": "2026-02-24T12:00:00Z",
    "blocked_reason": "Suspicious activity detected",
    "roles": ["cashier"]
  }
}
```

---

#### Unblock User (admin only)

```
POST /auth/users/:id/unblock
Authorization: Bearer <token>
```

Clears `blocked_at` and `blocked_reason`. Increments token_version. Admin cannot unblock themselves.

**Response (200):** Same shape as Get User (blocked fields cleared).

---

### Categories

**Read** — roles: `operator`, `cashier`, `manager`
**Write** — roles: `operator`

#### List Categories

```
GET /api/categories?page=1&limit=10&search=elec&order_by=name&order_direction=asc
Authorization: Bearer <token>
```

| Query Param | Type | Default | Description |
|-------------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `limit` | int | 10 | Items per page (max 200) |
| `skip` | int | 0 | Legacy offset, overrides page |
| `search` | string | — | Search by name (ILIKE) |
| `order_by` | string | `created_at` | `name` or `created_at` |
| `order_direction` | string | `desc` | `asc` or `desc` |

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": [
    {
      "id": 1,
      "name": "Electronics",
      "parent_id": null,
      "parent_name": null,
      "is_active": true,
      "created_at": "2026-01-10T08:00:00Z"
    },
    {
      "id": 5,
      "name": "Smartphones",
      "parent_id": 1,
      "parent_name": "Electronics",
      "is_active": true,
      "created_at": "2026-01-15T10:00:00Z"
    }
  ],
  "meta": {
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 2,
      "has_next": false,
      "has_prev": false,
      "total_pages": 1
    }
  }
}
```

---

#### Get Category

```
GET /api/categories/:id
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": {
    "id": 5,
    "name": "Smartphones",
    "parent_id": 1,
    "parent_name": "Electronics",
    "is_active": true,
    "created_at": "2026-01-15T10:00:00Z"
  }
}
```

---

#### Get Category Tree

```
GET /api/categories/tree
Authorization: Bearer <token>
```

Returns all active categories as a nested hierarchy.

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": {
    "items": [
      {
        "id": 1,
        "name": "Electronics",
        "children": [
          {
            "id": 5,
            "name": "Smartphones",
            "parent_id": 1,
            "children": []
          }
        ]
      },
      {
        "id": 2,
        "name": "Groceries",
        "children": []
      }
    ]
  }
}
```

---

#### Create Category

```
POST /api/categories
Authorization: Bearer <token>
```

**Request:**
```json
{
  "name": "Tablets",
  "parent_id": 1,
  "is_active": true
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `name` | string | yes | min=1, max=255 |
| `parent_id` | int | no | null = root category |
| `is_active` | bool | no | default: true |

**Response (201):**
```json
{
  "success": true,
  "status_code": 201,
  "data": {
    "id": 6,
    "name": "Tablets",
    "parent_id": 1,
    "is_active": true,
    "created_at": "2026-02-24T14:00:00Z"
  }
}
```

---

#### Update Category

```
PATCH /api/categories/:id
Authorization: Bearer <token>
```

**Request:**
```json
{
  "name": "Tablet Devices"
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `name` | string | no | min=1, max=255 |
| `parent_id` | int | no | |
| `is_active` | bool | no | |

**Response (200):** Same shape as Create response.

---

#### Delete Category (soft)

```
DELETE /api/categories/:id
Authorization: Bearer <token>
```

Sets `is_active=false`.

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": { "deleted": true }
}
```

---

### Products

**Read** — roles: `operator`, `cashier`, `manager`
**Write** — roles: `operator`

#### List Products

```
GET /api/products?page=1&limit=10&search=milk&order_by=name&order_direction=asc
Authorization: Bearer <token>
```

| Query Param | Type | Default | Description |
|-------------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `limit` | int | 10 | Items per page (max 200) |
| `skip` | int | 0 | Legacy offset |
| `search` | string | — | Search by name, SKU, or barcode |
| `order_by` | string | `created_at` | `name` or `created_at` |
| `order_direction` | string | `desc` | `asc` or `desc` |

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": [
    {
      "id": 1,
      "name": "Whole Milk 1L",
      "sku": "MLK-001",
      "barcode": "4901234567890",
      "unit": "piece",
      "purchase_price": 5000,
      "sale_price": 7500,
      "is_active": true,
      "unit_type": "piece",
      "unit_scale": 1000,
      "created_at": "2026-01-20T09:00:00Z",
      "updated_at": "2026-01-20T09:00:00Z"
    }
  ],
  "meta": {
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 1,
      "has_next": false,
      "has_prev": false,
      "total_pages": 1
    }
  }
}
```

---

#### Get Product

```
GET /api/products/:id
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": {
    "id": 1,
    "name": "Whole Milk 1L",
    "sku": "MLK-001",
    "barcode": "4901234567890",
    "unit": "piece",
    "purchase_price": 5000,
    "sale_price": 7500,
    "is_active": true,
    "unit_type": "piece",
    "unit_scale": 1000,
    "created_at": "2026-01-20T09:00:00Z",
    "updated_at": "2026-01-20T09:00:00Z"
  }
}
```

---

#### Get Product Card

```
GET /api/products/:id/card
Authorization: Bearer <token>
```

Returns product with aggregated stock, categories, and metadata.

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": {
    "product": {
      "id": 1,
      "name": "Whole Milk 1L",
      "sku": "MLK-001",
      "barcode": "4901234567890",
      "unit": "piece",
      "purchase_price": 5000,
      "sale_price": 7500,
      "is_active": true,
      "unit_type": "piece",
      "unit_scale": 1000,
      "created_at": "2026-01-20T09:00:00Z",
      "updated_at": "2026-01-20T09:00:00Z"
    },
    "stock": 25000,
    "tags": [],
    "supplier_ids": [1, 3],
    "categories": [
      { "id": 2, "name": "Groceries" }
    ]
  }
}
```

---

#### Get Product Categories

```
GET /api/products/:id/categories
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": [
    { "id": 2, "name": "Groceries" },
    { "id": 7, "name": "Dairy" }
  ]
}
```

---

#### Create Product

```
POST /api/products
Authorization: Bearer <token>
```

**Request:**
```json
{
  "name": "Whole Milk 1L",
  "sku": "MLK-001",
  "barcode": "4901234567890",
  "unit": "piece",
  "purchase_price": 5000,
  "sale_price": 7500,
  "is_active": true,
  "category_ids": [2, 7],
  "unit_type": "piece"
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `name` | string | yes | |
| `sku` | string | no | |
| `barcode` | string | no | |
| `unit` | string | yes | |
| `purchase_price` | int64 | no | in cents |
| `sale_price` | int64 | no | in cents |
| `is_active` | bool | no | default: true |
| `category_ids` | int64[] | no | must be active category IDs |
| `unit_type` | string | no | `piece`, `kg`, `liter`, `meter`, `box` |

**Response (201):** Same shape as Get Product.

---

#### Update Product

```
PATCH /api/products/:id
Authorization: Bearer <token>
```

All fields optional.

**Request:**
```json
{
  "sale_price": 8000,
  "category_ids": [2, 7, 9]
}
```

**Response (200):** Same shape as Get Product.

---

#### Set Product Categories

```
PUT /api/products/:id/categories
Authorization: Bearer <token>
```

Replaces all category associations.

**Request:**
```json
{
  "category_ids": [2, 7]
}
```

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": [
    { "id": 2, "name": "Groceries" },
    { "id": 7, "name": "Dairy" }
  ]
}
```

---

#### Remove Product Category

```
DELETE /api/products/:id/categories/:categoryId
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": { "deleted": true }
}
```

---

#### Delete Product (soft)

```
DELETE /api/products/:id
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": { "deleted": true }
}
```

---

### Suppliers

**All endpoints** — roles: `operator`

#### List Suppliers

```
GET /api/suppliers?page=1&limit=10&search=farm
Authorization: Bearer <token>
```

| Query Param | Type | Default | Description |
|-------------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `limit` | int | 10 | Items per page (max 200) |
| `skip` | int | 0 | Legacy offset |
| `search` | string | — | Search by name, phone, or email |
| `order_by` | string | `created_at` | `name` or `created_at` |
| `order_direction` | string | `desc` | `asc` or `desc` |

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": [
    {
      "id": 1,
      "name": "Green Farm LLC",
      "phone": "+99361000000",
      "email": "contact@greenfarm.com",
      "address": "123 Farm Road",
      "is_active": true,
      "created_at": "2026-01-05T08:00:00Z"
    }
  ],
  "meta": {
    "pagination": { "page": 1, "limit": 10, "total": 1 }
  }
}
```

---

#### Get Supplier

```
GET /api/suppliers/:id
Authorization: Bearer <token>
```

**Response (200):** Single supplier object (same shape as list item).

---

#### Create Supplier

```
POST /api/suppliers
Authorization: Bearer <token>
```

**Request:**
```json
{
  "name": "Green Farm LLC",
  "phone": "+99361000000",
  "email": "contact@greenfarm.com",
  "address": "123 Farm Road"
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `name` | string | yes | min=1, max=255 |
| `phone` | string | no | max=50 |
| `email` | string | no | valid email, max=255 |
| `address` | string | no | |

**Response (201):** Supplier object.

---

#### Update Supplier

```
PATCH /api/suppliers/:id
Authorization: Bearer <token>
```

**Request:**
```json
{
  "phone": "+99361999999"
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `name` | string | no | min=1, max=255 |
| `phone` | string | no | max=50 |
| `email` | string | no | valid email, max=255 |
| `address` | string | no | |
| `is_active` | bool | no | |

**Response (200):** Supplier object.

---

#### Delete Supplier (soft)

```
DELETE /api/suppliers/:id
Authorization: Bearer <token>
```

**Response (200):**
```json
{ "success": true, "status_code": 200, "data": { "deleted": true } }
```

---

### Customers

**Read** — roles: `cashier`, `operator`, `manager`

#### List Customers

```
GET /api/customers?page=1&limit=10&search=ali&active_only=true&order_by=total_spent&order_direction=desc
Authorization: Bearer <token>
```

| Query Param | Type | Default | Description |
|-------------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `limit` | int | 10 | Items per page (max 200) |
| `skip` | int | 0 | Legacy offset |
| `search` | string | — | Search name, phone, or email |
| `active_only` | bool | false | Only active customers |
| `order_by` | string | `created_at` | `name`, `created_at`, or `total_spent` |
| `order_direction` | string | `desc` | `asc` or `desc` |

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": [
    {
      "id": 1,
      "name": "Ali Orazov",
      "phone": "+99362000000",
      "email": "ali@example.com",
      "type": "regular",
      "total_spent": 150000,
      "bonus_points": 1500,
      "is_active": true,
      "notes": "",
      "created_at": "2026-01-12T09:00:00Z",
      "updated_at": "2026-02-20T15:30:00Z"
    }
  ],
  "meta": {
    "pagination": { "page": 1, "limit": 10, "total": 1 }
  }
}
```

---

#### Get Customer

```
GET /api/customers/:id
Authorization: Bearer <token>
```

Returns customer even if inactive (404 only if soft-deleted).

**Response (200):** Single customer object.

---

#### Create Customer

Roles: `cashier`, `manager`

```
POST /api/customers
Authorization: Bearer <token>
```

**Request:**
```json
{
  "name": "Ali Orazov",
  "phone": "+99362000000",
  "email": "ali@example.com",
  "type": "regular",
  "notes": "Frequent buyer"
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `name` | string | yes | min=1, max=255 |
| `phone` | string | no | max=50 |
| `email` | string | no | valid email, max=255 |
| `type` | string | no | `regular`, `vip`, or `wholesale` |
| `notes` | string | no | |

**Response (201):** Customer object.

---

#### Add Spent Amount

Roles: `cashier`, `manager`

```
POST /api/customers/:id/spent
Authorization: Bearer <token>
```

Records spent amount and auto-calculates bonus points (1% of amount).

**Request:**
```json
{
  "amount_cents": 50000
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `amount_cents` | int64 | yes | must be > 0 |

**Response (200):** Updated customer object with new `total_spent` and `bonus_points`.

---

#### Update Contact Info

Roles: `cashier`, `operator`, `manager`

```
PATCH /api/customers/:id/contact
Authorization: Bearer <token>
```

**Request:**
```json
{
  "phone": "+99362111111",
  "notes": "Updated phone number"
}
```

| Field | Type | Required |
|-------|------|----------|
| `name` | string | no |
| `phone` | string | no |
| `email` | string | no |
| `notes` | string | no |

**Response (200):** Updated customer object.

---

#### Update Business Fields

Roles: `manager`

```
PATCH /api/customers/:id/admin
Authorization: Bearer <token>
```

**Request:**
```json
{
  "type": "vip",
  "is_active": true
}
```

| Field | Type | Required |
|-------|------|----------|
| `type` | string | no |
| `is_active` | bool | no |

**Response (200):** Updated customer object.

---

#### Delete Customer (admin only)

```
DELETE /api/customers/:id
Authorization: Bearer <token>
```

Soft-deletes: sets `deleted_at` and `is_active=false`.

**Response (200):**
```json
{ "success": true, "status_code": 200, "data": { "deleted": true } }
```

---

### Workers

**Read** — roles: `manager`, `operator`
**Write** — roles: `manager`
**Delete** — admin only

#### List Workers

```
GET /api/workers?page=1&limit=10&search=ali&active_only=true
Authorization: Bearer <token>
```

| Query Param | Type | Default | Description |
|-------------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `limit` | int | 10 | Items per page (max 200) |
| `skip` | int | 0 | Legacy offset |
| `search` | string | — | Search name, phone, or email |
| `active_only` | bool | true | Only active workers |
| `order_by` | string | `created_at` | `name` or `created_at` |
| `order_direction` | string | `desc` | `asc` or `desc` |

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": [
    {
      "id": 1,
      "name": "Merdan Atayev",
      "position": "Warehouse Clerk",
      "department": "Logistics",
      "phone": "+99363000000",
      "email": "merdan@example.com",
      "address": "456 Main St",
      "salary": 5000.00,
      "hire_date": "2025-06-01T00:00:00Z",
      "is_active": true,
      "notes": "",
      "created_at": "2025-06-01T08:00:00Z",
      "updated_at": "2025-06-01T08:00:00Z"
    }
  ],
  "meta": {
    "pagination": { "page": 1, "limit": 10, "total": 1 }
  }
}
```

---

#### Get Worker

```
GET /api/workers/:id
Authorization: Bearer <token>
```

**Response (200):** Single worker object. Returns 404 only if soft-deleted.

---

#### Create Worker

```
POST /api/workers
Authorization: Bearer <token>
```

**Request:**
```json
{
  "name": "Merdan Atayev",
  "position": "Warehouse Clerk",
  "department": "Logistics",
  "phone": "+99363000000",
  "email": "merdan@example.com",
  "address": "456 Main St",
  "salary": 5000.00,
  "hire_date": "2025-06-01",
  "notes": ""
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `name` | string | yes | min=1, max=255 |
| `position` | string | no | max=100 |
| `department` | string | no | max=100 |
| `phone` | string | no | max=50 |
| `email` | string | no | valid email, max=255 |
| `address` | string | no | |
| `salary` | float64 | no | >= 0 |
| `hire_date` | string | no | date string |
| `notes` | string | no | |

**Response (201):** Worker object.

---

#### Update Worker

```
PATCH /api/workers/:id
Authorization: Bearer <token>
```

All fields optional.

**Request:**
```json
{
  "salary": 6000.00,
  "position": "Senior Clerk"
}
```

**Response (200):** Updated worker object.

---

#### Delete Worker (admin only)

```
DELETE /api/workers/:id
Authorization: Bearer <token>
```

**Response (200):**
```json
{ "success": true, "status_code": 200, "data": { "deleted": true } }
```

---

### Warehouses

**Read** — roles: `operator`, `cashier`, `manager`
**Create** — roles: `manager`

#### List Warehouses

```
GET /api/warehouses?page=1&limit=10
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": [
    {
      "id": 1,
      "name": "Main Warehouse",
      "address": "789 Industrial Ave",
      "is_active": true,
      "created_at": "2025-12-01T08:00:00Z",
      "updated_at": "2025-12-01T08:00:00Z"
    }
  ],
  "meta": {
    "pagination": { "page": 1, "limit": 10, "total": 1 }
  }
}
```

---

#### Get Warehouse

```
GET /api/warehouses/:id
Authorization: Bearer <token>
```

**Response (200):** Single warehouse object.

---

#### Create Warehouse

```
POST /api/warehouses
Authorization: Bearer <token>
```

**Request:**
```json
{
  "name": "Main Warehouse",
  "address": "789 Industrial Ave"
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `name` | string | yes | min=1, max=255 |
| `address` | string | no | |

**Response (201):** Warehouse object.

---

### Stock

**Write** — roles: `operator`, `manager`
**Read** — roles: `cashier`, `operator`, `manager`

All quantities use `qty_milli` (SCALE=1000). For example, `5000` = 5 units. All write operations require a UUID `idempotency_key` for exactly-once delivery.

#### Stock In

```
POST /api/stock/in
Authorization: Bearer <token>
```

Add stock to a warehouse (purchase/receiving).

**Request:**
```json
{
  "warehouse_id": 1,
  "product_id": 1,
  "qty_milli": 10000,
  "idempotency_key": "550e8400-e29b-41d4-a716-446655440000",
  "price_cents": 5000,
  "worker_id": 1
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `warehouse_id` | int64 | yes | > 0 |
| `product_id` | int64 | yes | > 0 |
| `qty_milli` | int64 | yes | > 0 |
| `idempotency_key` | string | yes | UUID format |
| `price_cents` | int64 | no | unit cost in cents |
| `worker_id` | int64 | no | |

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": {
    "detail": {
      "id": 42,
      "idempotency_key": "550e8400-e29b-41d4-a716-446655440000",
      "warehouse_id": 1,
      "product_id": 1,
      "delta_milli": 10000,
      "type": "in",
      "price_cents": 5000,
      "worker_id": 1,
      "created_by": 1,
      "created_at": "2026-02-24T14:00:00Z"
    },
    "item": {
      "warehouse_id": 1,
      "product_id": 1,
      "qty_milli": 35000,
      "avg_cost_cents": 5000,
      "total_cost_cents": 175000,
      "updated_at": "2026-02-24T14:00:00Z"
    }
  }
}
```

---

#### Stock Out

```
POST /api/stock/out
Authorization: Bearer <token>
```

Remove stock from a warehouse (sale/consumption).

**Request:**
```json
{
  "warehouse_id": 1,
  "product_id": 1,
  "qty_milli": 3000,
  "idempotency_key": "660e8400-e29b-41d4-a716-446655440001",
  "price_cents": 7500,
  "worker_id": 2
}
```

Fields same as Stock In. **Response:** Same shape as Stock In (`detail` + `item`).

---

#### Transfer

```
POST /api/stock/transfer
Authorization: Bearer <token>
```

Transfer stock between warehouses.

**Request:**
```json
{
  "from_warehouse_id": 1,
  "to_warehouse_id": 2,
  "product_id": 1,
  "qty_milli": 5000,
  "idempotency_key": "770e8400-e29b-41d4-a716-446655440002"
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `from_warehouse_id` | int64 | yes | > 0 |
| `to_warehouse_id` | int64 | yes | > 0 |
| `product_id` | int64 | yes | > 0 |
| `qty_milli` | int64 | yes | > 0 |
| `idempotency_key` | string | yes | UUID format |

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": {
    "out_detail": {
      "id": 43,
      "warehouse_id": 1,
      "product_id": 1,
      "delta_milli": -5000,
      "type": "transfer_out",
      "created_at": "2026-02-24T14:30:00Z"
    },
    "in_detail": {
      "id": 44,
      "warehouse_id": 2,
      "product_id": 1,
      "delta_milli": 5000,
      "type": "transfer_in",
      "created_at": "2026-02-24T14:30:00Z"
    },
    "from_item": {
      "warehouse_id": 1,
      "product_id": 1,
      "qty_milli": 27000,
      "avg_cost_cents": 5000,
      "total_cost_cents": 135000,
      "updated_at": "2026-02-24T14:30:00Z"
    },
    "to_item": {
      "warehouse_id": 2,
      "product_id": 1,
      "qty_milli": 5000,
      "avg_cost_cents": 5000,
      "total_cost_cents": 25000,
      "updated_at": "2026-02-24T14:30:00Z"
    }
  }
}
```

---

#### Stock Move (adjustment/damaged/etc)

```
POST /api/stock/move
Authorization: Bearer <token>
```

Universal delta-based adjustment. `delta_milli` can be positive or negative (but not zero).

**Request:**
```json
{
  "warehouse_id": 1,
  "product_id": 1,
  "delta_milli": -2000,
  "type": "damaged",
  "idempotency_key": "880e8400-e29b-41d4-a716-446655440003",
  "price_cents": 5000,
  "worker_id": 1
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `warehouse_id` | int64 | yes | > 0 |
| `product_id` | int64 | yes | > 0 |
| `delta_milli` | int64 | yes | != 0 |
| `type` | string | yes | e.g. `damaged`, `adjustment` |
| `idempotency_key` | string | yes | UUID format |
| `price_cents` | int64 | no | |
| `worker_id` | int64 | no | |

**Response (200):** Same shape as Stock In (`detail` + `item`).

---

#### Opening Balance

```
POST /api/stock/opening-balance
Authorization: Bearer <token>
```

Set initial stock balance for a product in a warehouse.

**Request:**
```json
{
  "warehouse_id": 1,
  "product_id": 1,
  "qty_milli": 50000,
  "price_cents": 4500,
  "idempotency_key": "990e8400-e29b-41d4-a716-446655440004",
  "worker_id": 1
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `warehouse_id` | int64 | yes | > 0 |
| `product_id` | int64 | yes | > 0 |
| `qty_milli` | int64 | yes | > 0 |
| `price_cents` | int64 | yes | > 0 |
| `idempotency_key` | string | yes | UUID format |
| `worker_id` | int64 | no | |

**Response (200):** Same shape as Stock In (`detail` + `item`).

---

#### Get Stock Balance

```
GET /api/stock/balance?warehouse_id=1&product_id=1
Authorization: Bearer <token>
```

| Query Param | Type | Required | Description |
|-------------|------|----------|-------------|
| `warehouse_id` | int64 | no | Filter by warehouse |
| `product_id` | int64 | no | Filter by product |

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": [
    {
      "warehouse_id": 1,
      "product_id": 1,
      "qty_milli": 25000,
      "avg_cost_cents": 5000,
      "total_cost_cents": 125000,
      "updated_at": "2026-02-24T14:00:00Z"
    }
  ]
}
```

---

#### Get Stock Details (ledger)

```
GET /api/stock/details?warehouse_id=1&product_id=1&type=in&date_from=2026-01-01T00:00:00Z&page=1&limit=20
Authorization: Bearer <token>
```

| Query Param | Type | Required | Description |
|-------------|------|----------|-------------|
| `warehouse_id` | int64 | no | Filter by warehouse |
| `product_id` | int64 | no | Filter by product |
| `type` | string | no | `in`, `out`, `transfer_in`, `transfer_out`, `damaged`, `adjustment` |
| `date_from` | string | no | RFC3339 timestamp |
| `date_to` | string | no | RFC3339 timestamp |
| `page` | int | no | Page number |
| `limit` | int | no | Items per page |

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": [
    {
      "id": 42,
      "idempotency_key": "550e8400-e29b-41d4-a716-446655440000",
      "warehouse_id": 1,
      "product_id": 1,
      "delta_milli": 10000,
      "type": "in",
      "price_cents": 5000,
      "worker_id": 1,
      "created_by": 1,
      "created_at": "2026-02-24T14:00:00Z"
    }
  ],
  "meta": {
    "pagination": { "page": 1, "limit": 20, "total": 1 }
  }
}
```

---

### Notifications — SMS (admin only)

#### List SMS Logs

```
GET /api/notifications/sms?status=sent&phone=+99361234567&page=1&limit=20
Authorization: Bearer <token>
```

| Query Param | Type | Required | Description |
|-------------|------|----------|-------------|
| `status` | string | no | `queued`, `sending`, `sent`, `retrying`, `rate_limited`, `dlq`, `failed` |
| `type` | string | no | SMS type filter |
| `phone` | string | no | Filter by phone number |
| `job_id` | string | no | Filter by job ID |
| `from_date` | string | no | RFC3339 timestamp |
| `to_date` | string | no | RFC3339 timestamp |
| `page` | int | no | Page number |
| `limit` | int | no | Items per page |

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": [
    {
      "id": 1,
      "job_id": "sms-abc123",
      "type": "low_stock",
      "to_phone": "+99361234567",
      "message": "Low stock alert: Whole Milk 1L is below threshold",
      "sms_from": "HezzetMarket",
      "status": "sent",
      "attempt": 1,
      "max_attempts": 3,
      "provider": "twilio",
      "created_at": "2026-02-24T10:00:00Z",
      "updated_at": "2026-02-24T10:00:05Z",
      "sent_at": "2026-02-24T10:00:05Z"
    }
  ],
  "meta": {
    "pagination": { "page": 1, "limit": 20, "total": 1 }
  }
}
```

---

#### Get SMS Log

```
GET /api/notifications/sms/:job_id
Authorization: Bearer <token>
```

**Response (200):** Single SMS log object (same shape as list item).

---

#### Requeue SMS Job

```
POST /api/notifications/sms/:job_id/requeue
Authorization: Bearer <token>
```

Re-enqueue a job from `dlq`, `failed`, or `rate_limited` status back into the queue.

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "data": { "requeued": true }
}
```

---

## Response Format

All responses use a standard envelope:

**Success:**
```json
{
  "success": true,
  "status_code": 200,
  "data": { ... },
  "meta": {da
    "timestamp": "2026-02-24T12:00:00Z",
    "pagination": {
      "page": 1,
      "limit": 10,
      "offset": 0,
      "total": 42,
      "has_next": true,
      "has_prev": false,
      "total_pages": 5
    }
  }
}
```

**Error:**
```json
{
  "success": false,
  "status_code": 400,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "request validation failed",
    "details": { "name": "field is required" },
    "request_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

## Error Codes

| HTTP | Code | When |
|------|------|------|
| 400 | `VALIDATION_ERROR` | Invalid input / binding failure |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | Insufficient role |
| 404 | `NOT_FOUND` | Resource doesn't exist |
| 409 | `*_ALREADY_EXISTS` | Unique constraint violation |
| 499 | — | Client closed connection |
| 500 | `INTERNAL_ERROR` | Server error (details never exposed) |
| 504 | — | Request timeout |

## Migrations

```bash
# Apply all pending
go run ./cmd/migrate -cmd up

# Rollback last
go run ./cmd/migrate -cmd down

# Apply N steps
go run ./cmd/migrate -cmd steps -n 2

# Show current version
go run ./cmd/migrate -cmd version
```

Migration files follow `NNN_name.up.sql` / `NNN_name.down.sql` convention in `migrations/`.

Current schema version: **017** (sms_logs).

## Architecture Notes

- **No ORM** — all SQL written explicitly with pgxpool for full control
- **Fail-closed JWT** — `token_version` must exist and match DB; any mismatch rejects the token
- **Nil-safe notification service** — callers never nil-guard; service methods no-op gracefully when Redis is unavailable
- **Idempotent stock operations** — all stock writes enforce UUID idempotency keys
- **Monetary values** — stored as `int64` cents (migration 007)
- **Quantities** — stored as `int64` milli-units (SCALE=1000) for fractional support without floats
- **Async SMS** — Redis queue with configurable workers, per-phone rate limiting, automatic retry with DLQ
