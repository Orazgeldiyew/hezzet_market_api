# Cash Register & Shifts — Frontend Guide

## Overview

Cashier opens a shift at the start of the day, works, closes at the end. Manager sees full report (Z-report) with all financial details. Cashier sees only basic info — no amounts.

---

## API Endpoints

| Method | Path | Role | Description |
|--------|------|------|-------------|
| GET | `/api/registers` | cashier+ | List cash registers |
| POST | `/api/shifts/open` | cashier+ | Open shift |
| GET | `/api/shifts/current` | cashier+ | Current open shift (minimal) |
| POST | `/api/shifts/:id/close` | cashier+ | Close shift |
| GET | `/api/shifts` | manager | List all shifts |
| GET | `/api/shifts/:id` | manager | Z-report (full details) |

---

## Cashier Flow

### 1. List Cash Registers

```
GET /api/registers
```

```json
{
  "data": [
    { "id": 1, "name": "Касса 1", "is_active": true },
    { "id": 2, "name": "Касса 2", "is_active": true }
  ]
}
```

### 2. Open Shift

```
POST /api/shifts/open
```

```json
{ "register_id": 1, "opening_cash": 50000 }
```

- `register_id` — which register (from step 1)
- `opening_cash` — how much cash is in the register at start (in cents, 50000 = 500.00 TMT)

**Response (201):**
```json
{
  "data": {
    "id": 5,
    "register_id": 1,
    "user_id": 4,
    "opened_at": "2026-04-03T09:00:00Z",
    "opening_cash": 50000,
    "status": "open"
  }
}
```

**Error — already has open shift:**
```json
{ "error": { "code": "SHIFT_ALREADY_OPEN", "message": "you already have an open shift" } }
```

### 3. Check Current Shift

```
GET /api/shifts/current
```

**Response (cashier sees only basic info):**
```json
{
  "data": {
    "id": 5,
    "register_id": 1,
    "register_name": "Касса 1",
    "user_id": 4,
    "cashier_name": "Mergen",
    "opened_at": "2026-04-03T09:00:00Z",
    "status": "open"
  }
}
```

No amounts, no sales count — cashier doesn't see financial data.

**Error — no open shift:**
```json
{ "error": { "code": "NO_OPEN_SHIFT", "message": "no open shift found" } }
```

### 4. Close Shift

```
POST /api/shifts/5/close
```

```json
{ "closing_cash": 345000, "note": "all good" }
```

- `closing_cash` — cashier counts cash in register and enters amount (cents)
- `note` — optional comment

**Response:**
```json
{ "data": { "closed": true } }
```

Cashier does NOT see the difference or totals. Manager sees everything in Z-report.

---

## Manager Flow

### 5. List All Shifts

```
GET /api/shifts?status=closed&limit=10
```

**Query params:**
- `user_id` — filter by cashier
- `register_id` — filter by register
- `status` — `open` or `closed`
- `page`, `limit` — pagination

**Response:**
```json
{
  "data": [
    {
      "id": 5,
      "register_name": "Касса 1",
      "cashier_name": "Mergen",
      "opened_at": "2026-04-03T09:00:00Z",
      "closed_at": "2026-04-03T21:00:00Z",
      "opening_cash": 50000,
      "closing_cash": 345000,
      "expected_cash": 345000,
      "difference": 0,
      "sales_count": 45,
      "sales_total": 320000,
      "returns_total": 15000,
      "status": "closed"
    }
  ]
}
```

### 6. Z-Report (Shift Details)

```
GET /api/shifts/5
```

**Response (full financial details):**
```json
{
  "data": {
    "id": 5,
    "register_id": 1,
    "register_name": "Касса 1",
    "user_id": 4,
    "cashier_name": "Mergen",
    "opened_at": "2026-04-03T09:00:00Z",
    "closed_at": "2026-04-03T21:00:00Z",
    "opening_cash": 50000,
    "closing_cash": 345000,
    "expected_cash": 345000,
    "difference": 0,
    "sales_count": 45,
    "sales_total": 320000,
    "returns_total": 15000,
    "status": "closed",
    "note": "all good"
  }
}
```

**Fields explained:**
- `opening_cash` — cash at shift start
- `closing_cash` — cash counted by cashier at end
- `expected_cash` — `opening_cash + sales_total - returns_total`
- `difference` — `closing_cash - expected_cash` (0 = perfect, negative = shortage, positive = surplus)
- `sales_count` — number of confirmed sales
- `sales_total` — total revenue (cents)
- `returns_total` — total refunds (cents)

---

## Frontend Implementation

### Cashier UI

```javascript
// On app load — check if shift is open
async function checkShift() {
  try {
    const { data } = await api.get('/api/shifts/current');
    setShift(data);       // shift is open, show POS
  } catch (err) {
    if (err.response?.data?.error?.code === 'NO_OPEN_SHIFT') {
      showOpenShiftDialog();  // no shift, show "Open Shift" screen
    }
  }
}

// Open shift
async function openShift(registerId, openingCash) {
  const { data } = await api.post('/api/shifts/open', {
    register_id: registerId,
    opening_cash: Math.round(openingCash * 100)  // TMT to cents
  });
  setShift(data);
}

// Close shift
async function closeShift(shiftId, closingCash, note) {
  await api.post(`/api/shifts/${shiftId}/close`, {
    closing_cash: Math.round(closingCash * 100),
    note: note || undefined
  });
  setShift(null);
  showOpenShiftDialog();
}
```

### Cashier UI Mockup

```
+------------------------------------------+
|  OPEN SHIFT                              |
|                                          |
|  Cash Register: [Касса 1  v]            |
|                                          |
|  Opening Cash:  [  500.00  ] TMT        |
|                                          |
|          [Open Shift]                    |
+------------------------------------------+
```

After opening:
```
+------------------------------------------+
|  Shift #5 — Касса 1                     |
|  Opened: 03.04.2026 09:00               |
|  Status: Open                            |
|                                          |
|  ... POS screen ...                      |
|                                          |
|          [Close Shift]                   |
+------------------------------------------+
```

Close dialog:
```
+------------------------------------------+
|  CLOSE SHIFT                             |
|                                          |
|  Count the cash in your register:        |
|                                          |
|  Closing Cash:  [ 3450.00  ] TMT        |
|  Note:          [ all good         ]     |
|                                          |
|  [Cancel]              [Close Shift]     |
+------------------------------------------+
```

### Manager UI Mockup — Z-Report

```
+--------------------------------------------------+
|  Z-REPORT — Shift #5                             |
+--------------------------------------------------+
|  Register:    Касса 1                             |
|  Cashier:     Mergen                              |
|  Opened:      03.04.2026 09:00                   |
|  Closed:      03.04.2026 21:00                   |
+--------------------------------------------------+
|  Opening Cash:       500.00 TMT                  |
|  Sales (45):       3,200.00 TMT                  |
|  Returns:           -150.00 TMT                  |
+--------------------------------------------------+
|  Expected Cash:    3,550.00 TMT                  |
|  Actual Cash:      3,450.00 TMT                  |
|  Difference:        -100.00 TMT  (SHORTAGE)      |
+--------------------------------------------------+
|  Note: all good                                  |
+--------------------------------------------------+
```

- Green if difference = 0
- Red if difference < 0 (shortage)
- Yellow if difference > 0 (surplus)

---

## Rules

- Cashier cannot open 2 shifts at the same time
- Manager/admin can close anyone's shift
- Sales are NOT linked to shifts (yet) — system calculates totals by user + time range
- All money values in cents (50000 = 500.00 TMT)
