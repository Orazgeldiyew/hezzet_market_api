package product

import (
	"time"
)

type Product struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	SKU           string    `json:"sku"`
	Barcodes      []string  `json:"barcodes"`
	Unit          string    `json:"unit"`
	PurchasePrice int64     `json:"purchase_price"`
	SalePrice     int64     `json:"sale_price"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	UnitType      string    `json:"unit_type"`  // piece|kg|liter|meter|box
	UnitScale     int       `json:"unit_scale"` // 1000
}

type CreateRequest struct {
	Name          string   `json:"name" binding:"required"`
	SKU           string   `json:"sku"`
	Barcodes      []string `json:"barcodes"`
	Unit          string   `json:"unit" binding:"required"`
	PurchasePrice int64    `json:"purchase_price"`
	SalePrice     int64    `json:"sale_price"`
	IsActive      *bool    `json:"is_active"`
	CategoryIDs   []int64  `json:"category_ids"`
	UnitType      *string  `json:"unit_type"` // optional, default piece
}

type UpdateRequest struct {
	Name          *string   `json:"name"`
	SKU           *string   `json:"sku"`
	Barcodes      *[]string `json:"barcodes"`
	Unit          *string   `json:"unit"`
	PurchasePrice *int64    `json:"purchase_price"`
	SalePrice     *int64    `json:"sale_price"`
	IsActive      *bool     `json:"is_active"`
	CategoryIDs   *[]int64  `json:"category_ids"`
	UnitType      *string   `json:"unit_type"`
}

// ListResponse for paginated product list
type ListResponse struct {
	Items  []Product `json:"items"`
	Total  int       `json:"total"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}

type Card struct {
	Product    Product         `json:"product"`
	Stock      int64           `json:"stock"`
	Tags       []string        `json:"tags"`
	Suppliers  []int64         `json:"supplier_ids"`
	Categories []CategoryBrief `json:"categories"`
}

type CategoryBrief struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
type SetCategoriesRequest struct {
	CategoryIDs []int64 `json:"category_ids" binding:"required"`
}
