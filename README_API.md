# Hezzet Market Backend — API Reference (Frontend Guide)

## Базовая информация

- **Base URL:** `http://localhost:8080`
- **API prefix:** `/api`
- **Авторизация:** JWT Bearer Token — `Authorization: Bearer <access_token>`
- **Content-Type:** `application/json`

---

## Аутентификация

### POST `/auth/login` (публичный)
```json
// Request
{ "username": "admin", "password": "123456" }

// Response
{
  "access_token": "eyJhbG...",
  "refresh_token": "eyJhbG...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "user": { "id": 1, "username": "admin", "full_name": "Admin", ... },
  "roles": ["admin"]
}
```

### POST `/auth/refresh` (публичный)
```json
{ "refresh_token": "eyJhbG..." }
```

### POST `/auth/register` (публичный)
```json
{
  "username": "cashier1",
  "password": "123456",
  "full_name": "Кассир 1",
  "phone": "65012345",
  "email": "cashier@example.com"
}
```

---

## Роли

| Роль | Описание |
|------|----------|
| `admin` | Полный доступ, обходит все проверки модулей |
| `manager` | Отчёты, управление пользователями, настройки |
| `operator` | Склад, подтверждение платежей |
| `cashier` | Продажи, клиенты, базовый склад |

---

## Формат ответов

### Успешный ответ
```json
{
  "success": true,
  "status_code": 200,
  "data": { ... }
}
```

### Список с пагинацией
```json
{
  "success": true,
  "status_code": 200,
  "data": [...],
  "meta": {
    "timestamp": "2026-03-14T10:30:00Z",
    "pagination": {
      "page": 1,
      "limit": 10,
      "offset": 0,
      "total": 45,
      "has_next": true,
      "has_prev": false,
      "total_pages": 5
    }
  }
}
```

### Ошибка
```json
{
  "success": false,
  "status_code": 400,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "request validation failed",
    "details": { "username": "field is required" },
    "request_id": "550e8400-..."
  }
}
```

### Коды ошибок
| Код | HTTP | Описание |
|-----|------|----------|
| `VALIDATION_ERROR` | 400 | Невалидные данные |
| `UNAUTHORIZED` | 401 | Нет/невалидный токен |
| `FORBIDDEN` | 403 | Нет прав |
| `*_NOT_FOUND` | 404 | Ресурс не найден |
| `CONFLICT` | 409 | Конфликт (дубликат, idempotency) |
| `INTERNAL_ERROR` | 500 | Ошибка сервера |

---

## Пагинация

Все списочные эндпоинты поддерживают:
- `page` — номер страницы (default: 1)
- `limit` — кол-во на странице (default: 10, max: 200)

---

## Управление пользователями

> Роль: `manager`

### GET `/api/auth/users` — список пользователей
### GET `/api/auth/users/:id` — один пользователь

### POST `/api/auth/users` — создать пользователя
```json
{
  "username": "cashier2",
  "password": "123456",
  "full_name": "Кассир 2",
  "phone": "65012345",
  "email": "c2@example.com",
  "is_active": true,
  "roles": ["cashier"]
}
```

### PATCH `/api/auth/users/:id` — обновить
```json
{
  "username": "cashier2_new",
  "full_name": "Кассир 2 Updated",
  "is_active": false,
  "roles": ["cashier", "operator"]
}
```

### DELETE `/api/auth/users/:id` — удалить (soft)
### POST `/api/auth/users/:id/block` — заблокировать
### POST `/api/auth/users/:id/unblock` — разблокировать
### POST `/api/auth/users/:id/password` — сменить пароль

---

## Клиенты (Customers)

> Роли: `cashier`, `operator`, `manager`

### GET `/api/customers` — список
Query: `search`, `active_only` (bool), `order_by` (name|created_at|total_spent), `order_direction` (asc|desc)

### GET `/api/customers/:id` — один клиент

### GET `/api/customers/by-card/:code` — поиск по карте (QR/штрихкод)
Пример: `GET /api/customers/by-card/CUST-ABC123`

### POST `/api/customers` — создать (cashier, manager)
```json
{
  "name": "Ахмед",
  "phone": "65012345",
  "email": "ahmed@mail.com",
  "type": "regular",
  "notes": "Постоянный клиент"
}
```
Типы: `regular`, `vip`, `wholesale`

### PATCH `/api/customers/:id/contact` — обновить контакт
```json
{ "name": "Ахмед Updated", "phone": "65099999" }
```

### PATCH `/api/customers/:id/admin` — обновить тип (manager)
```json
{ "type": "vip", "is_active": true }
```

### POST `/api/customers/:id/spent` — добавить потраченное
### DELETE `/api/customers/:id` — удалить (admin)

### Объект Customer
```json
{
  "id": 1,
  "name": "Ахмед",
  "phone": "65012345",
  "email": "ahmed@mail.com",
  "type": "vip",
  "total_spent": 50000,
  "bonus_points": 5000,
  "card_code": "CUST-ABC123",
  "is_active": true,
  "notes": "Постоянный клиент",
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-03-14T10:00:00Z"
}
```

