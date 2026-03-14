# Reports Module — Dashboard Analytics

Provides KPI dashboard with period-over-period comparison plus sales and stock reports with Excel export.

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
- [Data Model](#data-model)
- [Excel Exports](#excel-exports)

---

## Overview

All routes are under `/api/reports` and require the `manager` or `admin` role.

No migrations required — all queries run against existing tables:

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
├── model.go       — DashboardStats, PeriodStats, TopProduct, SalesPeriodRow, SalesProductRow, StockMovementExportRow
├── repository.go  — SQL queries (Dashboard, SalesByPeriod, SalesByProduct, StockMovementsForExport)
├── handler.go     — HTTP handlers + Excel builders
└── routes.go      — route registration
```

---

## API

All endpoints require `Authorization: Bearer <token>` with role `manager` or `admin`.

---

### `GET /api/reports/dashboard`

Returns sales KPIs for three periods (today / this week / this month), each with revenue change % vs the previous same period, plus low-stock count, pending payroll, and top-5 products.

**Response `200`:**
```json
{
  "data": {
    "today": {
      "revenue_cents": 1250000,
      "profit_cents":  310000,
      "orders":        42,
      "change_pct":    12.5
    },
    "week": {
      "revenue_cents": 6800000,
      "profit_cents":  1700000,
      "orders":        210,
      "change_pct":    -4.3
    },
    "month": {
      "revenue_cents": 28000000,
      "profit_cents":  7200000,
      "orders":        860,
      "change_pct":    8.0
    },
    "low_stock_count": 5,
    "pending_payroll": 3,
    "top_products": [
      { "product_id": 12, "name": "Premium Tea", "revenue_cents": 450000 }
    ]
  }
}
```

**Period comparison logic:**

| Field | Current period | Compared to |
|---|---|---|
| `today` | From midnight today | Same window yesterday |
| `week` | From Monday this week | Same Mon–now window last week |
| `month` | From 1st of this month | Same day range last month |

**`change_pct` formula:**
```
change_pct = (current - previous) / previous * 100
```
- Returns `+100` when previous period had 0 revenue but current > 0
- Returns `0` when both periods are zero

**Other fields:**

| Field | Description |
|---|---|
| `low_stock_count` | Active products where `qty_milli < LOW_STOCK_DEFAULT * 1000` across all warehouses |
| `pending_payroll` | Payroll runs with `status = 'calculated'` (not yet paid) |
| `top_products` | Top 5 products by revenue since Monday of the current week |

---

### `GET /api/reports/sales`

Aggregated sales grouped by time period.

**Query parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| `group_by` | string | `day` | `day` \| `week` \| `month` |
| `from` | RFC3339 | Start of current month | Range start (inclusive) |
| `to` | RFC3339 | Now | Range end (inclusive) |
| `warehouse_id` | int | — | Filter to one warehouse |
| `customer_type` | string | — | `regular` \| `vip` \| `wholesale` |

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

> When `customer_type` is set, sales with no linked customer are excluded.

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

Ordered by `revenue_cents DESC`.

---

### `GET /api/reports/sales/export`

Downloads `.xlsx` with two sheets: **By Period** and **By Product**.

**Query parameters:** same as `/api/reports/sales`.

**Response:** `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
Filename: `sales_YYYY-MM-DD_YYYY-MM-DD.xlsx`

| Sheet | Columns |
|---|---|
| By Period | Period, Orders, Revenue (TMT), Cost (TMT), Profit (TMT), Margin % |
| By Product | Product, Qty, Revenue (TMT), Cost (TMT), Profit (TMT), Margin % |

---

### `GET /api/reports/stock/export`

Downloads `.xlsx` with stock ledger movements. Capped at **10 000 rows**.

**Query parameters:** `from`, `to`, `warehouse_id`.

**Response:** `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
Filename: `stock_YYYY-MM-DD_YYYY-MM-DD.xlsx`

| Column | Description |
|---|---|
| Date | `created_at` of the movement |
| Warehouse | Warehouse name |
| Product | Product name |
| Delta (qty) | Change in quantity (negative = outgoing, highlighted red) |
| Type | `in` / `out` / `transfer_in` / `transfer_out` / `damaged` / `adjustment` / `opening_balance` / `sale` |
| Unit Price (TMT) | `price_cents / 100` |
| Created By | Username of the operator |

---

## Data Model

```go
type PeriodStats struct {
    RevenueCents int64   `json:"revenue_cents"`
    ProfitCents  int64   `json:"profit_cents"`
    Orders       int     `json:"orders"`
    ChangePct    float64 `json:"change_pct"`
}

type DashboardStats struct {
    Today          PeriodStats  `json:"today"`
    Week           PeriodStats  `json:"week"`
    Month          PeriodStats  `json:"month"`
    LowStockCount  int          `json:"low_stock_count"`
    PendingPayroll int          `json:"pending_payroll"`
    TopProducts    []TopProduct `json:"top_products"`
}
```

---

## Excel Exports

Both exports share consistent styling:

| Element | Style |
|---|---|
| Header row | Dark (`#1F2937`) background, white bold Arial 12, centered, height 22 |
| Data rows | Alternating white / light gray (`#F2F2F2`) zebra striping, height 18 |
| Money columns | Custom format `#,##0.00 TMT` |
| Borders | Thin `#D9D9D9` on all cells |
| Freeze | Row 1 frozen, AutoFilter enabled |
| Negative delta | Red fill on stock rows with `delta_milli < 0` |
