package shift

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes wires shift endpoints under the `sales` permission module
// since shifts are inherently part of a cashier's selling workflow:
//   sales:create → open/close own shift
//   sales:history → list/inspect any shift's Z-report (manager-level review)
//
// Cash registers list (used to pick a register when opening) is read-only;
// gate behind sales:view so cashier can see them.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db)
	h := NewHandler(repo)

	// Cash registers — read-only.
	rg.GET("/registers",
		middleware.RequirePermission(permChecker, "sales", "view"),
		h.ListRegisters,
	)

	// Shifts — own-shift operations: open/close/current.
	shifts := rg.Group("/shifts")
	shifts.POST("/open",
		middleware.RequirePermission(permChecker, "sales", "create"),
		h.OpenShift,
	)
	shifts.GET("/current",
		middleware.RequirePermission(permChecker, "sales", "view"),
		h.GetCurrent,
	)
	// CloseShift checks ownership internally (cashier can close own, manager
	// can force-close any) — so we only need sales:create as the floor.
	shifts.POST("/:id/close",
		middleware.RequirePermission(permChecker, "sales", "create"),
		h.CloseShift,
	)

	// Cross-cashier review — list/detail of all shifts including discrepancies.
	shifts.GET("",
		middleware.RequirePermission(permChecker, "sales", "history"),
		h.ListShifts,
	)
	shifts.GET("/:id",
		middleware.RequirePermission(permChecker, "sales", "history"),
		h.GetShift,
	)
}
