# Product API Documentation

Base URL: `/api`

All requests must include:
```
Authorization: Bearer <token>
Content-Type: application/json
```

---

## Roles

| Role     | Read products | Create / Update / Delete |
| -------- | ------------- | ------------------------ |
| admin    | yes           | yes                      |
| manager  | yes           | no                       |
| operator | yes           | yes                      |
| cashier  | yes           | no                       |

---

## Endpoints

### 1. Create Product

```
POST /api/products
```

**Roles:** operator

**Request body:**

```json
{
  "name": "Coca-Cola 0.5L",
  "sku": "COCA-05",
  "unit": "pcs",
  "unit_type": "piece",
  "purchase_price": 5000,
  "sale_price": 8000,
  "is_active": true,
  "barcodes": ["4607038328394", "1234567890123"],
  "category_ids": [1, 3]
}
```

| Field            | Type     | Required | Description                                        |
| ---------------- | -------- | -------- | -------------------------------------------------- |
| `name`           | string   | yes      | Product name                                       |
| `sku`            | string   | no       | Unique product code (Stock Keeping Unit)           |
| `unit`           | string   | yes      | Display unit label — e.g. `"pcs"`, `"kg"`, `"L"` |
| `unit_type`      | string   | no       | One of: `piece`, `kg`, `liter`, `meter`, `box`. Default: `piece` |
| `purchase_price` | int      | no       | Cost price in cents. Default: 0                    |
| `sale_price`     | int      | no       | Selling price in cents. Default: 0                 |
| `is_active`      | bool     | no       | Default: `true`                                    |
| `barcodes`       | string[] | no       | List of barcode strings                            |
| `category_ids`   | int[]    | no       | Category IDs to assign on creation                 |

**Response `201`:**

```json
{
  "success": true,
  "status_code": 201,
  "data": {
    "id": 12,
    "name": "Coca-Cola 0.5L",
    "sku": "COCA-05",
    "unit": "pcs",
    "unit_type": "piece",
    "unit_scale": 1000,
    "purchase_price": 5000,
    "sale_price": 8000,
    "is_active": true,
    "barcodes": ["4607038328394"],
    "created_at": "2026-02-25T10:00:00Z",
    "updated_at": "2026-02-25T10:00:00Z"
  }
}
```

**Errors:**

| Status | Code                     | When                                 |
| ------ | ------------------------ | ------------------------------------ |
| 400    | `VALIDATION_ERROR`       | Missing required fields              |
| 409    | `PRODUCT_ALREADY_EXISTS` | SKU or barcode already used          |

---

### 2. List Products

```
GET /api/products
```

**Roles:** operator, cashier, manager

**Query params:**

| Param             | Type   | Default      | Description                              |
| ----------------- | ------ | ------------ | ---------------------------------------- |
| `search`          | string | —            | Search by name, SKU, or barcode          |
| `page`            | int    | 1            | Page number                              |
| `limit`           | int    | 10           | Items per page (max 200)                 |
| `order_by`        | string | `created_at` | Sort field: `name` or `created_at`       |
| `order_direction` | string | `desc`       | Sort direction: `asc` or `desc`          |

**Response `200`:**

