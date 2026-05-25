package receiptsettings

// ReceiptSettings is the merged result (DB overrides + env defaults).
type ReceiptSettings struct {
	ShopName     string `json:"shop_name"`
	ShopAddress  string `json:"shop_address"`
	ShopPhone    string `json:"shop_phone"`
	LogoPath     string `json:"logo_path"`
	LogoWidth    string `json:"logo_width"`
	LogoHeight   string `json:"logo_height"`
	Footer       string `json:"footer"`
	Template     string `json:"template"`
	DeleteCode   string `json:"delete_code"`
	BonusPercent int    `json:"bonus_percent"`
	// Thermal-receipt font sizes in pixels (used by printer/html_render.go
	// as @media-print overrides). Range 10–80 enforced at the DB layer.
	FontSizeHeader int `json:"font_size_header"`
	FontSizeItems  int `json:"font_size_items"`
	FontSizeMeta   int `json:"font_size_meta"`
	FontSizeTotal  int `json:"font_size_total"`
	// Legal entity details for B2B documents (purchase invoices).
	LegalName string `json:"legal_name"`
	TaxID     string `json:"tax_id"`
}

// UpdateRequest holds optional fields. Non-nil means "set this field".
// Empty string means "clear override, revert to default".
type UpdateRequest struct {
	ShopName       *string `json:"shop_name"`
	ShopAddress    *string `json:"shop_address"`
	ShopPhone      *string `json:"shop_phone"`
	LogoPath       *string `json:"logo_path"`
	LogoWidth      *string `json:"logo_width"`
	LogoHeight     *string `json:"logo_height"`
	Footer         *string `json:"footer"`
	Template       *string `json:"template"`
	DeleteCode     *string `json:"delete_code"`
	BonusPercent   *int    `json:"bonus_percent"`
	FontSizeHeader *int    `json:"font_size_header" binding:"omitempty,min=10,max=80"`
	FontSizeItems  *int    `json:"font_size_items"  binding:"omitempty,min=10,max=80"`
	FontSizeMeta   *int    `json:"font_size_meta"   binding:"omitempty,min=10,max=80"`
	FontSizeTotal  *int    `json:"font_size_total"  binding:"omitempty,min=10,max=80"`
	LegalName      *string `json:"legal_name"       binding:"omitempty,max=255"`
	TaxID          *string `json:"tax_id"           binding:"omitempty,max=64"`
}

// Defaults holds server-level defaults loaded from env vars.
type Defaults struct {
	ShopName    string
	ShopAddress string
	ShopPhone   string
	Footer      string
}
