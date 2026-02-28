# Hezzet Market Backend

Backend-система управления торговым предприятием. REST API на Go (Gin + PostgreSQL), покрывает полный цикл работы магазина: склад, продажи, закупки, финансы, персонал, клиенты.

---

## Технологический стек

| Компонент | Технология |
|-----------|------------|
| Язык | Go |
| HTTP-фреймворк | Gin |
| База данных | PostgreSQL (pgxpool, без ORM) |
| Очередь задач | Redis |
| Документация API | Swagger (swaggo) |
| Аутентификация | JWT (access + refresh токены) |
| SMS-уведомления | Twilio / log-провайдер |

---

## Архитектура

```
modules/
├── auth/           — аутентификация, пользователи
├── category/       — категории товаров
├── customer/       — клиенты
├── finance/        — финансовые транзакции, платежи
├── notification/   — SMS-уведомления
├── payroll/        — расчёт зарплаты
├── product/        — товары, штрихкоды, фото
├── purchase/       — заказы поставщикам, долги
├── reports/        — отчёты, экспорт
├── sale/           — продажи
├── stock/          — склад, движения товаров
├── supplier/       — поставщики
├── warehouse/      — склады
├── workerfinance/  — штрафы, долги сотрудников
├── workers/        — сотрудники
└── auditlog/       — журнал действий
```

Каждый модуль следует паттерну: `model → repository → service → handler → routes`.

---

## Роли пользователей

| Роль | Описание |
|------|----------|
| `admin` | Полный доступ ко всем эндпоинтам |
| `manager` | Управление: отчёты, зарплата, финансы |
| `operator` | Операционная работа: склад, закупки, товары |
| `cashier` | Касса: создание и подтверждение продаж |

---

## Модули и функции

### Аутентификация (`/auth`)

Управление пользователями системы и сессиями.

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| POST | `/auth/register` | — | Регистрация аккаунта |
| POST | `/auth/login` | — | Вход, получение JWT-токенов |
| POST | `/auth/refresh` | — | Обновление access-токена |
| POST | `/auth/users` | admin | Создать пользователя |
| GET | `/auth/users` | admin | Список пользователей |
| GET | `/auth/users/:id` | admin | Профиль пользователя |
| PATCH | `/auth/users/:id` | admin | Изменить данные пользователя |
| DELETE | `/auth/users/:id` | admin | Удалить пользователя |
| POST | `/auth/users/:id/block` | admin | Заблокировать пользователя |
| POST | `/auth/users/:id/unblock` | admin | Разблокировать пользователя |
| POST | `/auth/users/:id/password` | admin / self | Сменить пароль |

---

### Товары (`/api/products`)

Каталог товаров с поддержкой категорий, штрихкодов и фотографий.

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| POST | `/api/products` | operator | Создать товар (multipart/form-data) |
| GET | `/api/products` | all | Список товаров (поиск, пагинация) |
| GET | `/api/products/:id` | all | Карточка товара |
| PATCH | `/api/products/:id` | operator | Обновить товар |
| DELETE | `/api/products/:id` | operator | Удалить товар |
| GET | `/api/products/:id/card` | all | Карточка с остатками по складам |
| GET | `/api/products/:id/categories` | all | Категории товара |
| PUT | `/api/products/:id/categories` | operator | Установить категории |
| DELETE | `/api/products/:id/categories/:categoryId` | operator | Убрать категорию |
| POST | `/api/products/:id/photo` | operator | Загрузить фото |

Особенности:
- Единицы измерения: штуки, кг, литры, метры и др.
- Количество хранится в **миллиединицах** (1000 = 1 единица) для точного расчёта дробей
- Штрихкоды (barcode) для сканера

---

### Категории (`/api/categories`)

Иерархическое дерево категорий для товаров.

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| POST | `/api/categories` | operator | Создать категорию |
| GET | `/api/categories` | all | Список категорий |
| GET | `/api/categories/tree` | all | Дерево категорий |
| GET | `/api/categories/:id` | all | Категория по ID |
| PATCH | `/api/categories/:id` | operator | Обновить категорию |
| DELETE | `/api/categories/:id` | operator | Удалить категорию |

---

### Склад (`/api/stock`)

Управление остатками, движениями товаров и взвешенной средней себестоимостью.

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| POST | `/api/stock/in` | operator, manager | Приход товара на склад |
| POST | `/api/stock/in/bulk` | operator, manager | Массовый приход |
| POST | `/api/stock/out` | operator, manager | Списание товара со склада |
| POST | `/api/stock/transfer` | operator, manager | Перемещение между складами |
| POST | `/api/stock/move` | operator, manager | Корректировка остатков |
| GET | `/api/stock/balance` | all | Текущие остатки по складам |
| GET | `/api/stock/details` | all | История движений товаров |

