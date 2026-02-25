# API Documentation for Frontend

Base URL: `/api`

All endpoints require `Authorization: Bearer <token>` header.

---

## 1. Finance

Manages income/expense transactions and payments.

### GET /api/payment-types

Returns all available payment types (cash, card, etc.).

**Roles:** cashier, operator, manager

**Response:**

```json
{
  "data": [
    { "id": 1, "code": "cash", "name": "Cash", "is_active": true }
  ]
}
```

---

### GET /api/transactions

List all financial transactions with filters.

**Roles:** cashier, operator, manager

**Query params:**

| Param            | Type   | Required | Description                                          |
| ---------------- | ------ | -------- | ---------------------------------------------------- |
| `type`           | string | no       | `income` or `expense`                                |
| `status`         | string | no       | `pending`, `partial`, `paid`, `canceled`             |
| `related_table`  | string | no       | `sale`, `purchase`, `manual`, `adjustment`           |
| `related_id`     | int    | no       | Filter by related entity ID                          |
| `payment_type_id`| int    | no       | Filter by payment type                               |
| `date_from`      | string | no       | RFC3339 datetime, inclusive                          |
| `date_to`        | string | no       | RFC3339 datetime, inclusive                          |
| `page`           | int    | no       | Page number (default 1)                              |
| `limit`          | int    | no       | Items per page (default 10)                          |

**Response:**

```json
{
  "data": [
    {
      "id": 1,
      "type": "income",
      "amount_cents": 50000,
      "status": "paid",
      "related_table": "sale",
      "related_id": 5,
      "reason": "Sale payment",
      "payment_type_id": 1,
      "created_by": 2,
      "created_at": "2026-02-25T10:00:00Z"
    }
  ],
  "meta": { "pagination": { "total": 50, "page": 1, "limit": 10 } }
}
```

---

### GET /api/transactions/{id}

Get full transaction detail with all payments.

**Roles:** cashier, operator, manager

**Response:**

```json
{
  "data": {
    "id": 1,
    "type": "income",
    "amount_cents": 50000,
    "status": "partial",
    "payments": [
      {
        "id": 1,
        "payment_type_id": 1,
        "amount_cents": 30000,
        "note": "First payment",
        "created_at": "2026-02-25T10:00:00Z"
      }
    ],
    "paid_cents": 30000,
    "debt_cents": 20000
  }
}
```

- `paid_cents` = total amount paid so far
- `debt_cents` = remaining amount owed

---

### POST /api/transactions/manual

Create a manual income or expense transaction.

**Roles:** operator, manager

**Body:**

```json
{
  "type": "income",
  "amount_cents": 100000,
  "reason": "Rent payment",
  "payment_type_id": 1,
  "payment_amount": 100000,
  "payment_note": "Paid in cash"
}
```

| Field              | Type   | Required | Description                                   |
| ------------------ | ------ | -------- | --------------------------------------------- |
| `type`             | string | yes      | `income` or `expense`                         |
| `amount_cents`     | int    | yes      | Amount in cents (> 0)                         |
| `reason`           | string | no       | Description                                   |
| `payment_type_id`  | int    | no       | Payment method. Required if payment_amount set|
| `payment_amount`   | int    | no       | Initial payment amount. Cannot exceed amount  |
| `payment_note`     | string | no       | Note for the payment                          |

- If `payment_amount` is not provided, status = `pending`
- If `payment_amount` = `amount_cents`, status = `paid`
- If `payment_amount` < `amount_cents`, status = `partial`

---

### POST /api/transactions/{id}/payments

Add a payment to an existing transaction.

**Roles:** operator, manager

**Body:**

```json
{
  "payment_type_id": 1,
  "amount_cents": 20000,
  "note": "Second payment"
}
```

| Field             | Type   | Required | Description                        |
| ----------------- | ------ | -------- | ---------------------------------- |
| `payment_type_id` | int    | yes      | Payment method ID (> 0)           |
| `amount_cents`    | int    | yes      | Payment amount in cents (> 0)     |
| `note`            | string | no       | Payment note                       |

