package workerfinance

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
)

// RegisterRoutes wires worker-finance endpoints under the `workers` permission
// module. Money-related actions (fines, debts, compensation) reuse the same
// granularity: workers:view = read, workers:update = create/edit/pay.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, financeRepo *finance.Repository, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db)
	svc := NewService(repo, financeRepo)
	h := NewHandler(svc)

	workers := rg.Group("/workers")

	// Compensation
	workers.POST("/compensation",
		middleware.RequirePermission(permChecker, "workers", "update"),
		h.SetCompensation,
	)
	workers.GET("/:id/compensation",
		middleware.RequirePermission(permChecker, "workers", "view"),
		h.GetCompensation,
	)

	// Fines
	workers.POST("/:id/fines",
		middleware.RequirePermission(permChecker, "workers", "update"),
		h.CreateFine,
	)
	workers.GET("/:id/fines",
		middleware.RequirePermission(permChecker, "workers", "view"),
		middleware.PaginationMiddleware(),
		h.ListFines,
	)

	// Debts
	workers.GET("/debts/debtors",
		middleware.RequirePermission(permChecker, "workers", "view"),
		h.AllDebtors,
	)
	workers.POST("/:id/debts",
		middleware.RequirePermission(permChecker, "workers", "update"),
		h.CreateDebt,
	)
	workers.GET("/:id/debts",
		middleware.RequirePermission(permChecker, "workers", "view"),
		middleware.PaginationMiddleware(),
		h.ListDebts,
	)
	workers.GET("/debts/:debt_id",
		middleware.RequirePermission(permChecker, "workers", "view"),
		h.GetDebt,
	)
	workers.POST("/debts/:debt_id/pay",
		middleware.RequirePermission(permChecker, "workers", "update"),
		h.PayDebt,
	)
}