Особенности:
- **Идемпотентность**: каждая операция принимает `idempotency_key` — повторный запрос с тем же ключом возвращает оригинальный результат без дублирования
- **Средняя себестоимость** (`avg_cost_cents`) пересчитывается автоматически при каждом приходе
- **Доступное количество** (`available_milli`) = остаток минус активные резервации (незакрытые продажи)

---

### Продажи (`/api/sales`)

Двухшаговый процесс продажи: черновик → подтверждение.

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| POST | `/api/sales` | cashier, operator, manager | Создать продажу (черновик) |
| GET | `/api/sales` | all | Список продаж |
| GET | `/api/sales/:id` | all | Детали продажи |
| POST | `/api/sales/:id/confirm` | cashier, operator, manager | Подтвердить продажу |
| POST | `/api/sales/:id/cancel` | cashier, operator, manager | Отменить продажу |

Жизненный цикл:
```
draft ──→ confirmed   (товар списан, проведена оплата, COGS записан)
  └─────→ cancelled   (резервация снята, склад не тронут)
```

- При создании черновика — товар **резервируется** (не списывается физически)
- При подтверждении — резервация закрывается, товар списывается, создаётся финансовая транзакция
- При отмене подтверждённой продажи — товар возвращается на склад

---

### Заказы поставщикам / Закупки (`/api/purchases`)

Управление закупками и долгами перед поставщиками.

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| POST | `/api/purchases` | operator, manager | Создать заказ (черновик) |
| GET | `/api/purchases` | operator, manager | Список заказов |
| GET | `/api/purchases/debt` | operator, manager | Сводка долгов по поставщикам |
| GET | `/api/purchases/:id` | operator, manager | Детали заказа |
| POST | `/api/purchases/:id/receive` | operator, manager | Принять товар |
| POST | `/api/purchases/:id/cancel` | operator, manager | Отменить заказ |
| POST | `/api/purchases/:id/payments` | operator, manager | Записать платёж поставщику |

Жизненный цикл:
```
draft ──→ received   (товар поступил на склад, создан долг поставщику)
  └─────→ cancelled
```

- При приёмке — товар добавляется на склад, пересчитывается `avg_cost_cents`
- Долг поставщику — это финансовая транзакция типа `expense` со статусом `pending`
- Платежи переводят статус долга: `pending → partial → paid`
- `GET /debt` показывает только поставщиков с непогашенным остатком

---

### Финансы (`/api/transactions`, `/api/payment-types`)

Учёт всех денежных операций предприятия.

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| GET | `/api/payment-types` | all | Типы оплаты (наличные, карта и др.) |
| GET | `/api/transactions` | all | Список транзакций (с фильтрами) |
| GET | `/api/transactions/:id` | all | Детали транзакции + платежи |
| POST | `/api/transactions/manual` | operator, manager | Ручная транзакция |
| POST | `/api/transactions/:id/payments` | operator, manager | Добавить платёж |
| POST | `/api/transactions/:id/cancel` | manager | Отменить транзакцию |

Типы транзакций: `income` (доход) / `expense` (расход).
Статусы транзакций: `pending` → `partial` → `paid` / `canceled`.

---

### Поставщики (`/api/suppliers`)

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| POST | `/api/suppliers` | operator | Создать поставщика |
| GET | `/api/suppliers` | operator | Список поставщиков |
| GET | `/api/suppliers/:id` | operator | Поставщик по ID |
| PATCH | `/api/suppliers/:id` | operator | Обновить поставщика |
| DELETE | `/api/suppliers/:id` | operator | Удалить поставщика |

---

### Клиенты (`/api/customers`)

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| POST | `/api/customers` | cashier, manager | Создать клиента |
| GET | `/api/customers` | all | Список клиентов |
| GET | `/api/customers/:id` | all | Клиент по ID |
| PATCH | `/api/customers/:id/contact` | all | Обновить контактные данные |
| PATCH | `/api/customers/:id/admin` | manager | Обновить бизнес-поля |
| POST | `/api/customers/:id/spent` | cashier, manager | Начислить потраченную сумму |
| DELETE | `/api/customers/:id` | admin | Удалить клиента |

---

### Склады (`/api/warehouses`)

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| POST | `/api/warehouses` | manager | Создать склад |
| GET | `/api/warehouses` | all | Список складов |
| GET | `/api/warehouses/:id` | all | Склад по ID |

---

### Сотрудники (`/api/workers`)

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| POST | `/api/workers` | manager | Создать сотрудника |
| GET | `/api/workers` | manager, operator | Список сотрудников |
| GET | `/api/workers/:id` | manager, operator | Сотрудник по ID |
| PATCH | `/api/workers/:id` | manager | Обновить данные |
| DELETE | `/api/workers/:id` | admin | Удалить сотрудника |

---

### Финансы сотрудников (`/api/workers/:id/...`)

