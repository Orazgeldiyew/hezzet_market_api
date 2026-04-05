# Discount System — Frontend Guide

## Overview

Two levels of discount:
1. **Per-item discount** — on individual products (at sale creation)
2. **Per-sale discount** — on the entire sale (at confirmation)

Both levels work together. Controlled by `sales.discount` permission — admin/manager can enable/disable per role.

---

## Permission Check

After login, check `permissions` in response:

```json
{
  "permissions": [
    { "module": "sales", "discount": true }
  ]
}
```

- `discount: true` — show discount input fields
- `discount: false` — hide discount inputs, user cannot apply discounts

Admin/manager toggles:
```
PUT /api/permissions/:role_id/sales/discount
{ "granted": false }
```

---

## API

### 1. Create Sale with Per-Item Discount

```
POST /api/sales
```

```json
{
  "warehouse_id": 1,
  "items": [
    { "product_id": 1, "qty_milli": 1000, "discount_percent": 15 },
    { "product_id": 2, "qty_milli": 2000, "discount_percent": 0 },
    { "product_id": 3, "qty_milli": 1000 }
  ]
}
```

- `discount_percent` — integer 0-100, optional (default 0)
- Applied per item: `line_total = (qty × price) - discount%`

**Response:**
```json
{
  "data": {
    "sale": {
      "id": 42,
      "total_cents": 21500,
      "discount_percent": 0,
      "discount_cents": 0
    },
    "items": [
      {
        "id": 10,
        "product_id": 1,
        "unit_price_cents": 10000,
        "line_total_cents": 8500,
        "discount_percent": 15
      },
      {
        "id": 11,
        "product_id": 2,
        "unit_price_cents": 5000,
        "line_total_cents": 10000,
        "discount_percent": 0
      },
      {
        "id": 12,
        "product_id": 3,
        "unit_price_cents": 3000,
        "line_total_cents": 3000,
        "discount_percent": 0
      }
    ]
  }
}
```

---

### 2. Confirm Sale with Per-Sale Discount

```
POST /api/sales/:id/confirm
```

```json
{
  "discount_percent": 5,
  "payment_type_id": 1,
  "payment_amount": 20425
}
```

- `discount_percent` — integer 0-100, optional (default 0)
- Applied on total after item discounts

**Calculation:**
```
Item 1: 100.00 TMT × 1  = 100.00  → -15% → 85.00 TMT
Item 2:  50.00 TMT × 2  = 100.00  →  0%  → 100.00 TMT
Item 3:  30.00 TMT × 1  =  30.00  →  0%  → 30.00 TMT
                          Subtotal:         215.00 TMT
                     Sale discount 5%:      -10.75 TMT
                              TOTAL:        204.25 TMT
```

**Response:**
```json
{
  "data": {
    "sale": {
      "id": 42,
      "status": "confirmed",
      "total_cents": 20425,
      "discount_percent": 5,
      "discount_cents": 1075
    }
  }
}
```

---

### 3. No Discount (default behavior)

If `discount_percent` is 0 or not provided — no discount applied. Everything works as before.

```json
POST /api/sales
{ "warehouse_id": 1, "items": [{ "product_id": 1, "qty_milli": 1000 }] }

POST /api/sales/42/confirm
{ "payment_type_id": 1 }
```

---

## Receipt

The receipt automatically shows discounts:

```
+------------------------------------------+
|          HEZZET MARKET                   |
|     Ashgabat, Oguzhan koc. 15           |
+------------------------------------------+
| Check #00042                             |
| Date: 03.04.2026 14:30                  |
| Cashier: admin                           |
+------------------------------------------+
| Product         Qty    Price     Total   |
|------------------------------------------|
| Red Bull (-15%) 1    180.00    153.00    |
| Bread           2     50.00    100.00    |
| Cheese          1     30.00     30.00    |
|------------------------------------------|
| Discount 5%:                   -14.15   |
| TOTAL:                         268.85   |
+------------------------------------------+
```

