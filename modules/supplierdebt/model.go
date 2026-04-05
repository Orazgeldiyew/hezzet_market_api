package supplierdebt

import "time"

type SupplierDebt struct {
	ID             int64     `json:"id"`
	SupplierID     int64     `json:"supplier_id"`
	SupplierName   string    `json:"supplier_name,omitempty"`
	PurchaseID     *int64    `json:"purchase_id,omitempty"`
	AmountCents    int64     `json:"amount_cents"`
	RemainingCents int64     `json:"remaining_cents"`
	Status         string    `json:"status"`
	Note           *string   `json:"note,omitempty"`
	CreatedBy      *int64    `json:"created_by,omitempty"`
	CreatedByName  string    `json:"created_by_name,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type DebtPayment struct {
	ID            int64     `json:"id"`
	DebtID        int64     `json:"debt_id"`
	AmountCents   int64     `json:"amount_cents"`
	PaymentTypeID *int64    `json:"payment_type_id,omitempty"`
	Note          *string   `json:"note,omitempty"`
	CreatedBy     *int64    `json:"created_by,omitempty"`
	CreatedByName string    `json:"created_by_name,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// ─── Requests ───────────────────────────────────────────────────────────────

type CreateRequest struct {
	SupplierID  int64  `json:"supplier_id" binding:"required,gt=0"`
	AmountCents int64  `json:"amount_cents" binding:"required,gt=0"`
	Note        string `json:"note"`
}

type PayRequest struct {
	AmountCents   int64  `json:"amount_cents" binding:"required,gt=0"`
	PaymentTypeID *int64 `json:"payment_type_id"`
	Note          string `json:"note"`
}

// ─── Responses ──────────────────────────────────────────────────────────────

type SupplierDebtSummary struct {
	SupplierID   int64  `json:"supplier_id"`
	SupplierName string `json:"supplier_name"`
	TotalDebt    int64  `json:"total_debt_cents"`
	OpenDebts    int    `json:"open_debts"`
}
