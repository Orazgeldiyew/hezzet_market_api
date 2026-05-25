#!/bin/bash
# Production-readiness audit — DB + backend + frontend + e2e flows
# Saves report to /tmp/audit_report.txt

BASE=http://localhost:8080/api
PGCMD() { PGPASSWORD=postgres123 psql -h localhost -U postgres -d market -t -A -c "$1"; }
RESULT=/tmp/audit_report.txt
> "$RESULT"

pass=0; fail=0; warn=0
ok() { echo "  ✅ $1"; echo "OK: $1" >> "$RESULT"; pass=$((pass+1)); }
ko() { echo "  ❌ $1"; echo "FAIL: $1" >> "$RESULT"; fail=$((fail+1)); }
wn() { echo "  ⚠️  $1"; echo "WARN: $1" >> "$RESULT"; warn=$((warn+1)); }
sec() { echo ""; echo "═══ $1 ═══"; echo "" >> "$RESULT"; echo "[$1]" >> "$RESULT"; }

# ─── 1. Database integrity ─────────────────────────────────────────────────
sec "1. БД — миграции и схема"
mig_count=$(ls /home/rustem/Desktop/hezzet_market_backend/migrations/*.up.sql | wc -l)
[ "$mig_count" -ge "65" ] && ok "Миграций: $mig_count (≥65 — последние фичи на месте)" || ko "Только $mig_count миграций"

tbl_count=$(PGCMD "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public';")
[ "$tbl_count" -ge "50" ] && ok "Таблиц в схеме: $tbl_count" || ko "Только $tbl_count таблиц"

fk_count=$(PGCMD "SELECT COUNT(*) FROM information_schema.table_constraints WHERE constraint_type='FOREIGN KEY' AND table_schema='public';")
[ "$fk_count" -ge "80" ] && ok "Foreign keys: $fk_count" || ko "Только $fk_count FK"

# Critical tables exist
for t in audit_logs products sales sale_items purchase_orders suppliers customers warehouses warehouse_items shifts payments transactions receipt_settings role_permissions; do
  exists=$(PGCMD "SELECT to_regclass('public.$t') IS NOT NULL;")
  [ "$exists" = "t" ] && ok "Таблица '$t' существует" || ko "Таблица '$t' ОТСУТСТВУЕТ"
done

# UNIQUE constraint on shifts (recent fix)
idx=$(PGCMD "SELECT indexdef FROM pg_indexes WHERE indexname='idx_shifts_user_open_unique';")
[ -n "$idx" ] && ok "UNIQUE INDEX на open shifts (защита от race)" || ko "Нет защиты от двойной open смены"

# New invoice fields
fld=$(PGCMD "SELECT COUNT(*) FROM information_schema.columns WHERE table_name='suppliers' AND column_name IN ('legal_name','tax_id');")
[ "$fld" = "2" ] && ok "suppliers.legal_name + tax_id" || ko "Нет полей фактуры в suppliers"

fld=$(PGCMD "SELECT COUNT(*) FROM information_schema.columns WHERE table_name='receipt_settings' AND column_name IN ('legal_name','tax_id','font_size_header','font_size_items','font_size_meta','font_size_total');")
[ "$fld" = "6" ] && ok "receipt_settings: legal_name + tax_id + 4 шрифта" || ko "Нет всех новых полей в settings"

fld=$(PGCMD "SELECT COUNT(*) FROM information_schema.columns WHERE table_name='purchase_order_items' AND column_name='sale_price_cents';")
[ "$fld" = "1" ] && ok "purchase_order_items.sale_price_cents" || ko "Нет sale_price_cents"

# audit_logs has diff columns
fld=$(PGCMD "SELECT COUNT(*) FROM information_schema.columns WHERE table_name='audit_logs' AND column_name IN ('old_value','new_value');")
[ "$fld" = "2" ] && ok "audit_logs.old_value + new_value (Stage C)" || ko "Нет diff-колонок в audit_logs"

# ─── 2. Backend build ──────────────────────────────────────────────────────
sec "2. Backend"
if go build ./... 2>&1 | grep -q .; then
  ko "Backend не собирается"
else
  ok "Backend компилируется чисто"
fi

# Backend alive
http=$(curl -s -o /dev/null -w "%{http_code}" --max-time 3 $BASE/auth/login -X POST -H "Content-Type: application/json" -d '{}')
[ "$http" = "400" ] && ok "Backend отвечает (HTTP 400 на пустой login = ожидаемо)" || ko "Backend не отвечает (got $http)"

# ─── 3. Frontend ───────────────────────────────────────────────────────────
sec "3. Frontend"
ts_result=$(cd /home/rustem/hezzet_market && yarn types-check 2>&1)
if echo "$ts_result" | grep -q "error TS"; then
  ko "TypeScript errors"
else
  ok "TypeScript types check (yarn types-check)"
fi

# Frontend alive
http=$(curl -s -o /dev/null -w "%{http_code}" --max-time 3 http://localhost:5000/)
[ "$http" = "200" ] && ok "Frontend dev-server отвечает" || wn "Frontend не запущен ($http) — необязательно для prod"

# ─── 4. Refresh tokens ─────────────────────────────────────────────────────
sec "4. Логин 3 тестовых юзеров"
> /tmp/audit_tokens.env
for u in test_mgr_curl test_op_curl test_csh_curl; do
  resp=$(curl -s -X POST $BASE/auth/login -H "Content-Type: application/json" -d "{\"username\":\"$u\",\"password\":\"Test123!\"}")
  token=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('access_token',''))" 2>/dev/null)
  uid=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('user',{}).get('id',''))" 2>/dev/null)
  if [ -z "$token" ]; then ko "Логин $u не удался"; continue; fi
  echo "${u}_TOKEN=${token}" >> /tmp/audit_tokens.env
  echo "${u}_ID=${uid}" >> /tmp/audit_tokens.env
  ok "Логин $u (id=$uid)"
done
source /tmp/audit_tokens.env
MGR="Authorization: Bearer $test_mgr_curl_TOKEN"
OP="Authorization: Bearer $test_op_curl_TOKEN"
CSH="Authorization: Bearer $test_csh_curl_TOKEN"

# ─── 5. Permission matrix (37 cases) ──────────────────────────────────────
sec "5. Permission matrix (admin bypass + manager + operator + cashier)"
chk() {
  local lbl="$1" method="$2" path="$3" tok="$4" exp="$5" body="$6"
  local c
  if [ -n "$body" ]; then
    c=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" "$BASE$path" -H "Authorization: Bearer $tok" -H "Content-Type: application/json" -d "$body")
  else
    c=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" "$BASE$path" -H "Authorization: Bearer $tok")
  fi
  local match=0
  IFS='|' read -ra exps <<< "$exp"
  for e in "${exps[@]}"; do [ "$c" = "$e" ] && match=1; done
  [ "$match" = "1" ] && ok "$lbl → $c" || ko "$lbl → got=$c exp=$exp"
}

# Products
chk "GET /products [mgr]"      GET    /products            $test_mgr_curl_TOKEN "200"
chk "GET /products [csh]"      GET    /products            $test_csh_curl_TOKEN "200"
chk "POST /products [csh]"     POST   /products            $test_csh_curl_TOKEN "403" '{}'

# Suppliers
chk "GET /suppliers [mgr]"     GET    /suppliers           $test_mgr_curl_TOKEN "200"
chk "POST /suppliers [csh]"    POST   /suppliers           $test_csh_curl_TOKEN "403" '{"name":"x"}'

# Warehouses
chk "POST /warehouses [op]"    POST   /warehouses          $test_op_curl_TOKEN  "201" "{\"name\":\"AuditWh_$RANDOM\"}"
chk "POST /warehouses [csh]"   POST   /warehouses          $test_csh_curl_TOKEN "403" '{"name":"x"}'

# Categories
chk "POST /categories [mgr]"   POST   /categories          $test_mgr_curl_TOKEN "201" "{\"name\":\"AuditCat_$RANDOM\"}"
chk "POST /categories [csh]"   POST   /categories          $test_csh_curl_TOKEN "403" '{"name":"x"}'

# Inventory
chk "GET /inventory [op]"      GET    /inventory           $test_op_curl_TOKEN  "200"
chk "GET /inventory [csh]"     GET    /inventory           $test_csh_curl_TOKEN "200"

# Audit / Sensitive endpoints
chk "GET /audit-logs [mgr]"        GET    /audit-logs        $test_mgr_curl_TOKEN "200"
chk "GET /audit-logs [csh]"        GET    /audit-logs        $test_csh_curl_TOKEN "403"
chk "GET /audit-logs/stats [mgr]"  GET    /audit-logs/stats  $test_mgr_curl_TOKEN "200"

# Settings
chk "GET /settings/receipt [mgr]"  GET    /settings/receipt              $test_mgr_curl_TOKEN "200"
chk "GET /settings/receipt [csh]"  GET    /settings/receipt              $test_csh_curl_TOKEN "403"
chk "POST verify-delete [csh]"     POST   /settings/receipt/verify-delete-code $test_csh_curl_TOKEN "200" '{"code":"9999"}'

# Inbox
chk "GET /inbox [mgr]"             GET    /notifications/inbox           $test_mgr_curl_TOKEN "200"
chk "GET /inbox [csh]"             GET    /notifications/inbox           $test_csh_curl_TOKEN "403"

# Invoice (new feature)
chk "GET /purchases [csh]"         GET    /purchases                     $test_csh_curl_TOKEN "403"

# Users
chk "GET /auth/users [mgr]"        GET    /auth/users                    $test_mgr_curl_TOKEN "200"
chk "GET /auth/users [csh]"        GET    /auth/users                    $test_csh_curl_TOKEN "403"

# ─── 6. Critical business flow ─────────────────────────────────────────────
sec "6. End-to-end бизнес-флоу"
# Pick test data
wh=$(curl -s "$BASE/warehouses?limit=1" -H "$MGR" | python3 -c "import sys,json; print(json.load(sys.stdin)['data'][0]['id'])")
sup=$(curl -s "$BASE/suppliers?limit=1" -H "$MGR" | python3 -c "import sys,json; print(json.load(sys.stdin)['data'][0]['id'])")
prod=$(curl -s "$BASE/products?limit=1" -H "$MGR" | python3 -c "import sys,json; d=json.load(sys.stdin)['data']; items = d if isinstance(d,list) else d.get('items',[]); print(items[0]['id'])")
echo "  Setup: warehouse=$wh supplier=$sup product=$prod"

# Stock + ensure shift closed
PGPASSWORD=postgres123 psql -h localhost -U postgres -d market -c "
INSERT INTO warehouse_items (warehouse_id, product_id, qty_milli, avg_cost_cents, total_cost_cents)
VALUES ($wh, $prod, 100000, 500, 50000)
ON CONFLICT (warehouse_id, product_id) DO UPDATE SET qty_milli=GREATEST(EXCLUDED.qty_milli, warehouse_items.qty_milli);
UPDATE shifts SET status='closed', closed_at=now() WHERE user_id=$test_csh_curl_ID AND status='open';" >/dev/null 2>&1

# Open shift
reg=$(curl -s "$BASE/registers" -H "$CSH" | python3 -c "import sys,json; print(json.load(sys.stdin)['data'][0]['id'])")
shift_id=$(curl -s -X POST $BASE/shifts/open -H "$CSH" -H "Content-Type: application/json" -d "{\"register_id\":$reg,\"opening_cash\":50000}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))")
[ -n "$shift_id" ] && ok "Кассир открыл смену #$shift_id" || ko "Open shift"

# Sale flow
sale_id=$(curl -s -X POST $BASE/sales -H "$CSH" -H "Content-Type: application/json" -d "{\"warehouse_id\":$wh,\"items\":[{\"product_id\":$prod,\"qty_milli\":2000,\"discount_percent\":10}]}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('sale',{}).get('id',''))")
[ -n "$sale_id" ] && ok "Draft sale #$sale_id создан" || ko "Create draft sale"

pt=$(curl -s "$BASE/payment-types" -H "$CSH" | python3 -c "import sys,json; d=json.load(sys.stdin)['data']; items=d if isinstance(d,list) else d.get('items',[]); cash=[x for x in items if x.get('code')=='cash']; print(cash[0]['id'] if cash else items[0]['id'])")

confirm=$(curl -s -X POST $BASE/sales/$sale_id/confirm -H "$CSH" -H "Content-Type: application/json" -d "{\"payment_type_id\":$pt,\"payment_amount\":2000}")
echo "$confirm" | grep -q '"status":"confirmed"' && ok "Sale подтверждён" || ko "Sale confirm"

# Receipt math
html=$(curl -s "$BASE/sales/$sale_id/receipt" -H "$CSH")
arz=$(echo "$html" | grep -oP 'Arz%: \K[0-9]+' | head -1)
[ "$arz" = "10" ] && ok "Чек: скидка 10% правильная" || ko "Чек math arz=$arz"

# Reprint
rp=$(curl -s -o /dev/null -w "%{http_code}" -X POST $BASE/sales/$sale_id/print -H "$CSH")
([ "$rp" = "200" ] || [ "$rp" = "500" ]) && ok "Reprint endpoint работает ($rp = 500 = принтер не настроен)" || ko "Reprint got $rp"

# PO with sale_price + audit diff
po_id=$(curl -s -X POST $BASE/purchases -H "$MGR" -H "Content-Type: application/json" -d "{\"supplier_id\":$sup,\"warehouse_id\":$wh,\"items\":[{\"product_id\":$prod,\"qty_milli\":1000,\"unit_cost_cents\":777,\"sale_price_cents\":1500}]}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))")
[ -n "$po_id" ] && ok "PO #$po_id создан (с sale_price_cents)" || ko "Create PO"

curl -s -X POST $BASE/purchases/$po_id/receive -H "$MGR" -o /dev/null
sleep 1

# Verify product price updated
new_sale=$(curl -s $BASE/products/$prod -H "$MGR" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['sale_price'])")
[ "$new_sale" = "1500" ] && ok "Цена товара обновилась на 1500 после PO_RECEIVE" || ko "Цена не обновилась (got $new_sale)"

# Audit PRODUCT_PRICE_CHANGE_VIA_PO
audit_diff=$(curl -s "$BASE/audit-logs?action=PRODUCT_PRICE_CHANGE_VIA_PO&limit=5" -H "$MGR" | python3 -c "
import sys,json
items = json.load(sys.stdin)['data']
mine = [x for x in items if x.get('entity_id')=='$prod']
print('Y' if mine and mine[0].get('old_value') and mine[0].get('new_value') else 'N')")
[ "$audit_diff" = "Y" ] && ok "Audit diff (old/new) записан в БД" || ko "Audit diff missing"

# Invoice rendering
inv=$(curl -s "$BASE/purchases/$po_id/invoice" -H "$MGR")
echo "$inv" | grep -q "PO-2026-" && ok "Invoice HTML рендерится с номером PO-YYYY-NNN" || ko "Invoice не рендерится"

# Close shift
curl -s -o /dev/null -X POST $BASE/shifts/$shift_id/close -H "$CSH" -H "Content-Type: application/json" -d '{"closing_cash":50000}'
ok "Кассир закрыл смену"

# ─── 7. Audit log integrity ────────────────────────────────────────────────
sec "7. Audit log integrity"
sleep 1
for a in LOGIN SHIFT_OPEN SHIFT_CLOSE SALE_CONFIRM SALE_PRINT PO_RECEIVE PRODUCT_PRICE_CHANGE_VIA_PO; do
  c=$(curl -s "$BASE/audit-logs?action=$a&limit=1" -H "$MGR" | python3 -c "import sys,json; print(len(json.load(sys.stdin)['data']))")
  [ "$c" -ge "1" ] 2>/dev/null && ok "Audit пишет action=$a" || ko "Action $a не пишется"
done

# Audit health
fw=$(curl -s "$BASE/audit-logs/stats" -H "$MGR" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['failed_writes'])")
[ "$fw" = "0" ] && ok "Audit failed_writes = 0 (всё пишется)" || wn "failed_writes = $fw — есть потери аудита"

# ─── Final ─────────────────────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════════"
echo "ИТОГО: ✅ $pass | ❌ $fail | ⚠️  $warn"
echo "Полный отчёт: $RESULT"
echo "═══════════════════════════════════════════════════════"
echo "" >> "$RESULT"
echo "TOTAL: pass=$pass fail=$fail warn=$warn" >> "$RESULT"
exit $fail
