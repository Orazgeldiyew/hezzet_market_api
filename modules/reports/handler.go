package reports

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

// ── helpers ──────────────────────────────────────────────────────────────────

// parseTimeRange parses optional "from" and "to" RFC3339 query params.
// Defaults: from = start of current month, to = now.
func parseTimeRange(c *gin.Context) (from, to time.Time, err error) {
	now := time.Now()
	from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	to = now

	if s := c.Query("from"); s != "" {
		from, err = time.Parse(time.RFC3339, s)
		if err != nil {
			err = apperr.Validation("from must be RFC3339 datetime")
			return
		}
	}
	if s := c.Query("to"); s != "" {
		to, err = time.Parse(time.RFC3339, s)
		if err != nil {
			err = apperr.Validation("to must be RFC3339 datetime")
			return
		}
	}
	return
}

func parseWarehouseID(c *gin.Context) (*int64, error) {
	if s := c.Query("warehouse_id"); s != "" {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, apperr.Validation("warehouse_id must be an integer")
		}
		return &id, nil
	}
	return nil, nil
}

func mustCell(col, row int) string {
	cell, _ := excelize.CoordinatesToCellName(col, row)
	return cell
}

func centsToFloat(cents int64) float64 { return float64(cents) / 100.0 }

func milliToQty(milli int64) float64 { return float64(milli) / 1000.0 }

// excelHeaders builds a consistent set of styles and returns a write-to-buffer function.
func buildExcelStyles(f *excelize.File) (headerStyle, textStyle, textZebra, moneyStyle, moneyZebra, redStyle int) {
	thinBorder := []excelize.Border{
		{Type: "left", Color: "D9D9D9", Style: 1},
		{Type: "right", Color: "D9D9D9", Style: 1},
		{Type: "top", Color: "D9D9D9", Style: 1},
		{Type: "bottom", Color: "D9D9D9", Style: 1},
	}
	tmtFmt := `#,##0.00" TMT"`

	headerStyle, _ = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Family: "Arial", Size: 12, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1F2937"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder,
	})
	textStyle, _ = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Size: 11},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    thinBorder,
	})
	textZebra, _ = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"F2F2F2"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    thinBorder,
	})
	moneyStyle, _ = f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Family: "Calibri", Size: 11},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:       thinBorder,
		CustomNumFmt: &tmtFmt,
	})
	moneyZebra, _ = f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Family: "Calibri", Size: 11},
		Fill:         excelize.Fill{Type: "pattern", Color: []string{"F2F2F2"}, Pattern: 1},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:       thinBorder,
		CustomNumFmt: &tmtFmt,
	})
	// Red tint for negative stock deltas
	redStyle, _ = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Size: 11, Color: "9C0006"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"FFC7CE"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:    thinBorder,
	})
	return
}

func writeExcelHeaders(f *excelize.File, sheet string, headers []string, headerStyle int, lastCol string) {
	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheet, 1, 22)
	f.SetPanes(sheet, &excelize.Panes{
		Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft",
	})
	f.AutoFilter(sheet, fmt.Sprintf("A1:%s1", lastCol), nil)
}

// ── handlers ─────────────────────────────────────────────────────────────────

// Dashboard godoc
// @Summary      Dashboard statistics
// @Description  Returns today's sales, low-stock count, pending payroll, and top products this week.
// @Tags         Reports
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.APIResponse{data=DashboardStats}
// @Failure      500  {object}  response.APIResponse
// @Router       /api/reports/dashboard [get]
func (h *Handler) Dashboard(c *gin.Context) {
	stats, err := h.repo.Dashboard(c.Request.Context())
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, stats)
}

