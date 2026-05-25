# Production Deploy Checklist

Чек-лист для деплоя backend (Go) + frontend (React) на production-сервер.
Проходи по пунктам **по порядку** — каждый шаг зависит от предыдущего.

---

## 0. Перед началом — backup

**Обязательно** перед миграциями БД:

```bash
# на сервере
pg_dump -h localhost -U postgres market > /tmp/market_backup_$(date +%Y%m%d_%H%M%S).sql
# проверь размер — не должен быть 0
ls -lh /tmp/market_backup_*.sql
```

Если деплой пойдёт не так — восстановишь:
```bash
psql -h localhost -U postgres -d market < /tmp/market_backup_YYYYMMDD_HHMMSS.sql
```

---

## 1. Подтянуть код

```bash
cd /path/to/hezzet_market_backend
git fetch origin
git checkout main
git pull origin main
# или git checkout dev && git pull origin dev — если работаешь на dev
```

Проверь что подтянулось:
```bash
git log --oneline -5
# должен быть видно "new features" (c48b24a) и более новые
```

---

## 2. Накатить миграции

**Список миграций добавленных в этой версии:**

| # | Файл | Что делает |
|---|---|---|
| 062 | `notification_reads.up.sql` | Per-user read state for inbox bell |
| 063 | `shifts_unique_open.up.sql` | UNIQUE INDEX (защита от двойной open смены) |
| 064 | `audit_logs_diff.up.sql` | `old_value`/`new_value` колонки (Stage C audit) |
| 065 | `po_item_sale_price.up.sql` | `sale_price_cents` в PO items |
| 066 | `receipt_font_sizes.up.sql` | 4 шрифта в receipt_settings |
| 067 | `invoice_fields.up.sql` | `legal_name`/`tax_id` для suppliers + shop |
| 068 | `transactions_idempotency.up.sql` | Idempotency key для manual transactions |
| 069 | `permission_import_action.up.sql` | Новый action `import` в role_permissions |
| 070 | `sales_history_permission.up.sql` | `sales:history` permission rows |

### Если есть migrate CLI

```bash
migrate -path migrations -database "$DB_DSN" up
```

### Если накатываешь вручную через psql

```bash
for f in migrations/06{2,3,4,5,6,7,8,9}_*.up.sql migrations/070_*.up.sql; do
    echo "=== $f ==="
    psql -h $DB_HOST -U $DB_USERNAME -d $DB_DATABASE -f "$f" || break
done
```

Проверь что накатилось:
```sql
-- В psql
SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public';
-- должно быть >= 53 таблиц

SELECT column_name FROM information_schema.columns
WHERE table_name='transactions' AND column_name='idempotency_key';
-- должна вернуть 1 строку

SELECT COUNT(*) FROM role_permissions WHERE module='products' AND action='import';
-- должно быть N rows (по числу ролей)
```

---

## 3. Сбросить Redis кэш permissions

**Критично** — иначе админ ставит галку в `/roles` UI и она не применяется до истечения TTL:

```bash
redis-cli --scan --pattern "perm:v2:*" | xargs -r redis-cli DEL
```

---

## 4. Пересобрать backend

```bash
# Опционально — обновить swagger docs (если используется)
make swagger

# Собрать бинарник
make build
# или
go build -o hezzet_market_backend main.go
```

---

## 5. Проверить env vars

Должны быть выставлены **минимум:**

```bash
# .env / systemd EnvironmentFile / docker-compose
APP_ENV=production
HTTP_ADDR=:8080
PUBLIC_BASE_URL=https://your-domain.tm   # ← обязательно для логотипов/чеков

DB_HOST=localhost
DB_PORT=5432
DB_USERNAME=postgres
DB_PASSWORD=<секрет>
DB_DATABASE=market

JWT_SECRET=<длинная случайная строка>
ACCESS_TOKEN_SECRET=<длинная>
REFRESH_TOKEN_SECRET=<длинная>
ACCESS_TOKEN_EXPIRES_IN=24h
REFRESH_TOKEN_EXPIRES_IN=30d

REDIS_URL=redis://localhost:6379

UPLOADS_DIR=/var/lib/hezzet/uploads   # ← должен существовать и быть writable

# SMS / notifications (если используются)
SMS_PROVIDER=log     # или twilio
SMS_FROM=HezzetMarket
SMS_WORKERS=1
SMS_RATE_LIMIT_PER_HOUR=3
```

**Не меняй `JWT_SECRET` / `ACCESS_TOKEN_SECRET` без острой необходимости** — иначе все юзеры будут разлогинены.

---

## 6. Перезапустить backend

### Если через systemd:

```bash
sudo systemctl restart hezzet_market_backend
sudo journalctl -u hezzet_market_backend -f --since "1 minute ago"
```

