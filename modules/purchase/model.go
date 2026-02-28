package purchase

import "time"

// ── Domain ───────────────────────────────────────────────────────────────────

type PurchaseOrder struct {
	ID          int64      `json:"id"`
	SupplierID  int64      `json:"supplier_id"`
	WarehouseID int64      `json:"warehouse_id"`
	Status      string     `json:"status"`
	TotalCents  int64      `json:"total_cents"`
	ItemsCount  int        `json:"items_count"`
	Note        *string    `json:"note,omitempty"`
	CreatedBy   *int64     `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ReceivedAt  *time.Time `json:"received_at,omitempty"`
	ReceivedBy  *int64     `json:"received_by,omitempty"`
}

type PurchaseItem struct {
	ID             int64     `json:"id"`
	POID           int64     `json:"po_id"`
	ProductID      int64     `json:"product_id"`
	ProductName    string    `json:"product_name"`
	QtyMilli       int64     `json:"qty_milli"`
	UnitCostCents  int64     `json:"unit_cost_cents"`
	LineTotalCents int64     `json:"line_total_cents"`
	CreatedAt      time.Time `json:"created_at"`
}

// ── Requests ─────────────────────────────────────────────────────────────────

type CreatePOItemRequest struct {
	ProductID     int64 `json:"product_id" binding:"required,gt=0"`
	QtyMilli      int64 `json:"qty_milli" binding:"required,gt=0"`
	UnitCostCents int64 `json:"unit_cost_cents" binding:"required,gte=0"`
}

type CreatePORequest struct {
	SupplierID  int64                `json:"supplier_id" binding:"required,gt=0"`
	WarehouseID int64                `json:"warehouse_id" binding:"required,gt=0"`
	Items       []CreatePOItemRequest `json:"items" binding:"required,min=1,dive"`
	Note        *string              `json:"note"`
}

type AddPaymentRequest struct {
	PaymentTypeID int64   `json:"payment_type_id" binding:"required,gt=0"`
	AmountCents   int64   `json:"amount_cents" binding:"required,gt=0"`
	Note          *string `json:"note"`
}

// ── Responses ─────────────────────────────────────────────────────────────────

type PODetail struct {
	PurchaseOrder
	Items         []PurchaseItem `json:"items"`
	TransactionID int64          `json:"transaction_id"`
}

type SupplierDebtRow struct {
	SupplierID   int64  `json:"supplier_id"`
	SupplierName string `json:"supplier_name"`
	TotalCents   int64  `json:"total_cents"`
	PaidCents    int64  `json:"paid_cents"`
	DebtCents    int64  `json:"debt_cents"`
}
