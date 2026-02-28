# Purchase Orders Module

Manages the full lifecycle of purchasing goods from suppliers: from creating a draft order, receiving stock, to tracking outstanding supplier debt and recording payments.

---

## How It Works

### Two-Step Lifecycle

```
[Create Draft PO]  →  [Receive PO]  →  [Add Payments]
      draft              received         pending → partial → paid
        ↓
   [Cancel PO]
    cancelled
```

**Step 1 — Create (draft)**
- Records what products are being ordered, at what agreed unit cost, and from which supplier/warehouse.
- No stock is added yet. No financial transaction is created.
- Status: `draft`

**Step 2 — Receive**
- Transitions the PO from `draft` → `received`.
- Atomically:
  1. Adds stock to `warehouse_item_details` (ledger entry, type `purchase`)
  2. Upserts `warehouse_items` — increments `qty_milli`, recalculates `avg_cost_cents` using weighted average
  3. Creates an `expense` finance transaction (`status = pending`) = supplier debt
- Status: `received`

**Cancel**
- Only `draft` POs can be cancelled. Received POs cannot be undone via this endpoint.
- Status: `cancelled`

### Supplier Debt & Payments

When a PO is received, an `expense` transaction is created with `status = pending`. This represents money owed to the supplier.

Payments are recorded via `POST /:id/payments`:
- Each payment reduces the outstanding balance.
- Transaction status progresses: `pending` → `partial` → `paid`
- Overpayment is rejected (validated against remaining balance).

`GET /debt` aggregates all suppliers with outstanding unpaid balances, showing `total_cents`, `paid_cents`, and `debt_cents`.

---

## API Endpoints

| Method | Path | Roles | Description |
|--------|------|-------|-------------|
| `POST` | `/api/purchases` | operator, manager | Create a draft purchase order |
| `GET` | `/api/purchases` | operator, manager | List POs with filters & pagination |
| `GET` | `/api/purchases/debt` | operator, manager | Supplier debt summary (unpaid only) |
| `GET` | `/api/purchases/:id` | operator, manager | Get PO detail with items & transaction ID |
| `POST` | `/api/purchases/:id/receive` | operator, manager | Receive goods — adds stock + creates debt |
| `POST` | `/api/purchases/:id/cancel` | operator, manager | Cancel a draft PO |
| `POST` | `/api/purchases/:id/payments` | operator, manager | Record a supplier payment |

### Create PO — Request Body

```json
{
  "supplier_id": 1,
  "warehouse_id": 2,
  "note": "optional note",
  "items": [
    {
      "product_id": 10,
      "qty_milli": 5000,
      "unit_cost_cents": 1200
    }
  ]
}
```

- `qty_milli` — quantity in milliunit (1000 = 1 unit)
- `unit_cost_cents` — agreed cost per unit (per 1000 milli)

### Add Payment — Request Body

```json
{
  "payment_type_id": 1,
  "amount_cents": 50000,
  "note": "partial payment"
}
```

### List POs — Query Parameters

| Param | Type | Description |
|-------|------|-------------|
| `supplier_id` | int | Filter by supplier |
| `warehouse_id` | int | Filter by warehouse |
| `status` | string | `draft` / `received` / `cancelled` |
| `date_from` | RFC3339 | Inclusive start datetime |
| `date_to` | RFC3339 | Inclusive end datetime |
| `page` | int | Page number (default: 1) |
| `limit` | int | Results per page (default: 20) |

---

## Database Tables

### `purchase_orders`

| Column | Type | Description |
|--------|------|-------------|
| `id` | bigserial | Primary key |
| `supplier_id` | bigint | FK → suppliers |
| `warehouse_id` | bigint | FK → warehouses |
| `status` | text | `draft` / `received` / `cancelled` |
| `total_cents` | bigint | Sum of all line totals |
| `items_count` | int | Number of line items |
| `note` | text | Optional note |
| `created_by` | bigint | FK → users |
| `received_at` | timestamptz | Set on receive |
| `received_by` | bigint | FK → users, set on receive |

### `purchase_order_items`

| Column | Type | Description |
|--------|------|-------------|
| `po_id` | bigint | FK → purchase_orders (CASCADE DELETE) |
| `product_id` | bigint | FK → products |
| `qty_milli` | bigint | Quantity in milliunit |
| `unit_cost_cents` | bigint | Agreed cost per unit |
| `line_total_cents` | bigint | `qty_milli * unit_cost_cents / 1000` |

---

## Cost Calculation

**Line total:** `line_total_cents = qty_milli * unit_cost_cents / 1000`

**Weighted average cost** (on receive):
```
new_avg = (old_total_cost + new_cost) * 1000 / (old_qty + new_qty)
```
This recalculates `avg_cost_cents` on `warehouse_items` each time stock arrives.

---

## Audit Log Events

| Action | Trigger |
|--------|---------|
| `CREATE` | `POST /api/purchases` |
| `PO_RECEIVE` | `POST /api/purchases/:id/receive` |
| `PO_CANCEL` | `POST /api/purchases/:id/cancel` |
| `PO_PAYMENT` | `POST /api/purchases/:id/payments` |

---

## Module Structure

```
modules/purchase/
├── model.go       — structs: PurchaseOrder, PurchaseItem, PODetail, SupplierDebtRow, request types
├── repository.go  — all DB logic (transactions, stock upsert, debt queries)
├── service.go     — thin layer: duplicate check, error wrapping
├── handler.go     — HTTP handlers with Swagger annotations
└── routes.go      — route registration under /api/purchases
```
