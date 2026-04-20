package reports

import "time"

// PeriodStats holds revenue/profit/orders for one period + % change vs previous.
type PeriodStats struct {
	RevenueCents int64   `json:"revenue_cents"`
	ProfitCents  int64   `json:"profit_cents"`
	Orders       int     `json:"orders"`
	ChangePct    float64 `json:"change_pct"` // revenue change vs previous same period
}

// DashboardStats is the payload for GET /api/reports/dashboard.
type DashboardStats struct {
	Today          PeriodStats  `json:"today"`   // today vs yesterday
	Week           PeriodStats  `json:"week"`    // this week vs last week
	Month          PeriodStats  `json:"month"`   // this month vs last month
	LowStockCount  int          `json:"low_stock_count"`
	PendingPayroll int          `json:"pending_payroll"`
	TopProducts    []TopProduct `json:"top_products"` // top-5 by revenue this week
}

// TopProduct is one entry in DashboardStats.TopProducts.
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

// ReorderSuggestion is one product/warehouse line in the reorder report.
// DaysUntilStockout is nil when the product has no sales history (can't project).
type ReorderSuggestion struct {
	ProductID           int64    `json:"product_id"`
	Name                string   `json:"name"`
	WarehouseID         int64    `json:"warehouse_id"`
	WarehouseName       string   `json:"warehouse_name"`
	CurrentQtyMilli     int64    `json:"current_qty_milli"`
	AvgDailyMilli       float64  `json:"avg_daily_milli"`
	LeadTimeDays        int      `json:"lead_time_days"`
	SafetyStockMilli    int64    `json:"safety_stock_milli"`
	ReorderPointMilli   int64    `json:"reorder_point_milli"`
	SuggestedOrderMilli int64    `json:"suggested_order_milli"`
	DaysUntilStockout   *float64 `json:"days_until_stockout"`
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
