# Transfer Draft Sale -- Frontend Guide

## What is this?

A cashier may start a sale (add items to the cart) but need to hand it off to another cashier -- for example, during a shift change or when moving to a different register. The **Transfer Draft** feature lets the current owner send an in-progress (draft) sale to another cashier, who then sees it in their open drafts and can continue or complete it.

### Key rules:
- Only **draft** sales can be transferred. Completed or cancelled sales cannot.
- Only the **owner** of the draft (the cashier who created it) can transfer it. A **manager** or **admin** can also transfer any draft.
- After transfer, the `cashier_id` and `owner_user_id` on the sale both change to the new cashier.
- The transfer action requires the **"transfer" permission** on the "sales" module (see README_transfer_permission.md).

---

## API Endpoint

### Transfer a Draft Sale

```
POST /api/sales/:id/transfer
Content-Type: application/json
Authorization: Bearer <token>
```

**URL parameter:**

| Parameter | Type   | Description          |
|-----------|--------|----------------------|
| id        | number | The sale ID to transfer |

**Request body:**

```json
{
  "cashier_id": 5
}
```

| Field      | Type   | Required | Description                                        |
|------------|--------|----------|----------------------------------------------------|
| cashier_id | number | Yes      | User ID of the cashier to receive the draft (> 0)  |

---

### Response (200 OK):

```json
{
  "success": true,
  "data": {
    "transferred": true,
    "new_cashier_id": 5
  }
}
```

---

### Error Responses

**400 -- Sale is not a draft:**
```json
{
  "success": false,
  "error": "only draft sales can be transferred"
}
```

**403 -- Not the owner and not manager/admin:**
```json
{
  "success": false,
  "error": "only the owner or manager can transfer a draft"
}
```

**403 -- No transfer permission:**
```json
{
  "success": false,
  "error": "forbidden"
}
```

**404 -- Sale not found:**
```json
{
  "success": false,
  "error": "sale not found"
}
```

---

## Step-by-Step Flow

```
 Cashier A                    Server                    Cashier B
 ---------                    ------                    ---------
    |                            |                          |
    |  1. Creates a draft sale   |                          |
    |  POST /api/sales           |                          |
    |--------------------------->|                          |
    |  <-- sale { id: 101 }      |                          |
    |                            |                          |
    |  2. Adds items to draft    |                          |
    |  POST /api/sales/101/items |                          |
    |--------------------------->|                          |
    |                            |                          |
    |  3. Needs to hand off      |                          |
    |  Opens "Transfer" dialog   |                          |
    |  Selects Cashier B         |                          |
    |                            |                          |
    |  4. Transfers the draft    |                          |
    |  POST /api/sales/101/transfer                         |
    |  { "cashier_id": 5 }       |                          |
    |--------------------------->|                          |
    |  <-- { transferred: true } |                          |
    |                            |                          |
    |  5. Sale disappears from   |  6. Sale appears in      |
    |     Cashier A's drafts     |     Cashier B's drafts   |
    |                            |------------------------->|
    |                            |                          |
    |                            |  7. Cashier B continues  |
    |                            |     and completes sale   |
```

---

## UI Mockup: Transfer Dialog

### On the Draft Sale Screen

When the cashier has an open draft, show a **Transfer** button alongside other actions:

```
+------------------------------------------------------------------+
|  Sale #S-2026-0342 (DRAFT)                                        |
+------------------------------------------------------------------+
|                                                                    |
|  | #  | Product       | Qty | Price    | Total    |               |
|  |----|---------------|-----|----------|----------|               |
|  | 1  | Coca-Cola 1L  |  2  | 15.00    | 30.00    |               |
|  | 2  | Bread White   |  1  |  5.00    |  5.00    |               |
|  |----|---------------|-----|----------|----------|               |
|  |                         TOTAL:       35.00 TMT |               |
|  +------------------------------------------------+               |
|                                                                    |
|  [Cancel Sale]      [Transfer >>]          [Confirm & Pay]         |
+------------------------------------------------------------------+
```

### Transfer Dialog (Modal)

When the cashier clicks **[Transfer >>]**, show a modal:

```
+--------------------------------------------+
|  Transfer Draft Sale                    [X] |
|--------------------------------------------|
|                                             |
|  Select cashier to receive this sale:       |
|                                             |
|  +---------------------------------------+  |
|  | Search cashier...                     |  |
|  +---------------------------------------+  |
|                                             |
|  ( ) Aman K.    - Register 1               |
|  (*) Merdan A.  - Register 2               |
|  ( ) Bayram G.  - Register 3               |
|                                             |
|  -------------------------------------------+
|  Note: The selected cashier will see this   |
|  sale in their open drafts list.            |
|  -------------------------------------------+
|                                             |
|  [Cancel]                    [Transfer]     |
+--------------------------------------------+
```

### After Transfer (Success Toast)

```
+--------------------------------------------+
|  Sale transferred to Merdan A.         [OK] |
+--------------------------------------------+
```

---

## Frontend Implementation

### JavaScript Example

```javascript
const API_BASE = '/api';
const token = localStorage.getItem('token');

const headers = {
  'Authorization': `Bearer ${token}`,
  'Content-Type': 'application/json',
};

// Transfer a draft sale to another cashier
async function transferDraft(saleId, newCashierId) {
  const res = await fetch(`${API_BASE}/sales/${saleId}/transfer`, {
    method: 'POST',
    headers,
    body: JSON.stringify({ cashier_id: newCashierId }),
  });

  const json = await res.json();

  if (!res.ok) {
    // Show error to user
    if (res.status === 400) {
      alert(json.error); // "only draft sales can be transferred"
    } else if (res.status === 403) {
      alert(json.error); // "only the owner or manager can transfer a draft"
    } else if (res.status === 404) {
      alert('Sale not found');
    }
    return false;
  }

  // Success -- refresh the draft list
  console.log('Transferred to cashier:', json.data.new_cashier_id);
  return true;
}

// Example: wire up the transfer button
document.getElementById('transfer-btn').addEventListener('click', async () => {
  const saleId = getCurrentDraftSaleId();
  const cashierId = getSelectedCashierId(); // from the modal

  const ok = await transferDraft(saleId, cashierId);
  if (ok) {
    showToast('Sale transferred successfully');
    navigateToDraftList(); // sale is no longer yours
  }
});
```

### Checking Transfer Permission

Before showing the **[Transfer >>]** button, check if the user has the transfer permission. The login response includes the user's permission matrix:

```javascript
// After login, store permissions
const loginResponse = await login(username, password);
const permissions = loginResponse.data.permissions;
// permissions is an array like:
// [{ module: "sales", view: true, create: true, update: true, delete: false, transfer: true }]

// Check if user can transfer sales
function canTransferSales(permissions) {
  const salesPerm = permissions.find(p => p.module === 'sales');
  return salesPerm && salesPerm.transfer === true;
}

// Only show the transfer button if allowed
if (canTransferSales(permissions)) {
  document.getElementById('transfer-btn').style.display = 'block';
}
```

---

## Quick Reference

| Action         | Method | URL                          | Body                    |
|----------------|--------|------------------------------|-------------------------|
| Transfer draft | POST   | /api/sales/:id/transfer      | `{ "cashier_id": 5 }`  |

### Permission required:
- Module: `sales`
- Action: `transfer`
- The endpoint uses `RequirePermission("sales", "transfer")` middleware.

### Who can transfer:
| User role       | Own draft | Other's draft |
|-----------------|-----------|---------------|
| Cashier (owner) | Yes       | No            |
| Manager         | Yes       | Yes           |
| Admin           | Yes       | Yes           |
