# Price History — Frontend Guide

## Overview

Every time a product's `purchase_price` or `sale_price` changes, the system records:
- Which field changed (`purchase_price` or `sale_price`)
- Old value and new value (in cents)
- Who changed it (user ID + name)
- When it was changed

**Access:** manager and admin only.

---

## API

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/products/:id/price-history` | manager, admin | History of price changes |

---

## GET /api/products/:id/price-history

Returns paginated list of all price changes for a product, newest first.

**Query params:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `limit` | int | 10 | Items per page (max 100) |

**Request:**
```
GET /api/products/5/price-history?page=1&limit=20
Authorization: Bearer <token>
```

**Response:**
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 3,
        "product_id": 5,
        "field": "sale_price",
        "old_value": 15000,
        "new_value": 18000,
        "changed_by": 1,
        "changed_by_name": "Admin",
        "changed_at": "2026-04-04T14:30:00Z"
      },
      {
        "id": 2,
        "product_id": 5,
        "field": "purchase_price",
        "old_value": 10000,
        "new_value": 12000,
        "changed_by": 1,
        "changed_by_name": "Admin",
        "changed_at": "2026-04-03T09:15:00Z"
      },
      {
        "id": 1,
        "product_id": 5,
        "field": "sale_price",
        "old_value": 12000,
        "new_value": 15000,
        "changed_by": 3,
        "changed_by_name": "Mergen",
        "changed_at": "2026-04-01T11:00:00Z"
      }
    ],
    "page": 1,
    "limit": 20,
    "offset": 0,
    "total": 3
  }
}
```

---

## How It Works

Price history is recorded **automatically** when `PATCH /api/products/:id` changes `purchase_price` or `sale_price`.

```
PATCH /api/products/5
{ "sale_price": 18000 }
```

If the old `sale_price` was 15000 and the new is 18000 — a record is created:
```
field: "sale_price", old_value: 15000, new_value: 18000
```

If both prices change in the same request — **two records** are created (one per field).

If a price is sent but hasn't actually changed (same value) — **no record** is created.

---

## Field Reference

| Field | Type | Description |
|-------|------|-------------|
| `id` | int64 | Record ID |
| `product_id` | int64 | Product ID |
| `field` | string | `"purchase_price"` or `"sale_price"` |
| `old_value` | int64 | Previous price in cents |
| `new_value` | int64 | New price in cents |
| `changed_by` | int64 | User ID who made the change |
| `changed_by_name` | string | User's full name |
| `changed_at` | string | ISO 8601 timestamp |

**Money format:** all values in **cents** (tiin). 15000 = 150.00 TMT.

---

## Frontend Implementation

### 1. Check Role

Only show the price history button/tab for manager/admin:

```javascript
const userRoles = loginResponse.roles; // ["manager"] or ["admin"]
const canViewPriceHistory = userRoles.some(r => ['manager', 'admin'].includes(r));
```

### 2. Fetch Price History

```javascript
async function fetchPriceHistory(productId, page = 1, limit = 20) {
  const res = await api.get(
    `/api/products/${productId}/price-history?page=${page}&limit=${limit}`
  );
  return res.data; // { items, page, limit, offset, total }
}
```

### 3. Display

```javascript
function formatPrice(cents) {
  return (cents / 100).toFixed(2) + ' TMT';
}

function formatField(field) {
  return field === 'purchase_price' ? 'Закупочная цена' : 'Цена продажи';
}

function PriceChange({ item }) {
  const diff = item.new_value - item.old_value;
  const arrow = diff > 0 ? '↑' : '↓';
  const color = diff > 0 ? 'red' : 'green';

  return (
    <div>
      <span>{formatField(item.field)}</span>
      <span>{formatPrice(item.old_value)}</span>
      <span style={{ color }}>{arrow} {formatPrice(item.new_value)}</span>
      <span>{item.changed_by_name}</span>
      <span>{new Date(item.changed_at).toLocaleString()}</span>
    </div>
  );
}
```

---

## UI Mockup

### Product Card — Price History Tab

```
+----------------------------------------------------------+
|  Product: Red Bull                                       |
|  [Details]  [Stock]  [Price History]                     |
|----------------------------------------------------------|
|                                                          |
|  Date              Field          Old        New    Who  |
|  --------------------------------------------------------|
|  04.04.2026 14:30  Sale price    150.00 →  180.00  Admin |
|  03.04.2026 09:15  Purchase      100.00 →  120.00  Admin |
|  01.04.2026 11:00  Sale price    120.00 →  150.00  Mergen|
|  --------------------------------------------------------|
|                                                          |
|  Page 1 of 1          Total: 3 records                   |
+----------------------------------------------------------+
```

### Price Change Indicator

For `sale_price` increases, show red arrow up. For decreases, show green arrow down:

```
Sale price:    150.00 → 180.00 TMT  ↑ +30.00 TMT  (red)
Purchase:      120.00 → 100.00 TMT  ↓ -20.00 TMT  (green)
```

---

## Errors

| Code | HTTP | Description |
|------|------|-------------|
| `PRODUCT_NOT_FOUND` | 404 | Product with this ID doesn't exist |
| `UNAUTHORIZED` | 401 | Not authenticated |
| `FORBIDDEN` | 403 | User is not manager/admin |