---

## Продажи (Sales)

> Роли: `cashier`, `operator`, `manager`

### Жизненный цикл продажи

```
1. POST /api/sales          → создать DRAFT (резервирует товар)
2. GET /api/sales/:id       → посмотреть детали
3. DELETE /api/sales/:id/items/:item_id → удалить товар (нужен delete_code!)
4. POST /api/sales/:id/confirm → COMPLETED (списать товар, создать транзакцию)
5. POST /api/sales/:id/cancel  → CANCELLED (отменить резерв)
6. GET /receipt/:id?token=JWT   → публичный чек (HTML)
```

### POST `/api/sales` — создать драфт
```json
{
  "warehouse_id": 1,
  "customer_id": 5,
  "worker_id": 10,
  "items": [
    { "product_id": 100, "qty_milli": 2500 }
  ],
  "note": "Примечание",
  "force": false
}
```
- `qty_milli` = кол-во × 1000 (например 2.5 шт = 2500)
- `customer_id` — опционально
- `worker_id` — опционально (кредит работника)
- `force: true` — разрешить продажу в минус

### GET `/api/sales` — список продаж
Query: `warehouse_id`, `customer_id`, `created_by`, `status` (DRAFT|COMPLETED|CANCELLED), `date_from`, `date_to`

### GET `/api/sales/:id` — детали продажи
```json
{
  "sale": {
    "id": 1,
    "warehouse_id": 1,
    "customer_id": 5,
    "worker_id": 10,
    "total_cents": 75000,
    "cost_cents": 45000,
    "bonus_used_cents": 0,
    "items_count": 2,
    "note": "...",
    "status": "DRAFT",
    "sale_number": "SAL-2026-001234",
    "created_at": "2026-03-14T10:00:00Z"
  },
  "items": [
    {
      "id": 1,
      "product_id": 100,
      "product_name": "Товар А",
      "qty_milli": 2500,
      "unit_price_cents": 30000,
      "cost_cents": 15000,
      "line_total_cents": 75000
    }
  ],
  "transaction_id": 999
}
```

### POST `/api/sales/:id/confirm` — подтвердить
```json
{
  "payment_type_id": 1,
  "payment_amount": 50000,
  "payment_note": "Наличные",
  "bonus_used_cents": 0,
  "force": false
}
```

### POST `/api/sales/:id/cancel` — отменить

### DELETE `/api/sales/:id/items/:item_id` — удалить товар из драфта
```json
{ "delete_code": "0000" }
```
- Код по умолчанию: `0000`
- Менеджер/админ меняет через `PUT /api/receipt-settings`
- Неправильный код → `403 FORBIDDEN`
- Нельзя удалить последний товар

### GET `/receipt/:id?token=JWT` — публичный чек (HTML)
Открыть в браузере для печати на 80мм принтере.

---

## Склад (Stock)

> Роли: `operator`, `manager` (баланс: + `cashier`)

**Все количества в milli-units:** `qty_milli = кол-во × 1000`
**Все цены в центах:** `price_cents` (за 1.000 единиц)

### POST `/api/stock/in` — приход
```json
{
  "warehouse_id": 1,
  "product_id": 100,
  "qty_milli": 5000,
  "price_cents": 20000,
  "idempotency_key": "uuid4"
}
```

### POST `/api/stock/in/bulk` — массовый приход
```json
{
  "warehouse_id": 1,
  "items": [
    { "product_id": 100, "qty_milli": 5000, "price_cents": 20000, "idempotency_key": "uuid4" },
    { "product_id": 101, "qty_milli": 3000, "price_cents": 15000, "idempotency_key": "uuid4" }
  ]
}
```

### POST `/api/stock/out` — расход
```json
{
  "warehouse_id": 1,
  "product_id": 100,
  "qty_milli": 2500,
  "idempotency_key": "uuid4"
}
```

### POST `/api/stock/transfer` — перемещение
```json
{
  "from_warehouse_id": 1,
  "to_warehouse_id": 2,
  "product_id": 100,
  "qty_milli": 3000,
  "idempotency_key": "uuid4"
}
```

### POST `/api/stock/move` — корректировка
```json
{
  "warehouse_id": 1,
  "product_id": 100,
  "delta_milli": -1500,
  "type": "damaged",
  "idempotency_key": "uuid4",
  "note": "Брак"
}
```

### GET `/api/stock/balance` — текущие остатки
### GET `/api/stock/details` — история движений
### GET `/api/stock/negative` — товары в минусе

### Idempotency
Все движения требуют `idempotency_key` (UUID4). Повторный запрос с тем же ключом вернёт кэшированный результат. Другие параметры с тем же ключом → `409 CONFLICT`.

---

## Финансы (Finance)

> Роли: `cashier`, `operator`, `manager`

