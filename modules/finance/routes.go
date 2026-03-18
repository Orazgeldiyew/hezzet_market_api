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
		middleware.RequireRoles("cashier", "operator", "manager", "admin"),
		h.ListPaymentTypes,
	)

	// Transactions
	txns := rg.Group("/transactions")

	// List — manager and admin
	txns.GET("",
		middleware.RequireRoles("manager", "admin"),
		h.ListTransactions,
	)

	// Get detail — manager and admin
	txns.GET("/:id",
		middleware.RequireRoles("manager", "admin"),
		h.GetTransaction,
	)

	// Create manual transaction — operator/manager/admin
	txns.POST("/manual",
		middleware.RequireRoles("operator", "manager", "admin"),
		h.CreateManual,
	)

	// Add payment — operator/manager/admin
	txns.POST("/:id/payments",
		middleware.RequireRoles("operator", "manager", "admin"),
		h.AddPayment,
	)

	// Cancel — manager and admin
	txns.POST("/:id/cancel",
		middleware.RequireRoles("manager", "admin"),
		h.CancelTransaction,
	)
}
