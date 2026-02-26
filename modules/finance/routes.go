package finance

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	// Payment types — read for cashier and above
	rg.GET("/payment-types",
		middleware.RequireRoles("cashier", "operator", "manager"),
		h.ListPaymentTypes,
	)

	// Transactions
	txns := rg.Group("/transactions")

	// List — read for cashier and above
	txns.GET("",
		middleware.RequireRoles("cashier", "operator", "manager"),
		h.ListTransactions,
	)

	// Get detail — read for cashier and above
	txns.GET("/:id",
		middleware.RequireRoles("cashier", "operator", "manager"),
		h.GetTransaction,
	)

	// Create manual transaction — operator/manager
	txns.POST("/manual",
		middleware.RequireRoles("operator", "manager"),
		h.CreateManual,
	)

	// Add payment — operator/manager
	txns.POST("/:id/payments",
		middleware.RequireRoles("operator", "manager"),
		h.AddPayment,
	)

	// Cancel — manager only
	txns.POST("/:id/cancel",
		middleware.RequireRoles("manager"),
		h.CancelTransaction,
	)
}
