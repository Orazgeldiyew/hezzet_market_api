package payroll

import "time"

// ---------- Domain ----------

type PayrollRun struct {
	ID              int64     `json:"id"`
	WorkerID        int64     `json:"worker_id"`
	Period          string    `json:"period"`
	BaseSalaryCents int64     `json:"base_salary_cents"`
	FinesCents      int64     `json:"fines_cents"`
	DebtsCents      int64     `json:"debts_cents"`
	NetSalaryCents  int64     `json:"net_salary_cents"`
	Status          string    `json:"status"`
	TransactionID   *int64    `json:"transaction_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ---------- Requests ----------

type CalculateRequest struct {
	WorkerID int64  `json:"worker_id" binding:"required,gt=0"`
	Period   string `json:"period" binding:"required"`
}

type PayRequest struct {
	PaymentTypeCode string `json:"payment_type_code" binding:"required"`
}

// ---------- Export ----------

// PayrollExportRow joins payroll_runs with worker info for the Excel report.
type PayrollExportRow struct {
	WorkerName      string
	Position        string
	Period          string
	BaseSalaryCents int64
	FinesCents      int64
	DebtsCents      int64
	NetSalaryCents  int64
	Status          string
}

// ---------- Responses ----------

type PayrollListResult struct {
	Items  []PayrollRun `json:"items"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}