Учёт компенсаций, штрафов и долгов сотрудников.

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| POST | `/api/workers/compensation` | manager | Установить ставку компенсации |
| GET | `/api/workers/:id/compensation` | manager | Получить ставку |
| POST | `/api/workers/:id/fines` | manager | Создать штраф |
| GET | `/api/workers/:id/fines` | manager | Список штрафов |
| POST | `/api/workers/:id/debts` | manager | Создать долг сотрудника |
| GET | `/api/workers/:id/debts` | manager | Список долгов |

---

### Зарплата (`/api/payroll`)

Расчёт и выплата зарплаты сотрудникам.

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| GET | `/api/payroll` | manager | История расчётов зарплаты |
| POST | `/api/payroll/calculate` | manager | Рассчитать зарплату за период |
| POST | `/api/payroll/:id/pay` | manager | Провести выплату |
| GET | `/api/payroll/export` | manager | Экспорт в Excel |

---

### Отчёты (`/api/reports`)

Аналитика и экспорт данных.

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| GET | `/api/reports/dashboard` | manager, admin | Сводный дашборд (выручка, расходы, топ товары) |
| GET | `/api/reports/sales` | manager, admin | Продажи по периоду |
| GET | `/api/reports/sales/products` | manager, admin | Продажи по товарам |
| GET | `/api/reports/sales/export` | manager, admin | Экспорт продаж в Excel |
| GET | `/api/reports/stock/export` | manager, admin | Экспорт остатков в Excel |

Фильтры отчётов: `date_from`, `date_to`, `warehouse_id`, `product_id`, `category_id`.

---

### SMS-уведомления (`/api/notifications/sms`)

Автоматические уведомления о низком остатке товаров.

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| GET | `/api/notifications/sms` | admin | Лог SMS-сообщений |
| GET | `/api/notifications/sms/:job_id` | admin | Детали задачи |
| POST | `/api/notifications/sms/:job_id/requeue` | admin | Повторить упавшую задачу |

Статусы SMS: `queued → sending → sent` / `retrying → dlq` / `failed` / `rate_limited`.

---

### Журнал действий (`/api/audit-logs`)

Автоматическая запись всех изменяющих операций.

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| GET | `/api/audit-logs` | manager | Список записей журнала |

Фиксируются: кто выполнил, какое действие, какая сущность, IP-адрес, время.

Именованные действия:

| Действие | Событие |
|----------|---------|
| `STOCK_IN` / `STOCK_OUT` | Движение товара |
| `STOCK_TRANSFER` | Перемещение между складами |
| `SALE_CONFIRM` / `SALE_CANCEL` | Жизненный цикл продажи |
| `PO_RECEIVE` / `PO_CANCEL` | Жизненный цикл закупки |
| `PO_PAYMENT` | Платёж поставщику |
| `USER_BLOCK` / `USER_UNBLOCK` | Блокировка пользователя |
| `PASSWORD_CHANGE` | Смена пароля |
| `CALCULATE` / `PAY` | Расчёт/выплата зарплаты |
| `FINE_CREATE` / `DEBT_CREATE` | Штраф/долг сотрудника |

---

## Ключевые концепции

### Миллиединицы (qty_milli)
Все количества хранятся умноженными на 1000.
- `1000 milli` = 1 единица
- `500 milli` = 0.5 кг
Позволяет работать с дробными количествами без потери точности.

### Взвешенная средняя себестоимость
При каждом приходе товара `avg_cost_cents` пересчитывается:
```
avg = (старый_итог + новый_приход) * 1000 / (старое_qty + новое_qty)
```

### Резервирование остатков
При создании черновика продажи товар **резервируется**.
`available_milli = qty_milli - активные_резервации`
Клиент видит реальное доступное количество, не допуская двойных продаж.

### Идемпотентность складских операций
Каждая операция с `idempotency_key` выполняется ровно один раз.
Повторный запрос с тем же ключом вернёт оригинальный ответ без дублирования данных.

### Двойной цикл закупок и продаж

```
Закупка:   draft → received → (partial payments) → paid
Продажа:   draft → confirmed
                └→ cancelled
```

---

## Запуск

```bash
# Установить переменные окружения (.env)
cp .env.example .env

# Применить миграции (используйте golang-migrate или аналог)
migrate -path migrations -database "postgres://..." up

# Запустить сервер
go run ./cmd/...
```

### Переменные окружения

| Переменная | Описание |
|------------|----------|
| `DATABASE_URL` | Строка подключения к PostgreSQL |
| `JWT_SECRET` | Секрет для подписи JWT |
| `REDIS_URL` | Адрес Redis (опционально) |
| `SMS_PROVIDER` | `log` (дев) или `twilio` |
| `TWILIO_*` | Параметры Twilio (если используется) |
| `UPLOADS_DIR` | Папка для загружаемых файлов |
| `PUBLIC_BASE_URL` | Базовый URL для ссылок на фото |
| `ENV` | `dev` / `prod` |
| `SWAGGER_*` | Параметры доступа к Swagger-документации |

---

## API-документация

После запуска в `dev`-режиме доступна по адресу:
```
http://localhost:{PORT}/swagger/index.html
```