**Errors:**

- 404 — transaction not found
- 400 — transaction already paid or canceled
- 400 — payment exceeds remaining debt

---

### POST /api/transactions/{id}/cancel

Cancel a transaction.

**Roles:** manager only

**Body:** none

**Response:** updated transaction with `status: "canceled"`

---

## 2. Sales (POS / Cashier)

Create sales from the cash register. When a sale is created:
- Product prices are **auto-fetched** from the database (cashier cannot change price)
- Stock is **automatically deducted** from the warehouse
- A finance transaction (income) is **automatically created**

### POST /api/sales

Create a new sale.

**Roles:** cashier, operator, manager

**Body:**

```json
{
  "warehouse_id": 1,
  "customer_id": 5,
  "items": [
    { "product_id": 10, "qty_milli": 2000 },
    { "product_id": 15, "qty_milli": 500 }
  ],
  "note": "Regular sale",
  "payment_type_id": 1,
  "payment_amount": 75000,
  "payment_note": "Cash payment"
}
```

| Field              | Type   | Required | Description                                    |
| ------------------ | ------ | -------- | ---------------------------------------------- |
| `warehouse_id`     | int    | yes      | Warehouse to sell from                         |
| `customer_id`      | int    | no       | Customer (null = anonymous)                    |
| `items`            | array  | yes      | At least 1 item                                |
| `items[].product_id` | int | yes      | Product ID                                     |
| `items[].qty_milli`  | int | yes      | Quantity (1 piece = 1000, 0.5 kg = 500)       |
| `note`             | string | no       | Sale note                                      |
| `payment_type_id`  | int    | no       | Payment method. Required if payment_amount set |
| `payment_amount`   | int    | no       | Payment amount in cents                        |
| `payment_note`     | string | no       | Payment note                                   |

**qty_milli explained:**

- 1 piece = `1000`
- 2 pieces = `2000`
- 0.5 kg = `500`
- 1.5 kg = `1500`

**Payment behavior:**

- No payment fields = sale on credit (transaction status = `pending`)
- With payment = transaction status = `partial` or `paid`

**Errors:**

- 400 — duplicate product_id in items
- 400 — insufficient stock
- 400 — product not found or inactive

**Response:**

```json
{
  "data": {
    "id": 1,
    "warehouse_id": 1,
    "customer_id": 5,
    "total_cents": 75000,
    "cost_cents": 45000,
    "items_count": 2,
    "items": [
      {
        "id": 1,
        "product_id": 10,
        "qty_milli": 2000,
        "unit_price_cents": 25000,
        "cost_cents": 15000,
        "line_total_cents": 50000
      }
    ],
    "transaction_id": 42,
    "created_by": 3,
    "created_at": "2026-02-25T14:00:00Z"
  }
}
```

- `total_cents` = sale revenue (sum of line totals)
- `cost_cents` = cost of goods sold
- `transaction_id` = linked finance transaction (use GET /api/transactions/{id} for payment details)

---

### GET /api/sales

List sales with filters.

**Roles:** cashier, operator, manager

**Query params:**

| Param        | Type   | Required | Description                  |
| ------------ | ------ | -------- | ---------------------------- |
| `warehouse_id` | int | no       | Filter by warehouse          |
| `customer_id`  | int | no       | Filter by customer           |
| `created_by`   | int | no       | Filter by cashier/user       |
| `date_from`    | string | no    | RFC3339 datetime, inclusive  |
| `date_to`      | string | no    | RFC3339 datetime, inclusive  |
| `page`         | int | no       | Page number (default 1)      |
| `limit`        | int | no       | Items per page (default 10)  |

---

### GET /api/sales/{id}

Get sale detail with all items.

**Roles:** cashier, operator, manager

**Response:** Full sale object with items array and transaction_id.

---

## 3. Worker Finance

Manage worker salaries, fines, and debts (advances/loans).

