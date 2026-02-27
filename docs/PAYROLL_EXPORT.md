# Payroll Excel Export

Download a monthly payroll report as an `.xlsx` file for all workers in a given period.

---

## Endpoint

```
GET /api/payroll/export?period=YYYY-MM
```

| | |
|---|---|
| **Auth** | Required — `Authorization: Bearer <token>` |
| **Role** | `manager` or `admin` |
| **Response** | `.xlsx` file download |

---

## Query Parameters

| Parameter | Type | Required | Example | Description |
|---|---|---|---|---|
| `period` | string | Yes | `2026-02` | Month to export in `YYYY-MM` format |

---

## Response

On success the server responds with:

```
HTTP 200 OK
Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
Content-Disposition: attachment; filename=payroll_2026-02.xlsx
```

The response body is a binary `.xlsx` file — **not JSON**.

On error the server responds with JSON as usual:

```json
{ "success": false, "error": { "code": "VALIDATION_ERROR", "message": "period query param is required" } }
```

---

## Excel File Structure

One sheet named **"Payroll YYYY-MM"**. One row per worker, ordered alphabetically by name.

| Column | Value | Example |
|---|---|---|
| `#` | Row number | `1` |
| `Worker` | Worker full name | `Ali Yusupov` |
| `Position` | Worker position | `Cashier` |
| `Period` | Month | `2026-02` |
| `Base Salary` | Gross salary (decimal) | `5000.00` |
| `Fines` | Total deducted fines (decimal) | `100.00` |
| `Debts` | Total deducted debts/advances (decimal) | `0.00` |
| `Net Salary` | Amount to pay = Base − Fines − Debts | `4900.00` |
| `Status` | `calculated` or `paid` | `calculated` |

> All monetary values are in the local currency unit (not cents).

---

## How It Works

```
Worker compensation table        Worker fines / debts
       ↓                                ↓
  POST /api/payroll/calculate   ←  sums open fines + debts
       ↓
  payroll_runs record created (status: calculated)
       ↓
  GET /api/payroll/export?period=2026-02
       ↓
  JOINs payroll_runs + workers table
       ↓
  Returns .xlsx file
```

A payroll run must be **calculated first** before it appears in the export.
Workers with no calculated payroll for the period are not included.

---

## Frontend Usage

### JavaScript / Fetch

```js
async function downloadPayrollExcel(period, token) {
  const response = await fetch(`/api/payroll/export?period=${period}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    const err = await response.json();
    throw new Error(err.error?.message || 'Export failed');
  }

  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `payroll_${period}.xlsx`;
  a.click();
  URL.revokeObjectURL(url);
}

// Usage
downloadPayrollExcel('2026-02', accessToken);
```

### Axios

```js
const response = await axios.get('/api/payroll/export', {
  params: { period: '2026-02' },
  headers: { Authorization: `Bearer ${token}` },
  responseType: 'blob',
});

const url = URL.createObjectURL(new Blob([response.data]));
const a = document.createElement('a');
a.href = url;
a.download = `payroll_2026-02.xlsx`;
a.click();
URL.revokeObjectURL(url);
```

> **Important:** Set `responseType: 'blob'` (Axios) or handle as blob (Fetch).
> Do not try to parse the response as JSON — it is binary data.

---

## curl Example

```bash
# 1. Login
TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | jq -r '.data.access_token')

# 2. Download
curl -X GET "http://localhost:8080/api/payroll/export?period=2026-02" \
  -H "Authorization: Bearer $TOKEN" \
  --output payroll_2026-02.xlsx
```

---

## Error Cases

| Situation | HTTP | Error code | Message |
|---|---|---|---|
| Missing `period` param | 400 | `VALIDATION_ERROR` | `period query param is required` |
| Invalid format (e.g. `2026-2`) | 400 | `VALIDATION_ERROR` | `period must be YYYY-MM format` |
| No token | 401 | `UNAUTHORIZED` | `authorization header missing` |
| Wrong role (e.g. cashier) | 403 | `FORBIDDEN` | `forbidden` |
| No payroll data for period | 200 | — | Empty sheet (header row only) |

---

## Full Workflow Example

```
1. Create worker          POST /api/workers
2. Set compensation       POST /api/workers/compensation
3. (Optional) Add fines   POST /api/workers/{id}/fines
4. (Optional) Add debts   POST /api/workers/{id}/debts
5. Calculate payroll      POST /api/payroll/calculate  { worker_id, period }
6. Export Excel           GET  /api/payroll/export?period=2026-02
```

Steps 1–5 must be done before export. Step 5 can be repeated for multiple workers.