### Если через docker-compose:

```bash
docker-compose pull
docker-compose up -d
docker-compose logs -f backend
```

### Smoke-check:

```bash
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/api/auth/login -X POST -H "Content-Type: application/json" -d '{}'
# должно вернуть 400 (валидация)

curl -s http://localhost:8080/health
# должно вернуть {"status":"ok"} или подобное
```

---

## 7. Frontend deploy

```bash
cd /path/to/hezzet_market_frontend
git pull origin main

# Установить пакеты если package.json менялся
yarn install --frozen-lockfile

# Сборка production-бандла
yarn build:react

# Положить на сервер
# Например, для nginx:
sudo cp -r dist/* /var/www/hezzet_market/
sudo systemctl reload nginx
```

---

## 8. Полный production-аудит (smoke)

Запусти готовый скрипт:

```bash
cd /path/to/hezzet_market_backend
bash dev/production_audit.sh
```

Должен выдать **68 ✅ из 68** (с warning о frontend dev-сервере если он не запущен — это нормально для prod).

---

## 9. Тест критичных flow вручную

| Кто | Что | Ожидается |
|---|---|---|
| **Кассир** | Открыть смену | ✅ |
| Кассир | Продажа: скан → подтвердить | ✅ чек печатается |
| Кассир | Попытка продать товар с ценой 0 | ❌ блокировано (валидация) |
| Кассир | Попытка открыть `/sales-history` | ❌ 403 / в меню нет |
| **Менеджер** | `/products` → импорт Excel | ✅ N товаров создано |
| Менеджер | `/products/edit/:id` → менять цены | ✅ |
| Менеджер | `/purchases` → создать PO → подтвердить | ✅ цена обновится автоматически |
| Менеджер | `/purchases/:id` → "Печать накладной" | ✅ A4 PDF в новой вкладке |
| Менеджер | `/audit-logs` → видит все действия с diff | ✅ |
| **Admin** | `/roles` → ставить/снимать галки | ✅ применяется без рестарта |

---

## 10. Post-deploy мониторинг (первые 30 мин)

Смотри логи на:
- ❌ `permission denied` — лишний 403 → откатить permission grants в БД
- ❌ `idempotency_key` ошибки — двойные клики
- ❌ `audit-log FAILED` — потери аудита
- ❌ `SCAN error` — проблема с NULL handling

Если что-то горит:
```bash
# Rollback миграций (если поддерживается)
migrate -path migrations -database "$DB_DSN" down 9   # назад на 9 миграций
# Или восстановить из backup'а (см. шаг 0)

# Откатить код
git reset --hard <предыдущий-коммит>
```

---

## Если что-то ломается — диагностика

### Permission errors после деплоя
```bash
redis-cli --scan --pattern "perm:v2:*" | xargs -r redis-cli DEL
```

### `docs.go: unknown field LeftDelim`
Версии swag CLI и library разошлись:
```bash
go get github.com/swaggo/swag@v1.16.4
go mod tidy
make swagger
```

### Frontend "товаров не видно"
- Проверь network tab — `GET /api/products?limit=20` возвращает данные?
- Backend кеш JS — `nginx -s reload` после деплоя
- Браузер кеш — попроси юзера Ctrl+Shift+R

### `Backend не стартует, port in use`
```bash
sudo fuser -k 8080/tcp
sudo systemctl restart hezzet_market_backend
```

---

## Контакты в случае проблем

- **Логи backend**: `journalctl -u hezzet_market_backend -n 200`
- **Логи БД**: `tail -f /var/log/postgresql/postgresql-16-main.log`
- **Логи nginx**: `tail -f /var/log/nginx/error.log`

---

## Что было задеплоено в этом релизе

Большие фичи:
- 📦 **Excel импорт товаров** (`POST /api/products/import`)
- 🧾 **A4 накладные** для поставщиков (`GET /api/purchases/:id/invoice`)
- 🔐 **Permission `import`** для bulk-операций
- 📊 **Audit diff** — Stage C (`old_value`/`new_value`)
- 💰 **PO с sale_price** — приёмка обновляет цену продажи товара
- 🔢 **Идемпотентность** ручных transactions
- 🛡️ **IDOR fix** на `/sales/:id` (cashier видит только свои)
- 🚫 **Блок продажи** товара с нулевой ценой
- 📑 **Sales history permission** — управляется через `/roles`

Smaller fixes:
- Размер шрифтов чека настраивается с фронта
- Колокольчик уведомлений для admin/manager
- Force-close смены менеджером
- Verify-delete-code endpoint (без утечки кода)
- Trigram-search opt (если включён `pg_trgm` extension)