### POST /api/workers/compensation

Set or update worker salary configuration.

**Roles:** manager

**Body:**

```json
{
  "worker_id": 1,
  "base_salary_cents": 5000000,
  "pay_day": 15
}
```

| Field               | Type | Required | Description                      |
| ------------------- | ---- | -------- | -------------------------------- |
| `worker_id`         | int  | yes      | Worker ID (> 0)                  |
| `base_salary_cents` | int  | yes      | Monthly salary in cents (> 0)    |
| `pay_day`           | int  | yes      | Day of month for payment (1-31)  |

If compensation already exists for this worker, it updates (upsert).

---

### GET /api/workers/{id}/compensation

Get worker salary configuration.

**Roles:** manager

**Response:**

```json
{
  "data": {
    "id": 1,
    "worker_id": 5,
    "base_salary_cents": 5000000,
    "pay_day": 15,
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-02-15T00:00:00Z"
  }
}
```

Returns 404 if compensation not configured for this worker.

---

### POST /api/workers/{id}/fines

Create a fine (penalty) for a worker.

**Roles:** manager

**Body:**

```json
{
  "amount_cents": 50000,
  "reason": "Late to work"
}
```

| Field          | Type   | Required | Description              |
| -------------- | ------ | -------- | ------------------------ |
| `amount_cents` | int    | yes      | Fine amount in cents (>0)|
| `reason`       | string | yes      | Reason for fine          |

---

### GET /api/workers/{id}/fines

List fines for a worker (paginated).

**Roles:** manager

**Query params:** `page`, `limit`

---

### POST /api/workers/{id}/debts

Create a debt (advance or loan) for a worker.

**Roles:** manager

**Body:**

```json
{
  "amount_cents": 1000000,
  "type": "advance",
  "note": "Salary advance for February"
}
```

| Field          | Type   | Required | Description                     |
| -------------- | ------ | -------- | ------------------------------- |
| `amount_cents` | int    | yes      | Debt amount in cents (> 0)      |
| `type`         | string | yes      | `advance` or `loan`             |
| `note`         | string | no       | Optional note                   |

When a debt is created, an **expense transaction** is automatically created in the finance module (cash payment type).

---

### GET /api/workers/{id}/debts

List debts for a worker (paginated).

**Roles:** manager

**Query params:** `page`, `limit`

**Response item fields:**

```json
{
  "id": 1,
  "worker_id": 5,
  "amount_cents": 1000000,
  "remaining_cents": 500000,
  "type": "advance",
  "note": "Salary advance",
  "status": "open",
  "created_by": 2,
  "created_at": "2026-02-01T00:00:00Z"
}
```

- `remaining_cents` = how much is still owed (decreases when payroll settles debts)

---

## 4. Payroll

Calculate and pay monthly salaries.

### GET /api/payroll

List payroll runs.

**Roles:** manager

**Query params:**

| Param       | Type   | Required | Description                 |
| ----------- | ------ | -------- | --------------------------- |
| `worker_id` | int    | no       | Filter by worker            |
| `period`    | string | no       | Filter by period (YYYY-MM)  |
| `page`      | int    | no       | Page number                 |
| `limit`     | int    | no       | Items per page              |

---

### POST /api/payroll/calculate

Calculate monthly payroll for a worker (without paying).

**Roles:** manager

**Body:**

```json
{
  "worker_id": 5,
  "period": "2026-02"
}
```

| Field       | Type   | Required | Description               |
| ----------- | ------ | -------- | ------------------------- |
| `worker_id` | int    | yes      | Worker ID (> 0)           |
| `period`    | string | yes      | Month in YYYY-MM format   |

**Calculation formula:**

```
net_salary = base_salary - fines - debts (minimum 0)
```

**Response:**

```json
{
  "data": {
    "id": 1,
    "worker_id": 5,
    "period": "2026-02",
    "base_salary_cents": 5000000,
    "fines_cents": 50000,
    "debts_cents": 1000000,
    "net_salary_cents": 3950000,
    "status": "calculated",
    "transaction_id": null,
    "created_at": "2026-02-25T10:00:00Z"
  }
}
```

