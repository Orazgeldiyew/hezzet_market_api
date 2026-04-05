package inventory

import "time"

type InventoryCount struct {
	ID            int64      `json:"id"`
	WarehouseID   int64      `json:"warehouse_id"`
	WarehouseName string     `json:"warehouse_name,omitempty"`
	Status        string     `json:"status"`
	Note          *string    `json:"note,omitempty"`
	CreatedBy     *int64     `json:"created_by,omitempty"`
	CreatedByName string     `json:"created_by_name,omitempty"`
	ConfirmedBy   *int64     `json:"confirmed_by,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	ConfirmedAt   *time.Time `json:"confirmed_at,omitempty"`
}

type InventoryCountItem struct {
	ID             int64  `json:"id"`
	CountID        int64  `json:"count_id"`
	ProductID      int64  `json:"product_id"`
	ProductName    string `json:"product_name"`
	SystemQtyMilli int64  `json:"system_qty_milli"`
	ActualQtyMilli int64  `json:"actual_qty_milli"`
	DiffMilli      int64  `json:"diff_milli"`
}

type InventoryDetail struct {
	InventoryCount
	Items []InventoryCountItem `json:"items"`
}

type CreateRequest struct {
	WarehouseID int64   `json:"warehouse_id" binding:"required,gt=0"`
	Note        *string `json:"note"`
}

type UpdateItemRequest struct {
	ProductID      int64 `json:"product_id" binding:"required,gt=0"`
	ActualQtyMilli int64 `json:"actual_qty_milli"`
}

type UpdateItemsRequest struct {
	Items []UpdateItemRequest `json:"items" binding:"required,min=1,dive"`
}
