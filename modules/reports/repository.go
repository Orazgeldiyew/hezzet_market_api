package reports

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db             *pgxpool.Pool
	lowStockThresh int64 // in milli-units (LowStockDefault * 1000)
}

func NewRepository(db *pgxpool.Pool, lowStockDefault int64) *Repository {
	return &Repository{db: db, lowStockThresh: lowStockDefault * 1000}
}

// Dashboard returns sales stats for today/week/month with comparison to previous
// periods, low-stock count, pending payroll, and top-5 products this week.
func (r *Repository) Dashboard(ctx context.Context) (*DashboardStats, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterdayStart := todayStart.AddDate(0, 0, -1)

	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := todayStart.AddDate(0, 0, -(weekday - 1))
	lastWeekStart := weekStart.AddDate(0, 0, -7)

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastMonthStart := monthStart.AddDate(0, -1, 0)

	stats := &DashboardStats{}

	// 1. All period aggregations in one query (confirmed sales only).
	//    Boundaries: $1=todayStart $2=yesterdayStart $3=weekStart
	//                $4=lastWeekStart $5=monthStart $6=lastMonthStart
	var (
		todayRev, todayCost         int64
		yesterdayRev, yesterdayCost int64
		weekRev, weekCost           int64
		lastWeekRev, lastWeekCost   int64
		monthRev, monthCost         int64
		lastMonthRev, lastMonthCost int64
		todayOrders, yesterdayOrders,
		weekOrders, lastWeekOrders,
		monthOrders, lastMonthOrders int
	)
	err := r.db.QueryRow(ctx, `
		SELECT
		  COALESCE(SUM(CASE WHEN created_at >= $1                        THEN total_cents END), 0),
		  COALESCE(SUM(CASE WHEN created_at >= $1                        THEN cost_cents  END), 0),
		  COUNT(   CASE WHEN created_at >= $1                            THEN 1           END),
		  COALESCE(SUM(CASE WHEN created_at >= $2 AND created_at < $1    THEN total_cents END), 0),
		  COALESCE(SUM(CASE WHEN created_at >= $2 AND created_at < $1    THEN cost_cents  END), 0),
		  COUNT(   CASE WHEN created_at >= $2 AND created_at < $1        THEN 1           END),
		  COALESCE(SUM(CASE WHEN created_at >= $3                        THEN total_cents END), 0),
		  COALESCE(SUM(CASE WHEN created_at >= $3                        THEN cost_cents  END), 0),
		  COUNT(   CASE WHEN created_at >= $3                            THEN 1           END),
		  COALESCE(SUM(CASE WHEN created_at >= $4 AND created_at < $3    THEN total_cents END), 0),
		  COALESCE(SUM(CASE WHEN created_at >= $4 AND created_at < $3    THEN cost_cents  END), 0),
		  COUNT(   CASE WHEN created_at >= $4 AND created_at < $3        THEN 1           END),
		  COALESCE(SUM(CASE WHEN created_at >= $5                        THEN total_cents END), 0),
		  COALESCE(SUM(CASE WHEN created_at >= $5                        THEN cost_cents  END), 0),
		  COUNT(   CASE WHEN created_at >= $5                            THEN 1           END),
		  COALESCE(SUM(CASE WHEN created_at >= $6 AND created_at < $5    THEN total_cents END), 0),
		  COALESCE(SUM(CASE WHEN created_at >= $6 AND created_at < $5    THEN cost_cents  END), 0),
		  COUNT(   CASE WHEN created_at >= $6 AND created_at < $5        THEN 1           END)
		FROM sales
		WHERE status = 'confirmed' AND created_at >= $6`,
		todayStart, yesterdayStart, weekStart, lastWeekStart, monthStart, lastMonthStart,
	).Scan(
		&todayRev, &todayCost, &todayOrders,
		&yesterdayRev, &yesterdayCost, &yesterdayOrders,
		&weekRev, &weekCost, &weekOrders,
		&lastWeekRev, &lastWeekCost, &lastWeekOrders,
		&monthRev, &monthCost, &monthOrders,
		&lastMonthRev, &lastMonthCost, &lastMonthOrders,
	)
	if err != nil {
		return nil, err
	}

	stats.Today = PeriodStats{
		RevenueCents: todayRev,
		ProfitCents:  todayRev - todayCost,
		Orders:       todayOrders,
		ChangePct:    changePct(todayRev, yesterdayRev),
	}
	stats.Week = PeriodStats{
		RevenueCents: weekRev,
		ProfitCents:  weekRev - weekCost,
		Orders:       weekOrders,
		ChangePct:    changePct(weekRev, lastWeekRev),
	}
	stats.Month = PeriodStats{
		RevenueCents: monthRev,
		ProfitCents:  monthRev - monthCost,
		Orders:       monthOrders,
		ChangePct:    changePct(monthRev, lastMonthRev),
	}

	// 2. Low-stock count
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM warehouse_items wi
		 JOIN products p ON p.id = wi.product_id
		 WHERE wi.qty_milli < $1 AND p.is_active = true`,
		r.lowStockThresh,
	).Scan(&stats.LowStockCount); err != nil {
		return nil, err
	}

	// 3. Pending payroll
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM payroll_runs WHERE status = 'calculated'`,
	).Scan(&stats.PendingPayroll); err != nil {
		return nil, err
	}

	// 4. Top 5 products by revenue this week
	rows, err := r.db.Query(ctx,
		`SELECT p.id, p.name, SUM(si.line_total_cents) AS rev
		 FROM sale_items si
		 JOIN products p ON p.id = si.product_id
		 JOIN sales    s ON s.id = si.sale_id
		 WHERE s.created_at >= $1 AND s.status = 'confirmed'
		 GROUP BY p.id, p.name
		 ORDER BY rev DESC
		 LIMIT 5`,
		weekStart,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats.TopProducts = []TopProduct{}
	for rows.Next() {
		var tp TopProduct
		if err := rows.Scan(&tp.ProductID, &tp.Name, &tp.RevenueCents); err != nil {
			return nil, err
		}
		stats.TopProducts = append(stats.TopProducts, tp)
	}
	return stats, rows.Err()
}