**Errors:**

- 404 — worker compensation not configured
- 409 — payroll already exists for this period

---

### POST /api/payroll/{id}/pay

Pay a calculated payroll.

**Roles:** manager

**Body:**

```json
{
  "payment_type_code": "cash"
}
```

| Field                | Type   | Required | Description                        |
| -------------------- | ------ | -------- | ---------------------------------- |
| `payment_type_code`  | string | yes      | Payment type code (e.g. `cash`)    |

**What happens:**

1. Creates an expense transaction in finance
2. Creates a payment record
3. Updates payroll status to `paid`
4. Marks all open fines as `deducted`
5. Settles all open debts

**Errors:**

- 404 — payroll run not found
- 400 — payroll not in `calculated` status

---

## 5. Notifications (SMS)

Admin-only endpoints for monitoring SMS notifications.

### GET /api/notifications/sms

List SMS logs.

**Roles:** admin only

**Query params:**

| Param       | Type   | Required | Description                                                     |
| ----------- | ------ | -------- | --------------------------------------------------------------- |
| `status`    | string | no       | `queued`, `sending`, `sent`, `retrying`, `rate_limited`, `dlq`, `failed` |
| `type`      | string | no       | Notification type (e.g. `customer_order_created`)               |
| `phone`     | string | no       | Filter by phone number                                          |
| `job_id`    | string | no       | Filter by job ID                                                |
| `from_date` | string | no       | RFC3339 datetime                                                |
| `to_date`   | string | no       | RFC3339 datetime                                                |
| `page`      | int    | no       | Page number                                                     |
| `limit`     | int    | no       | Items per page                                                  |

---

### GET /api/notifications/sms/{job_id}

Get a single SMS log by job ID.

**Roles:** admin only

**Response:**

```json
{
  "data": {
    "id": 1,
    "job_id": "abc-123",
    "type": "customer_order_created",
    "to_phone": "+99365000000",
    "message": "Your order #5 has been created",
    "status": "sent",
    "attempt": 1,
    "max_attempts": 3,
    "provider": "twilio",
    "last_error": null,
    "created_at": "2026-02-25T10:00:00Z",
    "sent_at": "2026-02-25T10:00:05Z"
  }
}
```

**SMS status lifecycle:**

`queued` -> `sending` -> `sent` (success)

`queued` -> `sending` -> `retrying` -> `dlq` or `failed` (failure)

---

### POST /api/notifications/sms/{job_id}/requeue

Re-enqueue a failed SMS job for retry.

**Roles:** admin only

**Body:** none

Can only requeue from terminal statuses: `dlq`, `failed`, `rate_limited`

---

## General Notes

### Authentication

All requests need the header:

```
Authorization: Bearer <token>
```

### Roles

| Role     | Access level                           |
| -------- | -------------------------------------- |
| admin    | Full access + SMS notifications        |
| manager  | Finance, sales, payroll, worker finance|
| operator | Finance, sales, stock                  |
| cashier  | Sales, view transactions               |

### Money format

All money values are in **cents** (integer). For example:

- 500.00 TMT = `50000` cents
- 1.50 TMT = `150` cents

### Quantity format (qty_milli)

All quantities use **milli-units** (integer, SCALE = 1000):

- 1 piece = `1000`
- 2 pieces = `2000`
- 0.5 kg = `500`
- 1.75 kg = `1750`

Frontend display: `qty_milli / 1000` = human-readable number

### Pagination

Paginated endpoints return:

```json
{
  "data": [ ... ],
  "meta": {
    "pagination": {
      "page": 1,
      "limit": 10,
      "offset": 0,
      "total": 50,
      "has_next": true,
      "has_prev": false,
      "total_pages": 5
    }
  }
}
```

### Standard error response

```json
{
  "success": false,
  "status_code": 400,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "description of what went wrong"
  }
}
```
