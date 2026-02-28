package sale

import "time"

// ── Domain ───────────────────────────────────────────────────────────────────

type Sale struct {
	ID          int64     `json:"id"`
	WarehouseID int64     `json:"warehouse_id"`
	CustomerID  *int64    `json:"customer_id,omitempty"`
	TotalCents  int64     `json:"total_cents"`
	CostCents   int64     `json:"cost_cents"`
	ItemsCount  int       `json:"items_count"`
	Note        *string   `json:"note,omitempty"`
	CreatedBy   *int64    `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"`
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
	CreatedAt       time.Time `json:"created_at"`
}

// ── Requests ─────────────────────────────────────────────────────────────────

type CreateSaleItemRequest struct {
	ProductID int64 `json:"product_id" binding:"required,gt=0"`
	QtyMilli  int64 `json:"qty_milli" binding:"required,gt=0"`
}

type CreateSaleRequest struct {
	WarehouseID int64                   `json:"warehouse_id" binding:"required,gt=0"`
	CustomerID  *int64                  `json:"customer_id"`
	Items       []CreateSaleItemRequest `json:"items" binding:"required,min=1,dive"`
	Note        *string                 `json:"note"`
	// Force allows selling even when available stock is insufficient (deficit sale).
	Force bool `json:"force"`
}

type ConfirmSaleRequest struct {
	PaymentTypeID *int64  `json:"payment_type_id"`
	PaymentAmount *int64  `json:"payment_amount"`
	PaymentNote   *string `json:"payment_note"`
	// Force allows confirming even when physical stock is insufficient (deficit sale).
	// The warehouse item qty_milli will go negative.
	Force bool `json:"force"`
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
