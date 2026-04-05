package supplierdebt

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool) {
	repo := NewRepository(db)
	h := NewHandler(repo)

	g := rg.Group("/supplier-debts")
	g.Use(middleware.RequireRoles("manager"))
	g.Use(middleware.PaginationMiddleware())
	{
		g.POST("", h.Create)
		g.GET("", h.ListAll)
		g.GET("/suppliers", h.DebtorsSummary)
		g.GET("/supplier/:supplier_id", h.ListBySupplier)
		g.GET("/:id", h.GetByID)
		g.POST("/:id/pay", h.Pay)
	}
}
