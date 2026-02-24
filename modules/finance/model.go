package finance

import "time"

// ---------- Domain ----------

type PaymentType struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type Transaction struct {
	ID                      int64     `json:"id"`
	PaymentTypeID           *int64    `json:"payment_type_id,omitempty"`
	Reason                  *string   `json:"reason,omitempty"`
	Status                  string    `json:"status"`
	AmountCents             int64     `json:"amount_cents"`
	Type                    string    `json:"type"`
	TablePaymentID          *string   `json:"table_payment_id,omitempty"`
	WarehouseItemDetailID   *int64    `json:"warehouse_item_detail_id,omitempty"`
	RelatedTable            string    `json:"related_table"`
	RelatedID               *int64    `json:"related_id,omitempty"`
	CreatedBy               *int64    `json:"created_by,omitempty"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type Payment struct {
	ID            int64     `json:"id"`
	TransactionID int64     `json:"transaction_id"`
	PaymentTypeID int64     `json:"payment_type_id"`
	AmountCents   int64     `json:"amount_cents"`
	Note          *string   `json:"note,omitempty"`
	CreatedBy     *int64    `json:"created_by,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// ---------- Requests ----------

type CreateManualRequest struct {
	Type          string  `json:"type" binding:"required,oneof=income expense"`
	AmountCents   int64   `json:"amount_cents" binding:"required,gt=0"`
	Reason        *string `json:"reason"`
	PaymentTypeID *int64  `json:"payment_type_id"`
	PaymentAmount *int64  `json:"payment_amount"`
	PaymentNote   *string `json:"payment_note"`
}

type AddPaymentRequest struct {
	PaymentTypeID int64   `json:"payment_type_id" binding:"required,gt=0"`
	AmountCents   int64   `json:"amount_cents" binding:"required,gt=0"`
	Note          *string `json:"note"`
}

type TransactionFilter struct {
	Type          *string
	Status        *string
	RelatedTable  *string
	RelatedID     *int64
	DateFrom      *time.Time
	DateTo        *time.Time
	PaymentTypeID *int64
}

// ---------- Responses ----------

type TransactionDetail struct {
	Transaction
	Payments  []Payment `json:"payments"`
	PaidCents int64     `json:"paid_cents"`
	DebtCents int64     `json:"debt_cents"`
}

type TransactionListResult struct {
	Items  []Transaction `json:"items"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}