// changePct returns revenue percentage change: (current - previous) / previous * 100.
// Returns +100 if previous was 0 but current > 0; 0 if both are 0.
func changePct(current, previous int64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100
		}
		return 0
	}
	return float64(current-previous) / float64(previous) * 100
}

// validGroupBy is the whitelist for group_by values to prevent SQL injection.
var validGroupBy = map[string]bool{"day": true, "week": true, "month": true}

// SalesByPeriod returns aggregated sales grouped by the given time unit.
// groupBy must be "day", "week", or "month".
func (r *Repository) SalesByPeriod(
	ctx context.Context,
	groupBy string,
	from, to time.Time,
	warehouseID *int64,
	customerType string,
) ([]SalesPeriodRow, error) {
	if !validGroupBy[groupBy] {
		return nil, fmt.Errorf("invalid group_by: %q", groupBy)
	}

	var custType *string
	if customerType != "" {
		custType = &customerType
	}

	// group_by is validated against the whitelist above — safe to interpolate.
	q := fmt.Sprintf(`
		SELECT DATE_TRUNC('%s', s.created_at) AS period,
		       COUNT(*)                        AS orders,
		       SUM(s.total_cents)              AS revenue,
		       SUM(s.cost_cents)               AS cost,
		       SUM(s.total_cents - s.cost_cents) AS profit
		FROM sales s
		LEFT JOIN customers c ON c.id = s.customer_id
		WHERE s.created_at BETWEEN $1 AND $2
		  AND s.status = 'confirmed'
		  AND ($3::bigint IS NULL OR s.warehouse_id = $3)
		  AND ($4::text   IS NULL OR c.type::text = $4)
		GROUP BY 1
		ORDER BY 1`, groupBy)

	rows, err := r.db.Query(ctx, q, from, to, warehouseID, custType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SalesPeriodRow
	for rows.Next() {
		var row SalesPeriodRow
		if err := rows.Scan(&row.Period, &row.Orders, &row.RevenueCents, &row.CostCents, &row.ProfitCents); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if out == nil {
		out = []SalesPeriodRow{}
	}
	return out, rows.Err()
}

// SalesByProduct returns per-product sales aggregation over the given date range.
func (r *Repository) SalesByProduct(
	ctx context.Context,
	from, to time.Time,
	warehouseID *int64,
) ([]SalesProductRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.id, p.name,
		       SUM(si.qty_milli),
		       SUM(si.line_total_cents),
		       SUM(si.cost_cents),
		       SUM(si.line_total_cents - si.cost_cents),
		       COALESCE(ROUND(
		           100.0 * SUM(si.line_total_cents - si.cost_cents)::numeric
		                 / NULLIF(SUM(si.line_total_cents), 0), 2
		       ), 0)
		FROM sale_items si
		JOIN products p ON p.id = si.product_id
		JOIN sales    s ON s.id = si.sale_id
		WHERE s.created_at BETWEEN $1 AND $2
		  AND s.status = 'confirmed'
		  AND ($3::bigint IS NULL OR s.warehouse_id = $3)
		GROUP BY p.id, p.name
		ORDER BY SUM(si.line_total_cents) DESC`,
		from, to, warehouseID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SalesProductRow
	for rows.Next() {
		var row SalesProductRow
		if err := rows.Scan(
			&row.ProductID, &row.Name, &row.QtyMilli,
			&row.RevenueCents, &row.CostCents, &row.ProfitCents, &row.MarginPct,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if out == nil {
		out = []SalesProductRow{}
	}
	return out, rows.Err()
}

// StockMovementsForExport returns stock ledger rows for the Excel export (max 10 000).
func (r *Repository) StockMovementsForExport(
	ctx context.Context,
	from, to time.Time,
	warehouseID *int64,
) ([]StockMovementExportRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT wid.created_at, w.name, p.name,
		       wid.delta_milli, wid.type::text, wid.price_cents, u.username
		FROM warehouse_item_details wid
		JOIN warehouses w ON w.id = wid.warehouse_id
		JOIN products   p ON p.id = wid.product_id
		LEFT JOIN users u ON u.id = wid.created_by
		WHERE wid.created_at BETWEEN $1 AND $2
		  AND ($3::bigint IS NULL OR wid.warehouse_id = $3)
		ORDER BY wid.created_at DESC
		LIMIT 10000`,
		from, to, warehouseID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StockMovementExportRow
	for rows.Next() {
		var row StockMovementExportRow
		if err := rows.Scan(
			&row.CreatedAt, &row.Warehouse, &row.Product,
			&row.DeltaMilli, &row.MovementType, &row.PriceCents, &row.CreatedBy,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if out == nil {
		out = []StockMovementExportRow{}
	}
	return out, rows.Err()
}