// SalesByPeriod godoc
// @Summary      Sales report by period
// @Description  Aggregated sales grouped by day, week, or month.
// @Tags         Reports
// @Produce      json
// @Security     BearerAuth
// @Param        group_by      query  string  false  "Grouping unit: day | week | month (default: day)"
// @Param        from          query  string  false  "Start datetime RFC3339 (default: start of current month)"
// @Param        to            query  string  false  "End datetime RFC3339 (default: now)"
// @Param        warehouse_id  query  int     false  "Filter by warehouse"
// @Param        customer_type query  string  false  "Filter by customer type: regular | wholesale"
// @Success      200  {object}  response.APIResponse{data=[]SalesPeriodRow}
// @Failure      400  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Router       /api/reports/sales [get]
func (h *Handler) SalesByPeriod(c *gin.Context) {
	from, to, err := parseTimeRange(c)
	if err != nil {
		c.Error(err)
		return
	}
	warehouseID, err := parseWarehouseID(c)
	if err != nil {
		c.Error(err)
		return
	}

	groupBy := c.DefaultQuery("group_by", "day")
	if !validGroupBy[groupBy] {
		c.Error(apperr.Validation("group_by must be day, week, or month"))
		return
	}

	rows, err := h.repo.SalesByPeriod(c.Request.Context(), groupBy, from, to, warehouseID, c.Query("customer_type"))
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, rows)
}

// SalesByProduct godoc
// @Summary      Sales report by product
// @Description  Aggregated sales per product with revenue, cost, profit, and margin.
// @Tags         Reports
// @Produce      json
// @Security     BearerAuth
// @Param        from          query  string  false  "Start datetime RFC3339"
// @Param        to            query  string  false  "End datetime RFC3339"
// @Param        warehouse_id  query  int     false  "Filter by warehouse"
// @Success      200  {object}  response.APIResponse{data=[]SalesProductRow}
// @Failure      400  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Router       /api/reports/sales/products [get]
func (h *Handler) SalesByProduct(c *gin.Context) {
	from, to, err := parseTimeRange(c)
	if err != nil {
		c.Error(err)
		return
	}
	warehouseID, err := parseWarehouseID(c)
	if err != nil {
		c.Error(err)
		return
	}

	rows, err := h.repo.SalesByProduct(c.Request.Context(), from, to, warehouseID)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, rows)
}

// ReorderSuggestions godoc
// @Summary      Reorder suggestions (dynamic low-stock)
// @Description  Lists products that should be reordered, sized from sales history.
// @Description  Uses per-product lead_time_days and safety_stock_milli. Products
// @Description  without sales history fall back to the static LOW_STOCK_DEFAULT threshold.
// @Tags         Reports
// @Produce      json
// @Security     BearerAuth
// @Param        warehouse_id  query  int  false  "Filter by warehouse"
// @Success      200  {object}  response.APIResponse{data=[]ReorderSuggestion}
// @Failure      400  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Router       /api/reports/reorder-suggestions [get]
func (h *Handler) ReorderSuggestions(c *gin.Context) {
	warehouseID, err := parseWarehouseID(c)
	if err != nil {
		c.Error(err)
		return
	}

	rows, err := h.repo.ReorderSuggestions(c.Request.Context(), warehouseID)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, rows)
}

