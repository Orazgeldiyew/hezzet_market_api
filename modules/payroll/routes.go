package payroll

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/workerfinance"
)

// RegisterRoutes wires payroll endpoints under the `payroll` permission
// module so admins can let, e.g., an HR operator view payroll without giving
// them the cash-out (Pay) capability.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, finRepo *finance.Repository, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db)
	wfRepo := workerfinance.NewRepository(db)
	svc := NewService(repo, wfRepo, finRepo)
	h := NewHandler(svc)

	pr := rg.Group("/payroll")

	pr.GET("",
		middleware.RequirePermission(permChecker, "payroll", "view"),
		middleware.PaginationMiddleware(),
		h.List,
	)
	pr.GET("/export",
		middleware.RequirePermission(permChecker, "payroll", "view"),
		h.ExportExcel,
	)
	// Calculate writes a payroll_runs row — it's a create operation.
	pr.POST("/calculate",
		middleware.RequirePermission(permChecker, "payroll", "create"),
		h.Calculate,
	)
	// Pay actually moves cash out of the till; reserve under update so it can
	// be granted separately from the (less sensitive) calculate action.
	pr.POST("/:id/pay",
		middleware.RequirePermission(permChecker, "payroll", "update"),
		h.Pay,
	)
}
