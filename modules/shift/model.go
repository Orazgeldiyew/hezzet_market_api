package shift

import "time"

type CashRegister struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type Shift struct {
	ID           int64      `json:"id"`
	RegisterID   int64      `json:"register_id"`
	RegisterName string     `json:"register_name,omitempty"`
	UserID       int64      `json:"user_id"`
	CashierName  string     `json:"cashier_name,omitempty"`
	OpenedAt     time.Time  `json:"opened_at"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
	OpeningCash  int64      `json:"opening_cash"`
	ClosingCash  *int64     `json:"closing_cash,omitempty"`
	ExpectedCash *int64     `json:"expected_cash,omitempty"`
	Difference   *int64     `json:"difference,omitempty"`
	SalesCount   int        `json:"sales_count"`
	SalesTotal   int64      `json:"sales_total"`
	ReturnsTotal int64      `json:"returns_total"`
	Status       string     `json:"status"`
	Note         *string    `json:"note,omitempty"`
	ClosedBy     *int64     `json:"closed_by,omitempty"`
}

type OpenRequest struct {
	RegisterID  int64 `json:"register_id" binding:"required,gt=0"`
	OpeningCash int64 `json:"opening_cash"`
}

type CloseRequest struct {
	ClosingCash int64   `json:"closing_cash"`
	Note        *string `json:"note"`
}
