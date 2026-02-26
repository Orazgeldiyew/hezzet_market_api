package product

import "time"

// ─── Measurement enums ───────────────────────────────────────────────────────

// UnitType is the measurement category stored in products.unit_type (unit_type_enum).
type UnitType string

const (
	UnitTypePiece  UnitType = "piece"  // countable items
	UnitTypeWeight UnitType = "weight" // mass-based items (kg, g)
	UnitTypeVolume UnitType = "volume" // liquid items (l, ml)
)

// Unit is the specific measurement unit stored in products.unit (unit_enum).
type Unit string

const (
	UnitPiece Unit = "piece"
	UnitKg    Unit = "kg"
	UnitG     Unit = "g"
	UnitL     Unit = "l"
	UnitML    Unit = "ml"
)

// ─── Domain model ────────────────────────────────────────────────────────────

type Product struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	SKU           string    `json:"sku"`
	Barcodes      []string  `json:"barcodes"`
	UnitType      UnitType  `json:"unit_type"`  // piece | weight | volume
	Unit          Unit      `json:"unit"`        // piece | kg | g | l | ml
	UnitScale     int       `json:"unit_scale"`  // always 1000 (1 display unit = 1000 milli-units)
	PurchasePrice int64     `json:"purchase_price"`
	SalePrice     int64     `json:"sale_price"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ─── Requests ────────────────────────────────────────────────────────────────

type CreateRequest struct {
	Name          string   `json:"name"           binding:"required"`
	SKU           string   `json:"sku"`
	Barcodes      []string `json:"barcodes"`
	UnitType      UnitType `json:"unit_type"      binding:"required,oneof=piece weight volume"`
	Unit          Unit     `json:"unit"           binding:"required,oneof=piece kg g l ml"`
	PurchasePrice int64    `json:"purchase_price"`
	SalePrice     int64    `json:"sale_price"`
	IsActive      *bool    `json:"is_active"`
	CategoryIDs   []int64  `json:"category_ids"`
}

type UpdateRequest struct {
	Name          *string   `json:"name"`
	SKU           *string   `json:"sku"`
	Barcodes      *[]string `json:"barcodes"`
	UnitType      *UnitType `json:"unit_type"  binding:"omitempty,oneof=piece weight volume"`
	Unit          *Unit     `json:"unit"       binding:"omitempty,oneof=piece kg g l ml"`
	PurchasePrice *int64    `json:"purchase_price"`
	SalePrice     *int64    `json:"sale_price"`
	IsActive      *bool     `json:"is_active"`
	CategoryIDs   *[]int64  `json:"category_ids"`
}

// ─── Responses ───────────────────────────────────────────────────────────────

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
