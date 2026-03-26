# Favorite Products (Hot Keys) -- Frontend Guide

## What is this?

Each cashier can save their most-sold products as **favorites**. These show up as quick-access buttons on the POS (Point of Sale) screen so the cashier does not have to search for them every time.

- Each user has their own list of favorites (not shared between users).
- Favorites have a **position** field so the cashier can arrange the buttons in any order.
- The API returns the product name, price, and photo so you can render the buttons immediately without extra API calls.

---

## Data Model

Each favorite item looks like this:

```json
{
  "id": 1,
  "user_id": 12,
  "product_id": 42,
  "position": 0,
  "created_at": "2026-03-20T10:30:00Z",
  "name": "Coca-Cola 1L",
  "price_cents": 1500,
  "photo_url": "https://cdn.example.com/products/coca-cola.jpg"
}
```

| Field        | Type    | Description                                      |
|--------------|---------|--------------------------------------------------|
| id           | number  | Internal favorite record ID                      |
| user_id      | number  | The user who owns this favorite                  |
| product_id   | number  | The product this favorite points to              |
| position     | number  | Sort order (0 = first, 1 = second, ...)          |
| created_at   | string  | When it was added                                |
| name         | string  | Product name (for display)                       |
| price_cents  | number  | Product price in cents (divide by 100 for TMT)   |
| photo_url    | string or null | Product image URL (can be null)            |

---

## API Endpoints

All endpoints require authentication (Bearer token in the `Authorization` header).
Base URL: `/api/favorites`

### 1. List Favorites

Get the current user's favorite products, sorted by position.

```
GET /api/favorites
```

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "user_id": 12,
      "product_id": 42,
      "position": 0,
      "created_at": "2026-03-20T10:30:00Z",
      "name": "Coca-Cola 1L",
      "price_cents": 1500,
      "photo_url": "https://cdn.example.com/products/coca-cola.jpg"
    },
    {
      "id": 2,
      "user_id": 12,
      "product_id": 77,
      "position": 1,
      "created_at": "2026-03-20T10:31:00Z",
      "name": "Bread White",
      "price_cents": 500,
      "photo_url": null
    }
  ]
}
```

---

### 2. Add a Product to Favorites

```
POST /api/favorites
Content-Type: application/json
```

**Request body:**
```json
{
  "product_id": 42
}
```

| Field      | Type   | Required | Description                |
|------------|--------|----------|----------------------------|
| product_id | number | Yes      | Must be greater than 0     |

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "user_id": 12,
    "product_id": 42,
    "position": 0,
    "created_at": "2026-03-20T10:30:00Z",
    "name": "Coca-Cola 1L",
    "price_cents": 1500,
    "photo_url": "https://cdn.example.com/products/coca-cola.jpg"
  }
}
```

**Error (409 Conflict):** Product is already in favorites.
```json
{
  "success": false,
  "error": "product already in favorites"
}
```

---

### 3. Remove a Product from Favorites

```
DELETE /api/favorites/:product_id
```

**Example:** `DELETE /api/favorites/42`

**Response (200):**
```json
{
  "success": true,
  "data": {
    "deleted": true
  }
}
```

**Error (404):** Product is not in favorites.

---

### 4. Reorder Favorites

Change the position of favorite buttons. Send the full list with new positions.

```
PUT /api/favorites/reorder
Content-Type: application/json
```

**Request body:**
```json
{
  "items": [
    { "product_id": 77, "position": 0 },
    { "product_id": 42, "position": 1 }
  ]
}
```

| Field              | Type   | Required | Description                 |
|--------------------|--------|----------|-----------------------------|
| items              | array  | Yes      | At least 1 item             |
| items[].product_id | number | Yes      | Must be greater than 0      |
| items[].position   | number | Yes      | 0 or higher (0 = first)     |

**Response (200):**
```json
{
  "success": true,
  "data": {
    "reordered": true
  }
}
```

---

## UI Mockup: POS Screen with Favorite Buttons

