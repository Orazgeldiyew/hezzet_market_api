# Force Sale (Deficit Selling)

## Overview

Allows cashiers to sell products even when stock is insufficient.
By default, the system rejects sales that exceed available stock.
With `force: true`, the sale goes through and stock goes negative.

---

## API

### Endpoint

```
POST /api/sales
```

### Headers

```
Authorization: Bearer <token>
Content-Type: application/json
```

---

## Normal Flow (without force)

### Request

```json
{
  "warehouse_id": 1,
  "items": [
    { "product_id": 1, "qty_milli": 15000 }
  ]
}
```

### Response — Error (stock = 10, requested = 15)

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION",
    "message": "insufficient available stock for product 1: have 10000 milli, reserved 0 milli, need 15000 milli (use force=true to sell in deficit)"
  }
}
```

---

## Force Flow (deficit sale)

### Step 1 — First request (without force)

```json
{
  "warehouse_id": 1,
  "items": [
    { "product_id": 1, "qty_milli": 15000 }
  ]
}
```

Backend returns `VALIDATION` error with insufficient stock message.

### Step 2 — Show confirmation dialog to cashier

```
+-------------------------------------+
|  Warning: Insufficient stock         |
|                                      |
|  Product: Bread                      |
|  Available: 10 pcs                   |
|  Requested: 15 pcs                   |
|  Deficit: -5 pcs                     |
|                                      |
|  Sell in deficit?                    |
|                                      |
|  [Cancel]          [Yes, sell]       |
+-------------------------------------+
```

### Step 3 — If cashier confirms, resend with force: true

```json
{
  "force": true,
  "warehouse_id": 1,
  "items": [
    { "product_id": 1, "qty_milli": 15000 }
  ]
}
```

### Response — Success

```json
{
  "success": true,
  "data": {
    "id": 12,
    "status": "draft",
    "total_cents": 750000,
    "items": [
      {
        "product_id": 1,
        "qty_milli": 15000,
        "unit_price_cents": 50000,
        "line_total_cents": 750000
      }
    ]
  }
}
```

Stock after sale: `10 - 15 = -5` (negative)

---

## Frontend Logic (pseudocode)

```javascript
async function createSale(saleData) {
  // 1. Try normal sale
  const response = await api.post('/api/sales', saleData);

  if (response.error?.code === 'VALIDATION' && response.error.message.includes('insufficient')) {
    // 2. Show confirmation dialog
    const confirmed = await showDialog({
      title: 'Insufficient stock',
      message: 'Sell in deficit?',
      buttons: ['Cancel', 'Yes, sell']
    });

    if (confirmed) {
      // 3. Resend with force
      return await api.post('/api/sales', { ...saleData, force: true });
    }
  }

  return response;
}
```

---

## Notes

- `force: true` works for all roles (cashier, operator, manager, admin)
- Stock can go negative — manager should monitor low stock via reports
- `qty_milli` uses milli-units: 1000 = 1 piece/kg. So 15000 = 15 pcs/kg
- `force` only affects stock validation, all other validations still apply (product exists, is active, etc.)
