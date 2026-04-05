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
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	SKU             string    `json:"sku"`
	Barcodes        []string  `json:"barcodes"`
	UnitType        UnitType  `json:"unit_type"`  // piece | weight | volume
	Unit            Unit      `json:"unit"`        // piece | kg | g | l | ml
	UnitScale       int       `json:"unit_scale"`  // always 1000 (1 display unit = 1000 milli-units)
	PurchasePrice   int64     `json:"purchase_price"`
	SalePrice       int64     `json:"sale_price"`
	DiscountPercent int       `json:"discount_percent"` // centralized discount set by admin/manager
	IsActive        bool      `json:"is_active"`
	PhotoURL        *string   `json:"photo_url,omitempty"` // public URL, computed
	PhotoPath       *string   `json:"-"`                   // raw DB value, used internally
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
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
	Name            *string   `json:"name"`
	SKU             *string   `json:"sku"`
	Barcodes        *[]string `json:"barcodes"`
	UnitType        *UnitType `json:"unit_type"  binding:"omitempty,oneof=piece weight volume"`
	Unit            *Unit     `json:"unit"       binding:"omitempty,oneof=piece kg g l ml"`
	PurchasePrice   *int64    `json:"purchase_price"`
	SalePrice       *int64    `json:"sale_price"`
	DiscountPercent *int      `json:"discount_percent"`
	IsActive        *bool     `json:"is_active"`
	CategoryIDs     *[]int64  `json:"category_ids"`
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

// BarcodeResult is returned by GetByBarcode — includes parsed weight for scale barcodes.
type BarcodeResult struct {
	Product  Product `json:"product"`
	QtyMilli int64   `json:"qty_milli,omitempty"` // parsed weight in milli-units (0 = normal barcode)
	IsWeight bool    `json:"is_weight"`           // true if parsed from weight barcode
}

// ─── Price history ──────────────────────────────────────────────────────────

type PriceHistory struct {
	ID            int64     `json:"id"`
	ProductID     int64     `json:"product_id"`
	Field         string    `json:"field"`          // "purchase_price" or "sale_price"
	OldValue      int64     `json:"old_value"`
	NewValue      int64     `json:"new_value"`
	ChangedBy     *int64    `json:"changed_by,omitempty"`
	ChangedByName string    `json:"changed_by_name,omitempty"`
	ChangedAt     time.Time `json:"changed_at"`
}
