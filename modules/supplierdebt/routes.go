package supplierdebt

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
)

// RegisterRoutes wires supplier-debt routes. Treats debts as an extension of
// the supplier record: read/pay use suppliers:view / suppliers:update. Create
// is gated on suppliers:create (a manual debt entry is essentially registering
// a new financial obligation to a supplier).
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, finRepo *finance.Repository, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db, finRepo)
	h := NewHandler(repo)

	g := rg.Group("/supplier-debts")
	g.Use(middleware.PaginationMiddleware())
	{
		g.POST("", middleware.RequirePermission(permChecker, "suppliers", "create"), h.Create)
		g.GET("", middleware.RequirePermission(permChecker, "suppliers", "view"), h.ListAll)
		g.GET("/suppliers", middleware.RequirePermission(permChecker, "suppliers", "view"), h.DebtorsSummary)
		g.GET("/supplier/:supplier_id", middleware.RequirePermission(permChecker, "suppliers", "view"), h.ListBySupplier)
		g.GET("/:id", middleware.RequirePermission(permChecker, "suppliers", "view"), h.GetByID)
		g.POST("/:id/pay", middleware.RequirePermission(permChecker, "suppliers", "update"), h.Pay)
	}
}
