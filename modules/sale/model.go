package sale

import "time"

// ── Domain ───────────────────────────────────────────────────────────────────
type SaleStatus string

const (
	SaleStatusDraft     SaleStatus = "DRAFT"
	SaleStatusCompleted SaleStatus = "COMPLETED"
	SaleStatusCancelled SaleStatus = "CANCELLED"
)

type Sale struct {
	ID              int64      `json:"id"`
	WarehouseID     int64      `json:"warehouse_id"`
	CustomerID      *int64     `json:"customer_id,omitempty"`
	WorkerID        *int64     `json:"worker_id,omitempty"`
	TotalCents      int64      `json:"total_cents"`
	CostCents       int64      `json:"cost_cents"`
	BonusUsedCents  int64      `json:"bonus_used_cents"`
	DiscountPercent int        `json:"discount_percent"`
	DiscountCents   int64      `json:"discount_cents"`
	ItemsCount      int        `json:"items_count"`
	Note            *string    `json:"note,omitempty"`
	CreatedBy       *int64     `json:"created_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	Status          SaleStatus `json:"status"`
	CompletedAt     *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CancelledAt     *time.Time `json:"cancelled_at,omitempty" db:"cancelled_at"`
	CashierID       *int64     `json:"cashier_id,omitempty"`
	RegisterID      *int64     `json:"register_id,omitempty"`
	SaleNumber      string     `json:"sale_number"`
	CreatedByUserID int64      `json:"created_by_user_id"`
	OwnerUserID     int64      `json:"owner_user_id"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ClientSessionID *string    `json:"client_session_id,omitempty"`
}

type SaleItem struct {
	ID              int64     `json:"id"`
	SaleID          int64     `json:"sale_id"`
	ProductID       int64     `json:"product_id"`
	ProductName     string    `json:"product_name"`
	ProductPhotoURL *string   `json:"product_photo_url,omitempty"`
	QtyMilli        int64     `json:"qty_milli"`
	UnitPriceCents  int64     `json:"unit_price_cents"`
	CostCents       int64     `json:"cost_cents"`
	LineTotalCents  int64     `json:"line_total_cents"`
	DiscountPercent int       `json:"discount_percent"`
	CreatedAt       time.Time `json:"created_at"`
}

// ── Requests ─────────────────────────────────────────────────────────────────

type CreateSaleItemRequest struct {
	ProductID       int64 `json:"product_id" binding:"required,gt=0"`
	QtyMilli        int64 `json:"qty_milli" binding:"required,gt=0"`
	DiscountPercent int   `json:"discount_percent"`
}

type CreateSaleRequest struct {
	WarehouseID int64                   `json:"warehouse_id" binding:"required,gt=0"`
	CustomerID  *int64                  `json:"customer_id"`
	WorkerID    *int64                  `json:"worker_id"`
	Items       []CreateSaleItemRequest `json:"items" binding:"required,min=1,dive"`
	Note        *string                 `json:"note"`
	// Force allows selling even when available stock is insufficient (deficit sale).
	Force bool `json:"force"`
}

type ConfirmSaleRequest struct {
	PaymentTypeID   *int64  `json:"payment_type_id"`
	PaymentAmount   *int64  `json:"payment_amount"`
	PaymentNote     *string `json:"payment_note"`
	BonusUsedCents  *int64  `json:"bonus_used_cents"`
	DiscountPercent *int    `json:"discount_percent"`
	// Force allows confirming even when physical stock is insufficient (deficit sale).
	// The warehouse item qty_milli will go negative.
	Force bool `json:"force"`
}

type TransferSaleRequest struct {
	CashierID int64 `json:"cashier_id" binding:"required,gt=0"`
}

type ReturnItemRequest struct {
	SaleItemID int64 `json:"sale_item_id" binding:"required,gt=0"`
	QtyMilli   int64 `json:"qty_milli"    binding:"required,gt=0"`
}

type ReturnSaleRequest struct {
	Items  []ReturnItemRequest `json:"items"`
	Reason string              `json:"reason"`
}

// ── Responses ────────────────────────────────────────────────────────────────

type SaleDetail struct {
	Sale          Sale       `json:"sale"`
	Items         []SaleItem `json:"items"`
	TransactionID int64      `json:"transaction_id"`
}

type SaleListItem struct {
	Sale
	CustomerName *string `json:"customer_name,omitempty"`
}

type SaleListResult struct {
	Items  []SaleListItem `json:"items"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}
