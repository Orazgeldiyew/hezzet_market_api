# Notification Phones — Admin Phone Management

Allows admins to manage the list of phone numbers that receive low-stock and other admin SMS notifications through the API, without requiring a server restart or config change.

---

## Table of Contents

- [Overview](#overview)
- [Module Structure](#module-structure)
- [Database](#database)
- [API](#api)
  - [List phones](#get-apinotificationsphones)
  - [Add phone](#post-apinotificationsphones)
  - [Update phone](#put-apinotificationsphonesid)
  - [Delete phone](#delete-apinotificationsphonesid)
- [Integration with Service](#integration-with-service)

---

## Overview

Admin phone numbers are stored in the `notification_phones` table. At runtime, `Service.NotifyAdminLowStock` queries the DB for enabled phones instead of using the static config list. The config list (`ADMIN_PHONES` env var) acts as a fallback if the DB returns no results.

All routes are under `/api/notifications/phones` and require the `admin` role.

---

## Module Structure

```
modules/notification/
├── phone_model.go       — NotificationPhone, AddPhoneRequest, UpdatePhoneRequest
├── phone_repository.go  — PhoneRepository (List, Add, Update, Delete, GetActivePhones)
├── phone_handler.go     — PhoneHandler (List, Add, Update, Delete)
├── routes.go            — route registration (phones + SMS logs)
└── service.go           — PhoneQuerier interface, used at low-stock notify time
```

---

## Database

Migration: `034_notification_phones`

```sql
CREATE TABLE notification_phones (
    id         BIGSERIAL PRIMARY KEY,
    phone      TEXT NOT NULL UNIQUE,
    label      TEXT NOT NULL DEFAULT '',
    enabled    BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

| Column | Description |
|---|---|
| `phone` | E.164 format recommended (`+99361XXXXXX`). Must be unique. |
| `label` | Human-readable name for the number (e.g. "Manager Ahmet") |
| `enabled` | When `false`, the number is excluded from notifications without deleting it |
| `created_at` | Set automatically on insert |

---

## API

All endpoints require `Authorization: Bearer <token>` with role `admin`.

---

### `GET /api/notifications/phones`

Returns all phones (enabled and disabled).

**Response `200`:**
```json
{
  "data": [
    {
      "id":         1,
      "phone":      "+99361234567",
      "label":      "Manager Ahmet",
      "enabled":    true,
      "created_at": "2025-06-01T10:00:00Z"
    }
  ]
}
```

---

### `POST /api/notifications/phones`

Adds a new phone number.

**Request body:**
```json
{
  "phone": "+99361234567",
  "label": "Manager Ahmet"
}
```

| Field | Required | Description |
|---|---|---|
| `phone` | Yes | Phone number (must be unique) |
| `label` | No | Human-readable description |

**Response `201`:**
```json
{
  "data": {
    "id":         2,
    "phone":      "+99361234567",
    "label":      "Manager Ahmet",
    "enabled":    true,
    "created_at": "2025-06-15T09:00:00Z"
  }
}
```

**Errors:**
- `400 VALIDATION_ERROR` — `phone` field missing
- `500 INTERNAL_ERROR` — duplicate phone (unique constraint violation)

---

### `PUT /api/notifications/phones/:id`

Updates `label` and/or `enabled` flag. Only provided fields are updated (PATCH semantics via `COALESCE`).

**Request body:**
```json
{
  "label":   "Head of Sales",
  "enabled": false
}
```

Both fields are optional — send only what needs to change.

**Response `200`:**
```json
{
  "data": {
    "id":         2,
    "phone":      "+99361234567",
    "label":      "Head of Sales",
    "enabled":    false,
    "created_at": "2025-06-15T09:00:00Z"
  }
}
```

**Errors:**
- `400 VALIDATION_ERROR` — invalid `id` or malformed body
- `404 PHONE_NOT_FOUND` — no phone with that ID

---

### `DELETE /api/notifications/phones/:id`

Permanently removes the phone.

**Response `200`:**
```json
{
  "data": { "deleted": true }
}
```

**Errors:**
- `400 VALIDATION_ERROR` — invalid `id`
- `404 PHONE_NOT_FOUND` — no phone with that ID

---

## Integration with Service

`Service.NotifyAdminLowStock` resolves target phones at call time:

```
1. Query notification_phones WHERE enabled = true  ← DB phones
2. If result is empty → fall back to ADMIN_PHONES env var
3. Enqueue one SMS job per phone
```

The `PhoneRepository` implements the `PhoneQuerier` interface so it can be injected into `Service` without a hard dependency on the concrete type:

```go
type PhoneQuerier interface {
    GetActivePhones(ctx context.Context) ([]string, error)
}
```

`PhoneRepository` is created once in `main.go` and shared between the `Service` (for runtime phone lookup) and the `PhoneHandler` (for admin CRUD).
