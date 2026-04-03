# Sale Return + Barcode Scanner — Frontend Guide

---

## 1. Barcode Scanner (Product Lookup)

### Endpoint

```
GET /api/products/by-barcode/:code
Authorization: Bearer <token>
```

### How it works

1. Cashier scans barcode with scanner (scanner types barcode as keyboard input)
2. Frontend catches the input and calls the endpoint
3. Backend returns the product with price, name, photo
4. Frontend adds the product to the current sale

### Example

```
GET /api/products/by-barcode/8690504012347
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "id": 7,
    "name": "Red Bull Energy Drink 0.5L",
    "sku": "RB-05L-003",
    "barcodes": ["8690504012347"],
    "unit_type": "piece",
    "purchase_price": 120,
    "sale_price": 180,
    "is_active": true,
    "photo_url": "/uploads/products/7.jpg"
  }
}
```

**Error (404):**
```json
{
  "success": false,
  "error": {
    "code": "PRODUCT_NOT_FOUND",
    "message": "product not found for this barcode"
  }
}
```

### Frontend Implementation

```javascript
// Listen for barcode scanner input
let barcodeBuffer = '';
let barcodeTimer = null;

document.addEventListener('keypress', (e) => {
  // Scanner sends characters rapidly, then Enter
  if (e.key === 'Enter' && barcodeBuffer.length >= 4) {
    lookupBarcode(barcodeBuffer);
    barcodeBuffer = '';
    return;
  }

  barcodeBuffer += e.key;

  // Reset buffer after 100ms of no input (manual typing)
  clearTimeout(barcodeTimer);
  barcodeTimer = setTimeout(() => { barcodeBuffer = ''; }, 100);
});

async function lookupBarcode(code) {
  try {
    const { data } = await api.get(`/api/products/by-barcode/${code}`);
    addProductToSale(data);  // add to current sale
  } catch (err) {
    if (err.response?.status === 404) {
      showError('Product not found');
    }
  }
}
```

---

## 2. Sale Return (Partial + Full)

### Permission

Return is controlled by the `return` action in permissions.
- By default only **manager** and **admin** can return
- Admin can enable return for other roles:

```
PUT /api/permissions/2/sales/return
{ "granted": true }
```

Check permission in login response:
```json
{
  "permissions": [
    { "module": "sales", "view": true, "create": true, "return": false }
  ]
}
```

If `return: false` — hide the return button in UI.

---

### Endpoint

```
POST /api/sales/:id/return
Authorization: Bearer <token>
Content-Type: application/json
```

Works only for **confirmed** or **partially_returned** sales.

---

### Partial Return (return specific items)

Customer bought 3 products, wants to return 1:

```json
POST /api/sales/42/return
{
  "items": [
    { "sale_item_id": 105, "qty_milli": 1000 }
  ],
  "reason": "defective product"
}
```

- `sale_item_id` — ID of the sale item (from `GET /api/sales/:id` response)
- `qty_milli` — how much to return (1000 = 1 piece/kg)
- `reason` — optional, why returning

**Response (200):**
```json
{
  "success": true,
  "data": { "returned": true }
}
```

Sale status changes to `partially_returned`.

---

### Full Return (return everything)

Send empty `items` array or omit it:

```json
POST /api/sales/42/return
{
  "reason": "customer changed mind"
}
```

All items returned. Sale status changes to `returned`.

---

### Return by quantity (weight/volume)

Customer bought 2.5 kg of cheese, wants to return 1 kg:

```json
{
  "items": [
    { "sale_item_id": 106, "qty_milli": 1000 }
  ]
}
```

`qty_milli: 1000` = 1 kg. Cannot exceed original quantity minus already returned.

---

### Multiple returns on same sale

First return:
```json
POST /api/sales/42/return
{
  "items": [{ "sale_item_id": 105, "qty_milli": 1000 }]
}
```
Status: `partially_returned`

Second return (later):
```json
POST /api/sales/42/return
{
  "items": [{ "sale_item_id": 106, "qty_milli": 2000 }]
}
```
If all items now fully returned: status changes to `returned`.

---

### What happens on return

1. **Stock restored** — returned items go back to the warehouse
2. **Finance** — expense transaction created (refund to customer)
3. **Bonus** — if customer used bonus points, proportional amount restored
4. **Sale status** — `partially_returned` or `returned`

---

### Error responses

| Error | When |
|-------|------|
| `404 SALE_NOT_FOUND` | Sale doesn't exist |
| `409 SALE_NOT_RETURNABLE` | Sale is draft, cancelled, or already fully returned |
| `404 SALE_ITEM_NOT_FOUND` | sale_item_id doesn't belong to this sale |
| `400 VALIDATION` | Return qty exceeds remaining quantity |
| `409 NOTHING_TO_RETURN` | All items already returned (full return attempted) |
| `403 FORBIDDEN` | User doesn't have `sales.return` permission |

---

### Sale statuses after return

| Status | Meaning |
|--------|---------|
| `draft` | Not yet confirmed |
| `confirmed` | Confirmed, not returned |
| `partially_returned` | Some items returned |
| `returned` | All items fully returned |
| `cancelled` | Cancelled (different from return) |

---

## Frontend UI Flow

### Scanner Flow
```
+------------------------------------------+
|  POS Screen                              |
|                                          |
|  [Scan barcode or type product name]     |
|                                          |
|  Cart:                                   |
|  1. Red Bull 0.5L    x1    180 TMT      |
|  2. Bread             x2    10 TMT      |
|                                          |
|  Total: 200 TMT                          |
|                                          |
|  [Confirm]  [Cancel]                     |
+------------------------------------------+
```

### Return Flow
```
+------------------------------------------+
|  Sale #42 — Confirmed                    |
|                                          |
|  Items:                                  |
|  [x] Red Bull 0.5L    x1    180 TMT     |
|  [ ] Bread             x2     10 TMT    |
|  [ ] Cheese 2.5kg      x1    250 TMT    |
|                                          |
|  Reason: [defective product        ]    |
|                                          |
|  [Cancel]        [Return Selected]       |
+------------------------------------------+
```

1. User opens a confirmed sale
2. Selects items to return (checkboxes)
3. For weight/volume items — enters return quantity
4. Enters reason (optional)
5. Clicks "Return Selected"
6. Frontend sends `POST /api/sales/42/return` with selected items
7. On success — refresh sale details, status updated
