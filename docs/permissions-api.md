# Permissions API — Документация для фронтенда

## Концепция

Система permissions позволяет **включать/выключать доступ к модулям** для каждой роли.

- `admin` — всегда имеет доступ ко всему, permissions на него не влияют
- `manager` — может менять permissions для `cashier` и `operator`
- `admin` — может менять permissions для всех ролей включая `manager`

### Модули
| Модуль | Что включает |
|--------|-------------|
| `sales` | Продажи `/api/sales/*` |
| `stock` | Склад `/api/stock/*`, склады `/api/warehouses/*`, поставщики `/api/suppliers/*` |
| `products` | Товары `/api/products/*`, категории `/api/categories/*` |
| `customers` | Клиенты `/api/customers/*` |
| `workers` | Сотрудники `/api/workers/*`, зарплаты `/api/payroll/*` |
| `finance` | Финансы `/api/transactions/*`, типы оплат `/api/payment-types/*` |
| `purchases` | Закупки `/api/purchases/*` |
| `reports` | Отчёты `/api/reports/*`, аудит `/api/audit-logs/*` |

---

## Endpoints

### GET /api/permissions
Получить список всех permissions.

**Требования:** роль `manager` или `admin`

**Headers:**
```
Authorization: Bearer <token>
```

**Response 200:**
```json
{
  "success": true,
  "data": [
    { "id": 1, "role": "cashier",  "module": "sales",     "enabled": true,  "updated_at": "2026-03-12T10:00:00Z" },
    { "id": 2, "role": "cashier",  "module": "stock",     "enabled": true,  "updated_at": "2026-03-12T10:00:00Z" },
    { "id": 3, "role": "cashier",  "module": "reports",   "enabled": false, "updated_at": "2026-03-12T10:00:00Z" },
    { "id": 4, "role": "cashier",  "module": "purchases", "enabled": false, "updated_at": "2026-03-12T10:00:00Z" },
    { "id": 5, "role": "operator", "module": "sales",     "enabled": true,  "updated_at": "2026-03-12T10:00:00Z" },
    { "id": 6, "role": "manager",  "module": "sales",     "enabled": true,  "updated_at": "2026-03-12T10:00:00Z" }
    // ... остальные
  ]
}
```

---

### PUT /api/permissions/{role}/{module}
Включить или выключить модуль для роли.

**Требования:**
- `manager` — может менять только `cashier` и `operator`
- `admin` — может менять любую роль включая `manager`

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Path params:**
- `role` — `cashier` | `operator` | `manager`
- `module` — `sales` | `stock` | `products` | `customers` | `workers` | `finance` | `purchases` | `reports`

**Body:**
```json
{ "enabled": false }
```

**Response 200:**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "role": "cashier",
    "module": "sales",
    "enabled": false,
    "updated_at": "2026-03-12T10:05:00Z"
  }
}
```

**Response 403 (manager пытается менять manager):**
```json
{
  "success": false,
  "error": "only admin can change manager permissions"
}
```

---

## Пример UI — таблица permissions

Рекомендуемый вид: **таблица с чекбоксами / тогглами**

```
               SALES   STOCK   PRODUCTS   CUSTOMERS   FINANCE   PURCHASES   WORKERS   REPORTS
cashier          ✅      ✅       ✅          ✅          ✅         ❌          ❌        ❌
operator         ✅      ✅       ✅          ✅          ✅         ✅          ✅        ❌
manager          ✅      ✅       ✅          ✅          ✅         ✅          ✅        ✅
```

- Строки = роли
- Колонки = модули
- Ячейка = toggle/switch (enabled / disabled)
- Строка `manager` — disabled для менеджера (только admin может редактировать)

---

## Пример кода (fetch)

### Загрузить все permissions
```js
const res = await fetch('/api/permissions', {
  headers: { Authorization: `Bearer ${token}` }
})
const { data } = await res.json()

// Преобразовать в удобную структуру для таблицы:
// { cashier: { sales: true, stock: true, ... }, operator: {...}, manager: {...} }
const matrix = {}
for (const p of data) {
  if (!matrix[p.role]) matrix[p.role] = {}
  matrix[p.role][p.module] = p.enabled
}
```

### Переключить permission
```js
async function togglePermission(role, module, enabled) {
  const res = await fetch(`/api/permissions/${role}/${module}`, {
    method: 'PUT',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ enabled }),
  })

  if (!res.ok) {
    const err = await res.json()
    alert(err.error) // например: "only admin can change manager permissions"
    return false
  }
  return true
}
```

---

## Важные моменты для UI

1. **Изменения применяются мгновенно** — Redis кэш сбрасывается при каждом PUT запросе. Не нужен перезапуск сервера.

2. **Строка manager** — если текущий пользователь `manager` (не `admin`), нужно отображать строку `manager` как **read-only** (disabled toggles).

3. **Определить текущую роль** — из JWT токена (после decode) поле `roles: ["manager"]`.

4. **Оптимистичный UI** — можно сразу переключить тоггл визуально, и откатить если придёт ошибка.

5. **Нет отдельного "сохранить"** — каждый тоггл делает отдельный PUT запрос сразу при изменении.

---

## Роли и доступ к странице permissions

| Роль | Может открыть страницу | Может редактировать |
|------|----------------------|---------------------|
| `cashier` | ❌ | ❌ |
| `operator` | ❌ | ❌ |
| `manager` | ✅ | Только cashier и operator |
| `admin` | ✅ | Все роли |
