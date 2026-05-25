package reports

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes wires read-only report endpoints under the `reports`
// permission module. Reorder-suggestions overlaps with stock domain, but
// it's a managerial report so we keep it under reports:view too.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, lowStockDefault int64, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db, lowStockDefault)
	h := NewHandler(repo)

	g := rg.Group("/reports")
	g.Use(middleware.RequirePermission(permChecker, "reports", "view"))

	g.GET("/dashboard", h.Dashboard)
	g.GET("/sales", h.SalesByPeriod)
	g.GET("/sales/products", h.SalesByProduct)
	g.GET("/sales/export", h.ExportSales)
	g.GET("/stock/export", h.ExportStock)
	g.GET("/reorder-suggestions", h.ReorderSuggestions)
}
