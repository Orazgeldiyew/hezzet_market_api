package customerdebt

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool) *Repository {
	repo := NewRepository(db)
	h := NewHandler(repo)

	g := rg.Group("/customer-debts")
	g.Use(middleware.RequireRoles("cashier", "operator", "manager"))
	g.Use(middleware.PaginationMiddleware())
	{
		g.GET("", h.ListAll)                             // all open debts
		g.GET("/debtors", h.DebtorsSummary)              // summary by customer
		g.GET("/customer/:customer_id", h.ListByCustomer) // debts of one customer
		g.GET("/customer/:customer_id/total", h.GetCustomerTotal)
		g.GET("/:id", h.GetByID)                         // single debt + payments
		g.POST("/:id/pay", h.Pay)                        // make payment
	}

	return repo
}
