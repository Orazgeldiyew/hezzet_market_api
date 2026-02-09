package product

import "time"

type Product struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	SKU           string    `json:"sku"`
	Barcode       string    `json:"barcode"`
	Unit          string    `json:"unit"`
	PurchasePrice float64   `json:"purchase_price"`
	SalePrice     float64   `json:"sale_price"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Name          string  `json:"name" binding:"required"`
	SKU           string  `json:"sku"`
	Barcode       string  `json:"barcode"`
	Unit          string  `json:"unit" binding:"required"`
	PurchasePrice float64 `json:"purchase_price"`
	SalePrice     float64 `json:"sale_price"`
	IsActive      *bool   `json:"is_active"`
}

type UpdateRequest struct {
	Name          *string  `json:"name"`
	SKU           *string  `json:"sku"`
	Barcode       *string  `json:"barcode"`
	Unit          *string  `json:"unit"`
	PurchasePrice *float64 `json:"purchase_price"`
	SalePrice     *float64 `json:"sale_price"`
	IsActive      *bool    `json:"is_active"`
}

type Card struct {
	Product    Product         `json:"product"`
	Stock      float64         `json:"stock"`
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
