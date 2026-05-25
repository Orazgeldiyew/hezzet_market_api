package finance

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes wires finance endpoints behind the `finance` permission
// module. Payment types are needed by the cashier UI (to render the payment
// type list at checkout) — they're a lightweight reference, gated on view.
// Transaction list/detail are sensitive financial reports → finance:view
// (default off for cashier). Manual transactions write money in/out, so
// create + update for create/payment, delete for cancel.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	// Payment types — kept on a wider gate (sales:view) since cashier needs
	// them at the till; finance:view would lock them out of confirming sales.
	rg.GET("/payment-types",
		middleware.RequirePermission(permChecker, "sales", "view"),
		h.ListPaymentTypes,
	)

	txns := rg.Group("/transactions")

	txns.GET("",
		middleware.RequirePermission(permChecker, "finance", "view"),
		h.ListTransactions,
	)
	txns.GET("/:id",
		middleware.RequirePermission(permChecker, "finance", "view"),
		h.GetTransaction,
	)
	txns.POST("/manual",
		middleware.RequirePermission(permChecker, "finance", "create"),
		h.CreateManual,
	)
	txns.POST("/:id/payments",
		middleware.RequirePermission(permChecker, "finance", "update"),
		h.AddPayment,
	)
	txns.POST("/:id/cancel",
		middleware.RequirePermission(permChecker, "finance", "delete"),
		h.CancelTransaction,
	)
}
