# Database Migrations

All migrations are plain SQL files under `migrations/`. They are numbered sequentially and applied in order.

---

## Convention

| File pattern | Purpose |
|---|---|
| `NNN_name.up.sql` | Apply the migration |
| `NNN_name.down.sql` | Roll back the migration |

Migrations are run manually or via the migration tool configured in `pkg/database/`. Never edit an already-applied migration — create a new one instead.

---

## Migration Index

| # | Name | Description |
|---|---|---|
| 001–033 | *(legacy)* | Initial schema, products, stock, sales, payroll, permissions, etc. |
| 034 | `notification_phones` | Admin phone number management table |
| 035 | `performance_indexes` | Composite indexes for common query patterns |

---

## 034 — notification_phones

**File:** `034_notification_phones.up.sql`

```sql
CREATE TABLE notification_phones (
    id         BIGSERIAL PRIMARY KEY,
    phone      TEXT NOT NULL UNIQUE,
    label      TEXT NOT NULL DEFAULT '',
    enabled    BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**Purpose:** Stores admin phone numbers for SMS notifications. Replaces the static `ADMIN_PHONES` env var as the primary source, with the env var remaining as a fallback when the table is empty.

**Rollback:** `DROP TABLE IF EXISTS notification_phones;`

**Used by:** `modules/notification/phone_repository.go`, `modules/notification/service.go`

---

## 035 — performance_indexes

**File:** `035_performance_indexes.up.sql`

Six partial and composite indexes targeting the most frequent query patterns across reporting and list endpoints.

| Index | Table | Columns | Condition | Use case |
|---|---|---|---|---|
| `idx_sales_created_by` | `sales` | `created_by` | `created_by IS NOT NULL` | Cashier / operator reports filtered by user |
| `idx_sales_status_created_at` | `sales` | `status, created_at DESC` | — | Sales list ordered by time with status filter |
| `idx_sales_warehouse_status` | `sales` | `warehouse_id, status` | `warehouse_id IS NOT NULL` | Sales list filtered by warehouse + status |
| `idx_purchase_orders_warehouse_status` | `purchase_orders` | `warehouse_id, status` | `warehouse_id IS NOT NULL` | Purchase order list by warehouse |
| `idx_wid_warehouse_type_time` | `warehouse_item_details` | `warehouse_id, type, created_at DESC` | — | Stock movement report with warehouse + type filter |
| `idx_transactions_created_by` | `transactions` | `created_by` | `created_by IS NOT NULL` | Finance reports per user |

**Rollback:** All indexes are dropped with `DROP INDEX CONCURRENTLY IF EXISTS`.

> Partial indexes (with `WHERE` clause) are smaller and faster than full indexes — rows with `NULL` in the filtered column are excluded from the index entirely.