### GET `/api/payment-types` — типы оплаты
### GET `/api/transactions` — список транзакций
Query: `type` (income|expense), `status` (pending|partial|paid|canceled), `related_table`, `date_from`, `date_to`, `payment_type_id`

### GET `/api/transactions/:id` — детали транзакции
```json
{
  "id": 1,
  "type": "income",
  "status": "paid",
  "amount_cents": 100000,
  "paid_cents": 100000,
  "debt_cents": 0,
  "payments": [
    { "id": 1, "payment_type_id": 1, "amount_cents": 100000, "note": "Наличные" }
  ]
}
```

### POST `/api/transactions/manual` — ручная транзакция (operator, manager)
```json
{
  "type": "expense",
  "amount_cents": 50000,
  "reason": "Аренда",
  "payment_type_id": 1,
  "payment_amount": 50000
}
```

### POST `/api/transactions/:id/payments` — добавить платёж (operator, manager)
```json
{ "payment_type_id": 1, "amount_cents": 25000, "note": "Частичная оплата" }
```

### POST `/api/transactions/:id/cancel` — отменить (manager)

---

## Отчёты (Reports)

> Роли: `manager`, `admin`

### GET `/api/reports/dashboard` — дашборд
```json
{
  "today": { "revenue_cents": 150000, "profit_cents": 50000, "orders": 5, "change_pct": 12.5 },
  "week": { ... },
  "month": { ... },
  "low_stock_count": 3,
  "pending_payroll": 2,
  "top_products": [
    { "product_id": 100, "name": "Товар А", "revenue_cents": 500000 }
  ]
}
```

### GET `/api/reports/sales` — продажи по периодам
Query: `group_by` (day|week|month), `from`, `to` (RFC3339), `warehouse_id`, `customer_type`

### GET `/api/reports/sales/products` — продажи по товарам
Query: `from`, `to`, `warehouse_id`, `customer_type`

### GET `/api/reports/sales/export` — Excel экспорт (2 листа)
### GET `/api/reports/stock/export` — Excel экспорт движений (max 10k строк)

---

## Карточки работников (Worker Cards)

### GET `/api/worker-cards/by-card/:code` — поиск по карте (cashier, operator, manager)
Пример: `GET /api/worker-cards/by-card/WRK-ABC123`

### GET `/api/worker-cards` — список (manager)
Query: `worker_id`

### POST `/api/worker-cards` — создать (manager)
```json
{ "worker_id": 10, "label": "Основная карта" }
```

### PUT `/api/worker-cards/:id` — обновить (manager)
```json
{ "label": "Новое имя", "enabled": false }
```

### DELETE `/api/worker-cards/:id` — удалить (manager)

---

## Настройки чека (Receipt Settings)

> Роли: `manager`, `admin`

### GET `/api/receipt-settings` — получить настройки
### PUT `/api/receipt-settings` — обновить
```json
{
  "shop_name": "Hezzet Market",
  "shop_address": "ул. Главная 1",
  "shop_phone": "65012345",
  "footer": "Спасибо за покупку!",
  "delete_code": "1234"
}
```

---

## Уведомления (Notifications)

> Роль: `admin`

### GET `/api/notifications/sms` — логи SMS
Query: `status`, `type`, `phone`, `job_id`, `from_date`, `to_date`

### GET `/api/notifications/sms/:job_id` — один лог
### POST `/api/notifications/sms/:job_id/requeue` — повторить отправку

### GET `/api/notifications/phones` — телефоны для уведомлений
### POST `/api/notifications/phones`
```json
{ "phone": "+99365012345", "label": "Главный админ" }
```
### PUT `/api/notifications/phones/:id`
```json
{ "label": "Новое имя", "enabled": false }
```
### DELETE `/api/notifications/phones/:id`

---

## Права модулей (Permissions)

> Роли: `manager`, `admin`

### GET `/api/permissions` — список всех прав
### PUT `/api/permissions/:role/:module` — вкл/выкл модуль для роли
```json
{ "enabled": true }
```

Модули: `products`, `stock`, `customers`, `sales`, `finance`, `reports`, `workers`, `purchases`, `notifications`

---

## Важные заметки для фронтенда

### Денежные суммы
Все суммы в **центах** (тийинах). Для отображения: `total_cents / 100` → `"150.00 TMT"`

### Количество
Все количества в **milli-units**. Для отображения:
- Штуки: `qty_milli / 1000` → `"2 шт"`
- Вес/объём: `qty_milli / 1000` → `"1.500 кг"`

### Сканирование карт
- Клиент: `GET /api/customers/by-card/:code`
- Работник: `GET /api/worker-cards/by-card/:code`

### Удаление товара из драфта
Кассир должен ввести **код подтверждения** (`delete_code`). Код по умолчанию `0000`, менеджер меняет через настройки чека.

### Idempotency
Для всех складских операций отправляйте уникальный `idempotency_key` (UUID v4). Это защищает от дублирования при повторных запросах.
