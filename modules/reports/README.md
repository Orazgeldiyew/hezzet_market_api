# Reports Module

Provides analytics and Excel exports for sales, stock, and dashboard KPIs. All data is derived from existing tables — no new migrations required.

---

## Table of Contents

- [Overview](#overview)
- [Module Structure](#module-structure)
- [API](#api)
  - [Dashboard](#get-apireportsdashboard)
  - [Sales by Period](#get-apireportssales)
  - [Sales by Product](#get-apireportssalesproducts)
  - [Export Sales Excel](#get-apireportssalesexport)
  - [Export Stock Excel](#get-apireportsstockexport)
- [Excel Exports](#excel-exports)
- [Integration](#integration)

---

## Overview

The reports module exposes read-only aggregation endpoints. All routes are under `/api/reports` and require the `manager` or `admin` role.

No new database tables are created. All queries run against existing tables:

| Query | Tables used |
|---|---|
| Dashboard | `sales`, `sale_items`, `products`, `warehouse_items`, `payroll_runs` |
| Sales by period | `sales`, `customers` |
| Sales by product | `sales`, `sale_items`, `products` |
| Stock export | `warehouse_item_details`, `warehouses`, `products`, `users` |

---

## Module Structure

```
modules/reports/
├── model.go       — DashboardStats, TopProduct, SalesPeriodRow, SalesProductRow, StockMovementExportRow
├── repository.go  — all SQL queries (Dashboard, SalesByPeriod, SalesByProduct, StockMovementsForExport)
├── handler.go     — HTTP handlers + Excel builders (buildSalesExcel, buildStockExcel)
└── routes.go      — route registration
```

---

## API

All endpoints require `Authorization: Bearer <token>` with `manager` or `admin` role.

### `GET /api/reports/dashboard`

Returns today's key metrics and the week's top products.

**Response `200`:**
```json
{
  "data": {
    "today_revenue_cents": 1250000,
    "today_profit_cents":  310000,
    "today_orders":        42,
    "low_stock_count":     5,
    "pending_payroll":     3,
    "top_products": [
      { "product_id": 12, "name": "Premium Tea", "revenue_cents": 450000 }
    ]
  }
}
```

| Field | Description |
|---|---|
| `today_revenue_cents` | Sum of `sales.total_cents` since midnight local time |
| `today_profit_cents` | Revenue minus cost (`total_cents - cost_cents`) |
| `today_orders` | Count of sales rows since midnight |
| `low_stock_count` | Products where `qty_milli < LOW_STOCK_DEFAULT * 1000` across all warehouses |
| `pending_payroll` | Payroll runs with `status = 'calculated'` (not yet paid) |
| `top_products` | Top 5 products by revenue since Monday of the current week |

---

### `GET /api/reports/sales`

Aggregated sales grouped by time period.

**Query parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| `group_by` | string | `day` | Grouping unit: `day` \| `week` \| `month` |
| `from` | RFC3339 | Start of current month | Range start |
| `to` | RFC3339 | Now | Range end |
| `warehouse_id` | int | — | Filter to one warehouse |
| `customer_type` | string | — | Filter by customer tier: `regular` \| `vip` \| `wholesale` |

**Response `200`:**
```json
{
  "data": [
    {
      "period":        "2025-06-01T00:00:00Z",
      "orders":        18,
      "revenue_cents": 890000,
      "cost_cents":    620000,
      "profit_cents":  270000
    }
  ]
}
```

> Note: when `customer_type` is provided, sales with no linked customer are excluded.

---

### `GET /api/reports/sales/products`

Per-product breakdown with margin percentage.

**Query parameters:** `from`, `to`, `warehouse_id` (same as above).

**Response `200`:**
```json
{
  "data": [
    {
      "product_id":    42,
      "name":          "Green Tea 500g",
      "qty_milli":     15000,
      "revenue_cents": 450000,
      "cost_cents":    310000,
      "profit_cents":  140000,
      "margin_pct":    31.11
    }
  ]
}
```

Results are ordered by `revenue_cents DESC`.

---

### `GET /api/reports/sales/export`

Downloads an `.xlsx` file with two sheets.

**Query parameters:** same as `/api/reports/sales` (`group_by`, `from`, `to`, `warehouse_id`, `customer_type`).

**Response:** `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
Filename: `sales_YYYY-MM-DD_YYYY-MM-DD.xlsx`

**Sheet layout:**

*Sheet "By Period"*

| Period | Orders | Revenue (TMT) | Cost (TMT) | Profit (TMT) | Margin % |
|---|---|---|---|---|---|

*Sheet "By Product"*

| Product | Qty | Revenue (TMT) | Cost (TMT) | Profit (TMT) | Margin % |
|---|---|---|---|---|---|

---

### `GET /api/reports/stock/export`

Downloads an `.xlsx` file with stock ledger movements.

**Query parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| `from` | RFC3339 | Start of current month | Range start |
| `to` | RFC3339 | Now | Range end |
| `warehouse_id` | int | — | Filter to one warehouse |

**Response:** `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
Filename: `stock_YYYY-MM-DD_YYYY-MM-DD.xlsx`
Capped at **10 000 rows** ordered by `created_at DESC`.

*Sheet "Stock Movements"*

| Date | Warehouse | Product | Delta (qty) | Type | Unit Price (TMT) | Created By |
|---|---|---|---|---|---|---|

Rows with a negative delta (outgoing stock) are highlighted red.

**Movement types:** `in`, `out`, `transfer_in`, `transfer_out`, `damaged`, `adjustment`, `opening_balance`, `sale`

---

## Excel Exports

Both exports share the same styling as the payroll export:

- **Header row:** dark (`#1F2937`) background, white bold Arial 12, centered, height 22
- **Data rows:** alternating white / light gray (`#F2F2F2`) zebra striping, height 18
- **Money columns:** custom format `#,##0.00 TMT`
- **Freeze:** header row frozen (row 1), AutoFilter enabled
- **Thin borders:** `#D9D9D9` on all cells

---

## Integration

Wired in `server/router.go` after all other modules:

```go
reports.RegisterRoutes(api, deps.DB, deps.Cfg.LowStockDefault)
```

`LowStockDefault` comes from the `LOW_STOCK_DEFAULT` env var (default: `10` regular units).
The repository multiplies by 1000 internally to convert to milli-units for the `warehouse_items` comparison.
