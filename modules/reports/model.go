package reports

import "time"

// DashboardStats is the payload for GET /api/reports/dashboard.
type DashboardStats struct {
	TodayRevenueCents int64        `json:"today_revenue_cents"`
	TodayProfitCents  int64        `json:"today_profit_cents"`
	TodayOrders       int          `json:"today_orders"`
	LowStockCount     int          `json:"low_stock_count"`
	PendingPayroll    int          `json:"pending_payroll"`
	TopProducts       []TopProduct `json:"top_products"`
}

// TopProduct is one entry in DashboardStats.TopProducts (top 5 by revenue this week).
type TopProduct struct {
	ProductID    int64  `json:"product_id"`
	Name         string `json:"name"`
	RevenueCents int64  `json:"revenue_cents"`
}

// SalesPeriodRow is one row in the sales-by-period report.
type SalesPeriodRow struct {
	Period       time.Time `json:"period"`
	Orders       int       `json:"orders"`
	RevenueCents int64     `json:"revenue_cents"`
	CostCents    int64     `json:"cost_cents"`
	ProfitCents  int64     `json:"profit_cents"`
}

// SalesProductRow is one row in the sales-by-product report.
type SalesProductRow struct {
	ProductID    int64   `json:"product_id"`
	Name         string  `json:"name"`
	QtyMilli     int64   `json:"qty_milli"`
	RevenueCents int64   `json:"revenue_cents"`
	CostCents    int64   `json:"cost_cents"`
	ProfitCents  int64   `json:"profit_cents"`
	MarginPct    float64 `json:"margin_pct"`
}

// StockMovementExportRow is one row in the stock-movements Excel export.
type StockMovementExportRow struct {
	CreatedAt    time.Time
	Warehouse    string
	Product      string
	DeltaMilli   int64
	MovementType string
	PriceCents   *int64
	CreatedBy    *string
}
