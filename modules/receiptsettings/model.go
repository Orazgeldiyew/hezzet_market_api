package receiptsettings

// ReceiptSettings is the merged result (DB overrides + env defaults).
type ReceiptSettings struct {
	ShopName    string `json:"shop_name"`
	ShopAddress string `json:"shop_address"`
	ShopPhone   string `json:"shop_phone"`
	LogoPath    string `json:"logo_path"`
	LogoWidth   string `json:"logo_width"`
	LogoHeight  string `json:"logo_height"`
	Footer      string `json:"footer"`
	Template    string `json:"template"`
	DeleteCode  string `json:"delete_code"`
}

// UpdateRequest holds optional fields. Non-nil means "set this field".
// Empty string means "clear override, revert to default".
type UpdateRequest struct {
	ShopName    *string `json:"shop_name"`
	ShopAddress *string `json:"shop_address"`
	ShopPhone   *string `json:"shop_phone"`
	LogoPath    *string `json:"logo_path"`
	LogoWidth   *string `json:"logo_width"`
	LogoHeight  *string `json:"logo_height"`
	Footer      *string `json:"footer"`
	Template    *string `json:"template"`
	DeleteCode  *string `json:"delete_code"`
}

// Defaults holds server-level defaults loaded from env vars.
type Defaults struct {
	ShopName    string
	ShopAddress string
	ShopPhone   string
	Footer      string
}