// ExportSales godoc
// @Summary      Export sales report as Excel
// @Description  Downloads an .xlsx file with two sheets: sales by period and by product.
// @Tags         Reports
// @Produce      application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security     BearerAuth
// @Param        group_by      query  string  false  "day | week | month"
// @Param        from          query  string  false  "RFC3339"
// @Param        to            query  string  false  "RFC3339"
// @Param        warehouse_id  query  int     false  "Filter by warehouse"
// @Param        customer_type query  string  false  "regular | wholesale"
// @Success      200
// @Router       /api/reports/sales/export [get]
func (h *Handler) ExportSales(c *gin.Context) {
	from, to, err := parseTimeRange(c)
	if err != nil {
		c.Error(err)
		return
	}
	warehouseID, err := parseWarehouseID(c)
	if err != nil {
		c.Error(err)
		return
	}
	groupBy := c.DefaultQuery("group_by", "day")
	if !validGroupBy[groupBy] {
		c.Error(apperr.Validation("group_by must be day, week, or month"))
		return
	}

	periodRows, err := h.repo.SalesByPeriod(c.Request.Context(), groupBy, from, to, warehouseID, c.Query("customer_type"))
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	productRows, err := h.repo.SalesByProduct(c.Request.Context(), from, to, warehouseID)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}

	data, err := buildSalesExcel(groupBy, periodRows, productRows)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}

	filename := fmt.Sprintf("sales_%s_%s.xlsx",
		from.Format("2006-01-02"), to.Format("2006-01-02"))
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// ExportStock godoc
// @Summary      Export stock movements as Excel
// @Description  Downloads an .xlsx file with stock ledger movements.
// @Tags         Reports
// @Produce      application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security     BearerAuth
// @Param        from          query  string  false  "RFC3339"
// @Param        to            query  string  false  "RFC3339"
// @Param        warehouse_id  query  int     false  "Filter by warehouse"
// @Success      200
// @Router       /api/reports/stock/export [get]
func (h *Handler) ExportStock(c *gin.Context) {
	from, to, err := parseTimeRange(c)
	if err != nil {
		c.Error(err)
		return
	}
	warehouseID, err := parseWarehouseID(c)
	if err != nil {
		c.Error(err)
		return
	}

	rows, err := h.repo.StockMovementsForExport(c.Request.Context(), from, to, warehouseID)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}

	data, err := buildStockExcel(rows)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}

	filename := fmt.Sprintf("stock_%s_%s.xlsx",
		from.Format("2006-01-02"), to.Format("2006-01-02"))
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// ── Excel builders ───────────────────────────────────────────────────────────

func buildSalesExcel(groupBy string, periodRows []SalesPeriodRow, productRows []SalesProductRow) ([]byte, error) {
	f := excelize.NewFile()
	hs, ts, tz, ms, mz, _ := buildExcelStyles(f)

	// ── Sheet 1: By Period ────────────────────────────────────────────────
	sheet1 := "By Period"
	f.SetSheetName("Sheet1", sheet1)

	writeExcelHeaders(f, sheet1,
		[]string{"Period", "Orders", "Revenue (TMT)", "Cost (TMT)", "Profit (TMT)", "Margin %"},
		hs, "F")

	f.SetColWidth(sheet1, "A", "A", 20)
	f.SetColWidth(sheet1, "B", "B", 10)
	f.SetColWidth(sheet1, "C", "F", 16)

	periodFmt := periodDateFormat(groupBy)
	for i, row := range periodRows {
		r := i + 2
		zebra := i%2 == 1
		cur_ts, cur_ms := ts, ms
		if zebra {
			cur_ts, cur_ms = tz, mz
		}
		marginPct := 0.0
		if row.RevenueCents > 0 {
			marginPct = float64(row.ProfitCents) * 100.0 / float64(row.RevenueCents)
		}
		f.SetCellValue(sheet1, mustCell(1, r), row.Period.Format(periodFmt))
		f.SetCellValue(sheet1, mustCell(2, r), row.Orders)
		f.SetCellValue(sheet1, mustCell(3, r), centsToFloat(row.RevenueCents))
		f.SetCellValue(sheet1, mustCell(4, r), centsToFloat(row.CostCents))
		f.SetCellValue(sheet1, mustCell(5, r), centsToFloat(row.ProfitCents))
		f.SetCellValue(sheet1, mustCell(6, r), fmt.Sprintf("%.2f%%", marginPct))
		f.SetCellStyle(sheet1, mustCell(1, r), mustCell(2, r), cur_ts)
		f.SetCellStyle(sheet1, mustCell(3, r), mustCell(5, r), cur_ms)
		f.SetCellStyle(sheet1, mustCell(6, r), mustCell(6, r), cur_ts)
		f.SetRowHeight(sheet1, r, 18)
	}

	// ── Sheet 2: By Product ───────────────────────────────────────────────
	sheet2 := "By Product"
	_, _ = f.NewSheet(sheet2)

	writeExcelHeaders(f, sheet2,
		[]string{"Product", "Qty", "Revenue (TMT)", "Cost (TMT)", "Profit (TMT)", "Margin %"},
		hs, "F")

	f.SetColWidth(sheet2, "A", "A", 28)
	f.SetColWidth(sheet2, "B", "B", 12)
	f.SetColWidth(sheet2, "C", "F", 16)

	for i, row := range productRows {
		r := i + 2
		zebra := i%2 == 1
		cur_ts, cur_ms := ts, ms
		if zebra {
			cur_ts, cur_ms = tz, mz
		}
		f.SetCellValue(sheet2, mustCell(1, r), row.Name)
		f.SetCellValue(sheet2, mustCell(2, r), milliToQty(row.QtyMilli))
		f.SetCellValue(sheet2, mustCell(3, r), centsToFloat(row.RevenueCents))
		f.SetCellValue(sheet2, mustCell(4, r), centsToFloat(row.CostCents))
		f.SetCellValue(sheet2, mustCell(5, r), centsToFloat(row.ProfitCents))
		f.SetCellValue(sheet2, mustCell(6, r), fmt.Sprintf("%.2f%%", row.MarginPct))
		f.SetCellStyle(sheet2, mustCell(1, r), mustCell(2, r), cur_ts)
		f.SetCellStyle(sheet2, mustCell(3, r), mustCell(5, r), cur_ms)
		f.SetCellStyle(sheet2, mustCell(6, r), mustCell(6, r), cur_ts)
		f.SetRowHeight(sheet2, r, 18)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write xlsx: %w", err)
	}
	return buf.Bytes(), nil
}

