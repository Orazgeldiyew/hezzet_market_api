package supplierreturn

import "time"

// ── Domain ──────────────────────────────────────────────────────────────────

type SupplierReturn struct {
	ID          int64      `json:"id"`
	SupplierID  int64      `json:"supplier_id"`
	WarehouseID int64      `json:"warehouse_id"`
	Status      string     `json:"status"`
	TotalCents  int64      `json:"total_cents"`
	ItemsCount  int        `json:"items_count"`
	Note        *string    `json:"note,omitempty"`
	CreatedBy   *int64     `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
	ConfirmedBy *int64     `json:"confirmed_by,omitempty"`
}

type ReturnItem struct {
	ID             int64     `json:"id"`
	ReturnID       int64     `json:"return_id"`
	ProductID      int64     `json:"product_id"`
	ProductName    string    `json:"product_name"`
	QtyMilli       int64     `json:"qty_milli"`
	UnitCostCents  int64     `json:"unit_cost_cents"`
	LineTotalCents int64     `json:"line_total_cents"`
	CreatedAt      time.Time `json:"created_at"`
}

// ── Requests ────────────────────────────────────────────────────────────────

type CreateItemRequest struct {
	ProductID     int64 `json:"product_id" binding:"required,gt=0"`
	QtyMilli      int64 `json:"qty_milli" binding:"required,gt=0"`
	UnitCostCents int64 `json:"unit_cost_cents" binding:"required,gte=0"`
}

type CreateRequest struct {
	SupplierID  int64               `json:"supplier_id" binding:"required,gt=0"`
	WarehouseID int64               `json:"warehouse_id" binding:"required,gt=0"`
	Items       []CreateItemRequest `json:"items" binding:"required,min=1,dive"`
	Note        *string             `json:"note"`
}

// ── Responses ───────────────────────────────────────────────────────────────

type ReturnDetail struct {
	SupplierReturn
	Items []ReturnItem `json:"items"`
}

type ReturnListItem struct {
	SupplierReturn
	SupplierName   string `json:"supplier_name"`
	WarehouseName  string `json:"warehouse_name"`
	CreatedByName  string `json:"created_by_name"`
	ConfirmedByName string `json:"confirmed_by_name,omitempty"`
}