- Per-item discount shows next to product name: `(-15%)`
- Per-sale discount shows as separate line before TOTAL

---

## Frontend Implementation

### 1. Check Permission

```javascript
const canDiscount = loginResponse.permissions
  .find(p => p.module === 'sales')?.discount ?? false;
```

### 2. Per-Item Discount Input

```javascript
// Only show if canDiscount is true
{canDiscount && (
  <input
    type="number"
    min="0"
    max="100"
    placeholder="Discount %"
    value={item.discount_percent || 0}
    onChange={(e) => setItemDiscount(item.id, parseInt(e.target.value))}
  />
)}
```

### 3. Per-Sale Discount Input (at confirm)

```javascript
// Confirm dialog
{canDiscount && (
  <div>
    <label>Sale Discount %</label>
    <input
      type="number"
      min="0"
      max="100"
      value={saleDiscountPercent}
      onChange={(e) => setSaleDiscountPercent(parseInt(e.target.value))}
    />
  </div>
)}
```

### 4. Calculate Preview

```javascript
function calculateTotal(items, saleDiscountPercent = 0) {
  // Step 1: item totals with per-item discounts
  let subtotal = 0;
  items.forEach(item => {
    let lineTotal = Math.round((item.qty_milli * item.unit_price_cents) / 1000);
    if (item.discount_percent > 0) {
      lineTotal = lineTotal - Math.floor((lineTotal * item.discount_percent) / 100);
    }
    subtotal += lineTotal;
  });

  // Step 2: per-sale discount
  let discountCents = 0;
  if (saleDiscountPercent > 0) {
    discountCents = Math.floor((subtotal * saleDiscountPercent) / 100);
  }

  return {
    subtotal,
    discountCents,
    total: subtotal - discountCents
  };
}
```

### 5. Submit

```javascript
// Create sale with item discounts
const createBody = {
  warehouse_id: warehouseId,
  items: cartItems.map(item => ({
    product_id: item.product_id,
    qty_milli: item.qty_milli,
    discount_percent: item.discount_percent || 0
  }))
};
const sale = await api.post('/api/sales', createBody);

// Confirm with sale discount
const confirmBody = {
  payment_type_id: paymentTypeId,
  payment_amount: totalAfterDiscount,
  discount_percent: saleDiscountPercent || 0
};
await api.post(`/api/sales/${sale.data.sale.id}/confirm`, confirmBody);
```

---

## UI Mockup

### POS Screen with Discounts

```
+-----------------------------------------------------+
|  POS                                                |
|-----------------------------------------------------|
|  Product        Qty   Price   Disc%   Total         |
|  Red Bull        1   180.00   [15]   153.00 TMT    |
|  Bread           2    50.00   [ 0]   100.00 TMT    |
|  Cheese          1    30.00   [ 0]    30.00 TMT    |
|-----------------------------------------------------|
|  Subtotal:                           283.00 TMT    |
|  Sale Discount:                [5] % -14.15 TMT    |
|  Bonus Used:                          -0.00 TMT    |
|-----------------------------------------------------|
|  TOTAL:                              268.85 TMT    |
|                                                     |
|  [Cancel]                         [Confirm Sale]    |
+-----------------------------------------------------+
```

- Disc% column only visible if user has `sales.discount` permission
- Sale Discount row only visible if user has `sales.discount` permission
- Both inputs accept 0-100 integer

---

## Quick Reference

| Field | Where | Type | Range | Default |
|-------|-------|------|-------|---------|
| `items[].discount_percent` | POST /api/sales | int | 0-100 | 0 |
| `discount_percent` | POST /api/sales/:id/confirm | int | 0-100 | 0 |
| `sale.discount_percent` | Response | int | 0-100 | — |
| `sale.discount_cents` | Response | int64 | >= 0 | — |
| `items[].discount_percent` | Response | int | 0-100 | — |

**Permission action:** `sales.discount`
**Admin manages:** `PUT /api/permissions/:role_id/sales/discount { "granted": true/false }`