func buildStockExcel(rows []StockMovementExportRow) ([]byte, error) {
	f := excelize.NewFile()
	hs, ts, tz, ms, mz, redStyle := buildExcelStyles(f)

	sheet := "Stock Movements"
	f.SetSheetName("Sheet1", sheet)

	writeExcelHeaders(f, sheet,
		[]string{"Date", "Warehouse", "Product", "Delta (qty)", "Type", "Unit Price (TMT)", "Created By"},
		hs, "G")

	f.SetColWidth(sheet, "A", "A", 20)
	f.SetColWidth(sheet, "B", "B", 20)
	f.SetColWidth(sheet, "C", "C", 28)
	f.SetColWidth(sheet, "D", "D", 12)
	f.SetColWidth(sheet, "E", "E", 16)
	f.SetColWidth(sheet, "F", "F", 16)
	f.SetColWidth(sheet, "G", "G", 18)

	for i, row := range rows {
		r := i + 2
		zebra := i%2 == 1
		cur_ts, cur_ms := ts, ms
		if zebra {
			cur_ts, cur_ms = tz, mz
		}

		qty := milliToQty(row.DeltaMilli)
		price := ""
		if row.PriceCents != nil {
			price = fmt.Sprintf("%.2f", centsToFloat(*row.PriceCents))
		}
		createdBy := ""
		if row.CreatedBy != nil {
			createdBy = *row.CreatedBy
		}

		f.SetCellValue(sheet, mustCell(1, r), row.CreatedAt.Format("2006-01-02 15:04"))
		f.SetCellValue(sheet, mustCell(2, r), row.Warehouse)
		f.SetCellValue(sheet, mustCell(3, r), row.Product)
		f.SetCellValue(sheet, mustCell(4, r), qty)
		f.SetCellValue(sheet, mustCell(5, r), row.MovementType)
		f.SetCellValue(sheet, mustCell(6, r), price)
		f.SetCellValue(sheet, mustCell(7, r), createdBy)

		f.SetCellStyle(sheet, mustCell(1, r), mustCell(3, r), cur_ts)
		// Delta qty: red if negative, normal otherwise
		if row.DeltaMilli < 0 {
			f.SetCellStyle(sheet, mustCell(4, r), mustCell(4, r), redStyle)
		} else {
			f.SetCellStyle(sheet, mustCell(4, r), mustCell(4, r), cur_ms)
		}
		f.SetCellStyle(sheet, mustCell(5, r), mustCell(7, r), cur_ts)
		f.SetRowHeight(sheet, r, 18)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write xlsx: %w", err)
	}
	return buf.Bytes(), nil
}

func periodDateFormat(groupBy string) string {
	switch groupBy {
	case "month":
		return "2006-01"
	case "week":
		return "2006-01-02"
	default:
		return "2006-01-02"
	}
}