```json
{
  "success": true,
  "status_code": 200,
  "data": [
    {
      "id": 12,
      "name": "Coca-Cola 0.5L",
      "sku": "COCA-05",
      "unit": "pcs",
      "unit_type": "piece",
      "unit_scale": 1000,
      "purchase_price": 5000,
      "sale_price": 8000,
      "is_active": true,
      "barcodes": ["4607038328394"],
      "created_at": "2026-02-25T10:00:00Z",
      "updated_at": "2026-02-25T10:00:00Z"
    }
  ],
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

**Notes:**

- Only active products are returned (`is_active = true`)
- Search works on name, SKU, and all barcodes simultaneously

---

### 3. Get Product by ID

```
GET /api/products/{id}
```

**Roles:** operator, cashier, manager

**URL params:** `id` — product ID (integer)

**Response `200`:** same shape as a single item from the list above.

**Errors:**

| Status | Code                | When                                    |
| ------ | ------------------- | --------------------------------------- |
| 404    | `PRODUCT_NOT_FOUND` | Product not found or is deleted         |

---

### 4. Update Product

```
PATCH /api/products/{id}
```

**Roles:** operator

All fields are optional — only send the ones you want to change.

**Request body:**

```json
{
  "name": "Coca-Cola 1L",
  "sku": "COCA-1L",
  "unit": "pcs",
  "unit_type": "piece",
  "purchase_price": 8000,
  "sale_price": 12000,
  "is_active": true,
  "barcodes": ["9876543210987"],
  "category_ids": [2]
}
```

| Field            | Type     | Description                                                     |
| ---------------- | -------- | --------------------------------------------------------------- |
| `name`           | string   | New name                                                        |
| `sku`            | string   | New SKU                                                         |
| `unit`           | string   | New display unit                                                |
| `unit_type`      | string   | `piece`, `kg`, `liter`, `meter`, or `box`                      |
| `purchase_price` | int      | New cost price in cents                                         |
| `sale_price`     | int      | New sale price in cents                                         |
| `is_active`      | bool     | Activate or deactivate                                          |
| `barcodes`       | string[] | **Replaces all barcodes** (send full new list)                  |
| `category_ids`   | int[]    | **Replaces all categories** (send full new list, or `[]` to clear) |

**Response `200`:** updated product object.

**Errors:**

| Status | Code                     | When                            |
| ------ | ------------------------ | ------------------------------- |
| 404    | `PRODUCT_NOT_FOUND`      | Product not found               |
| 409    | `PRODUCT_ALREADY_EXISTS` | SKU or barcode already used     |

---

### 5. Delete Product

```
DELETE /api/products/{id}
```

**Roles:** operator

Soft delete — the product is marked inactive, not permanently removed.

**Response `200`:**

```json
{
  "success": true,
  "status_code": 200,
  "data": { "deleted": true }
}
```

**Errors:**

| Status | Code                | When                  |
| ------ | ------------------- | --------------------- |
| 404    | `PRODUCT_NOT_FOUND` | Product not found     |

---

### 6. Get Product Card

```
GET /api/products/{id}/card
```

**Roles:** operator, cashier, manager

Returns full product info + total stock + categories. Use this for a product detail page.

**Response `200`:**

```json
{
  "success": true,
  "status_code": 200,
  "data": {
    "product": {
      "id": 12,
      "name": "Coca-Cola 0.5L",
      "sku": "COCA-05",
      "unit": "pcs",
      "unit_type": "piece",
      "unit_scale": 1000,
      "purchase_price": 5000,
      "sale_price": 8000,
      "is_active": true,
      "barcodes": ["4607038328394"],
      "created_at": "2026-02-25T10:00:00Z",
      "updated_at": "2026-02-25T10:00:00Z"
    },
    "stock": 5000,
    "categories": [
      { "id": 1, "name": "Beverages" }
    ],
    "tags": [],
    "supplier_ids": []
  }
}
```

- `stock` — total quantity in milli-units across all warehouses. Divide by 1000 to display: `5000 / 1000 = 5 pcs`

---

### 7. Get Product Categories

```
GET /api/products/{id}/categories
```

**Roles:** operator, cashier, manager

Returns the list of categories this product belongs to.

**Response `200`:**

```json
{
  "success": true,
  "status_code": 200,
  "data": [
    { "id": 1, "name": "Beverages" },
    { "id": 3, "name": "Cold drinks" }
  ]
}
```

---

### 8. Set Product Categories (replace all)

```
PUT /api/products/{id}/categories
```

**Roles:** operator

**Replaces all current categories** with the ones you send. To clear all categories, send an empty array.

**Request body:**

```json
{
  "category_ids": [1, 3]
}
```

**Response `200`:** array of assigned categories (same shape as endpoint 7).

**Errors:**

| Status | Code               | When                                  |
| ------ | ------------------ | ------------------------------------- |
| 400    | `VALIDATION_ERROR` | Some category IDs not found or inactive |

---

### 9. Remove One Category from Product

```
DELETE /api/products/{id}/categories/{categoryId}
```

**Roles:** operator

Removes a single category link from a product.

**URL params:**
- `id` — product ID
- `categoryId` — category ID to remove

**Response `200`:**

```json
{
  "success": true,
  "status_code": 200,
  "data": { "deleted": true }
}
```

**Errors:**

| Status | Code                | When                              |
| ------ | ------------------- | --------------------------------- |
| 404    | `PRODUCT_NOT_FOUND` | Product not found                 |
| 404    | `NOT_FOUND`         | Category not linked to product    |

---

## Field Reference

### Money (prices)

All prices are in **cents** (integer). Divide by 100 to show human-readable TMT:

- `5000` cents = 50.00 TMT
- `8000` cents = 80.00 TMT

### Quantity (stock)

Stock is in **milli-units** (integer). Divide by 1000 to display:

- `1000` = 1 piece
- `2500` = 2.5 kg
- `500` = 0.5 liter

### unit_type values

| Value    | Description          |
| -------- | -------------------- |
| `piece`  | Counted item (pcs)   |
| `kg`     | Weight in kilograms  |
| `liter`  | Volume in liters     |
| `meter`  | Length in meters     |
| `box`    | Box / carton         |

### unit_scale

Always `1000`. It means: 1 display unit = 1000 milli-units. Used for quantity math.

---

## Standard Error Response

All errors follow this format:

```json
{
  "success": false,
  "status_code": 404,
  "error": {
    "code": "PRODUCT_NOT_FOUND",
    "message": "product not found"
  }
}
```

---

## Quick Examples

**Search for a product by barcode:**
```
GET /api/products?search=4607038328394
```

**Get page 2 of products sorted by name A→Z:**
```
GET /api/products?page=2&limit=20&order_by=name&order_direction=asc
```

**Create a product and assign categories at once:**
```json
POST /api/products
{
  "name": "Sugar 1kg",
  "unit": "kg",
  "unit_type": "kg",
  "sale_price": 6000,
  "purchase_price": 4000,
  "barcodes": ["1234567890"],
  "category_ids": [5]
}
```

**Update only the sale price (partial update):**
```json
PATCH /api/products/12
{
  "sale_price": 9000
}
```