```
+------------------------------------------------------------------+
|  HEZZET MARKET - POS                          Cashier: Merdan     |
+------------------------------------------------------------------+
|                                                                    |
|  [Search product...]                              [Barcode Scan]   |
|                                                                    |
|  --- FAVORITES (Hot Keys) --------------------------------------- |
|  +-------------+  +-------------+  +-------------+  +-----------+ |
|  | [img]       |  | [img]       |  | [img]       |  | [+] Add   | |
|  | Coca-Cola   |  | Bread White |  | Milk 1L     |  | Favorite  | |
|  | 15.00 TMT   |  | 5.00 TMT   |  | 12.00 TMT   |  |           | |
|  +-------------+  +-------------+  +-------------+  +-----------+ |
|                                                                    |
|  --- CART -------------------------------------------------------- |
|  | #  | Product       | Qty | Price    | Total    |               |
|  |----|---------------|-----|----------|----------|               |
|  | 1  | Coca-Cola 1L  |  2  | 15.00    | 30.00    |               |
|  | 2  | Bread White   |  1  |  5.00    |  5.00    |               |
|  |----|---------------|-----|----------|----------|               |
|  |                         TOTAL:       35.00 TMT |               |
|  +------------------------------------------------+               |
|                                                                    |
|  [Cancel]                                        [Confirm Sale]    |
+------------------------------------------------------------------+
```

### How the favorites bar works:
1. Favorite buttons appear as a horizontal row (or grid) above the cart.
2. Tapping a favorite button **adds 1 unit of that product to the cart** (same as scanning its barcode).
3. The `[+] Add Favorite` button opens a product search dialog. When the cashier picks a product, call `POST /api/favorites`.
4. Long-press (or right-click) on a favorite opens a context menu: **Remove** or **Reorder**.
5. For drag-and-drop reorder, after the user finishes dragging, call `PUT /api/favorites/reorder` with the new positions.

---

## JavaScript Example (Frontend)

```javascript
const API_BASE = '/api';
const token = localStorage.getItem('token');

const headers = {
  'Authorization': `Bearer ${token}`,
  'Content-Type': 'application/json',
};

// 1. Load favorites when POS screen opens
async function loadFavorites() {
  const res = await fetch(`${API_BASE}/favorites`, { headers });
  const json = await res.json();
  return json.data; // array of favorite products
}

// 2. Add a product to favorites
async function addFavorite(productId) {
  const res = await fetch(`${API_BASE}/favorites`, {
    method: 'POST',
    headers,
    body: JSON.stringify({ product_id: productId }),
  });
  if (res.status === 409) {
    alert('This product is already in your favorites.');
    return null;
  }
  const json = await res.json();
  return json.data; // the new favorite object
}

// 3. Remove a product from favorites
async function removeFavorite(productId) {
  await fetch(`${API_BASE}/favorites/${productId}`, {
    method: 'DELETE',
    headers,
  });
}

// 4. Reorder favorites after drag-and-drop
async function reorderFavorites(items) {
  // items = [{ product_id: 77, position: 0 }, { product_id: 42, position: 1 }]
  await fetch(`${API_BASE}/favorites/reorder`, {
    method: 'PUT',
    headers,
    body: JSON.stringify({ items }),
  });
}

// 5. Render favorite buttons
function renderFavorites(favorites) {
  const container = document.getElementById('favorites-bar');
  container.innerHTML = '';

  favorites.forEach(fav => {
    const btn = document.createElement('button');
    btn.className = 'favorite-btn';
    btn.innerHTML = `
      ${fav.photo_url ? `<img src="${fav.photo_url}" alt="${fav.name}" />` : '<div class="no-photo"></div>'}
      <span class="name">${fav.name}</span>
      <span class="price">${(fav.price_cents / 100).toFixed(2)} TMT</span>
    `;
    btn.onclick = () => addToCart(fav.product_id);
    container.appendChild(btn);
  });
}
```

---

## Quick Reference

| Action           | Method | URL                          | Body                                  |
|------------------|--------|------------------------------|---------------------------------------|
| List favorites   | GET    | /api/favorites               | --                                    |
| Add favorite     | POST   | /api/favorites               | `{ "product_id": 42 }`               |
| Remove favorite  | DELETE | /api/favorites/:product_id   | --                                    |
| Reorder          | PUT    | /api/favorites/reorder       | `{ "items": [{ "product_id": 42, "position": 0 }] }` |
