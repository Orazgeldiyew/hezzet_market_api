package stock

import "time"

// ---------- Domain ----------

type WarehouseItemDetail struct {
	ID             int64     `json:"id"`
	IdempotencyKey string    `json:"idempotency_key"`
	WarehouseID    int64     `json:"warehouse_id"`
	ProductID      int64     `json:"product_id"`
	DeltaMilli     int64     `json:"delta_milli"`
	Type           string    `json:"type"` // movement_type enum in DB
	PriceCents     *int64    `json:"price_cents,omitempty"`
	WorkerID       *int64    `json:"worker_id,omitempty"`
	CreatedBy      *int64    `json:"created_by,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	SupplierID     *int64    `json:"supplier_id,omitempty"`
	Note           *string   `json:"note,omitempty"`
}

type WarehouseItem struct {
	WarehouseID    int64     `json:"warehouse_id"`
	ProductID      int64     `json:"product_id"`
	QtyMilli       int64     `json:"qty_milli"`
	AvgCostCents   int64     `json:"avg_cost_cents"`   // средняя себестоимость за 1.000 единицу
	TotalCostCents int64     `json:"total_cost_cents"` // себестоимость всего остатка
	UpdatedAt      time.Time `json:"updated_at"`
	AvailableMilli int64     `json:"available_milli"` // qty_milli minus active reservations
}
type OpeningBalanceRequest struct {
	WarehouseID    int64  `json:"warehouse_id" binding:"required,gt=0"`
	ProductID      int64  `json:"product_id" binding:"required,gt=0"`
	QtyMilli       int64  `json:"qty_milli" binding:"required,gt=0"`

	// ВАЖНО: price_cents = цена за 1.000 единицу (за 1 “unit” в твоей системе SCALE=1000)
	// пример: qty_milli=1000 и price_cents=5000 => партия стоила 5000 центов
	PriceCents     int64  `json:"price_cents" binding:"required,gt=0"`

	IdempotencyKey string `json:"idempotency_key" binding:"required,uuid"`
	WorkerID       *int64 `json:"worker_id"`
}



// ---------- Requests ----------

type InRequest struct {
	WarehouseID    int64  `json:"warehouse_id" binding:"required,gt=0"`
	ProductID      int64  `json:"product_id" binding:"required,gt=0"`
	QtyMilli       int64  `json:"qty_milli" binding:"required,gt=0"`
	IdempotencyKey string `json:"idempotency_key" binding:"required,uuid"`

	PriceCents *int64 `json:"price_cents"`
	WorkerID   *int64 `json:"worker_id"`
}

type OutRequest struct {
	WarehouseID    int64  `json:"warehouse_id" binding:"required,gt=0"`
	ProductID      int64  `json:"product_id" binding:"required,gt=0"`
	QtyMilli       int64  `json:"qty_milli" binding:"required,gt=0"`
	IdempotencyKey string `json:"idempotency_key" binding:"required,uuid"`

	PriceCents *int64 `json:"price_cents"`
	WorkerID   *int64 `json:"worker_id"`
}

type TransferRequest struct {
	FromWarehouseID int64  `json:"from_warehouse_id" binding:"required,gt=0"`
	ToWarehouseID   int64  `json:"to_warehouse_id" binding:"required,gt=0"`
	ProductID       int64  `json:"product_id" binding:"required,gt=0"`
	QtyMilli        int64  `json:"qty_milli" binding:"required,gt=0"`
	IdempotencyKey  string `json:"idempotency_key" binding:"required,uuid"`
}

type MoveRequest struct {
	WarehouseID    int64  `json:"warehouse_id" binding:"required,gt=0"`
	ProductID      int64  `json:"product_id" binding:"required,gt=0"`
	DeltaMilli     int64  `json:"delta_milli" binding:"required,ne=0"`
	Type           string `json:"type" binding:"required,oneof=damaged adjustment return_to_supplier"`
	IdempotencyKey string `json:"idempotency_key" binding:"required,uuid"`

	PriceCents *int64  `json:"price_cents"`
	WorkerID   *int64  `json:"worker_id"`
	SupplierID *int64  `json:"supplier_id"`
	Note       *string `json:"note"`
}

// ---------- Responses ----------

type MovementResult struct {
	Detail WarehouseItemDetail `json:"detail"`
	Item   WarehouseItem       `json:"item"`
}

type TransferResult struct {
	OutDetail WarehouseItemDetail `json:"out_detail"`
	InDetail  WarehouseItemDetail `json:"in_detail"`
	FromItem  WarehouseItem       `json:"from_item"`
	ToItem    WarehouseItem       `json:"to_item"`
}

type DetailsListResult struct {
	Items  []WarehouseItemDetail `json:"items"`
	Total  int                   `json:"total"`
	Limit  int                   `json:"limit"`
	Offset int                   `json:"offset"`
}

// ---------- Bulk Stock In ----------

type BulkInItem struct {
	ProductID      int64  `json:"product_id" binding:"required,gt=0"`
	QtyMilli       int64  `json:"qty_milli" binding:"required,gt=0"`
	PriceCents     *int64 `json:"price_cents"`
	IdempotencyKey string `json:"idempotency_key" binding:"required,uuid"`
	WorkerID       *int64 `json:"worker_id"`
}

type BulkInRequest struct {
	WarehouseID int64        `json:"warehouse_id" binding:"required,gt=0"`
	Items       []BulkInItem `json:"items" binding:"required,min=1,dive"`
}

type BulkInResult struct {
	Results []MovementResult `json:"results"`
}

// NegativeStockRow is returned by GET /api/stock/negative
type NegativeStockRow struct {
	WarehouseID   int64  `json:"warehouse_id"`
	WarehouseName string `json:"warehouse_name"`
	ProductID     int64  `json:"product_id"`
	ProductName   string `json:"product_name"`
	QtyMilli      int64  `json:"qty_milli"`      // negative value
	DeficitMilli  int64  `json:"deficit_milli"`  // abs(qty_milli)
}
