package customerdebt

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes wires customer-debt routes. Reads use the customers:view
// permission (debts are conceptually part of the customer record), payments
// use customers:update so a role can have view-only access to debts without
// being able to record cash receipts.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, permChecker middleware.PermissionChecker) *Repository {
	repo := NewRepository(db)
	h := NewHandler(repo)

	g := rg.Group("/customer-debts")
	g.Use(middleware.PaginationMiddleware())
	{
		g.GET("", middleware.RequirePermission(permChecker, "customers", "view"), h.ListAll)
		g.GET("/debtors", middleware.RequirePermission(permChecker, "customers", "view"), h.DebtorsSummary)
		g.GET("/customer/:customer_id", middleware.RequirePermission(permChecker, "customers", "view"), h.ListByCustomer)
		g.GET("/customer/:customer_id/total", middleware.RequirePermission(permChecker, "customers", "view"), h.GetCustomerTotal)
		g.GET("/:id", middleware.RequirePermission(permChecker, "customers", "view"), h.GetByID)
		g.POST("/:id/pay", middleware.RequirePermission(permChecker, "customers", "update"), h.Pay)
	}

	return repo
}
