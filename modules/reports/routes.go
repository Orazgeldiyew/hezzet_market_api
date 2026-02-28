package reports

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, lowStockDefault int64) {
	repo := NewRepository(db, lowStockDefault)
	h := NewHandler(repo)

	g := rg.Group("/reports")
	g.Use(middleware.RequireRoles("manager", "admin"))

	g.GET("/dashboard",      h.Dashboard)
	g.GET("/sales",          h.SalesByPeriod)
	g.GET("/sales/products", h.SalesByProduct)
	g.GET("/sales/export",   h.ExportSales)
	g.GET("/stock/export",   h.ExportStock)
}
