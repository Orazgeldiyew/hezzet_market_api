package printer

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes wires printer-management endpoints under the `reports`
// permission module (printers are a manager-level configuration item, same
// audience as receipt settings). Admin bypass still works for true admins.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, uploadsDir string, permChecker middleware.PermissionChecker) *Service {
	repo := NewRepository(db)
	svc := NewService(repo, db, uploadsDir)
	h := NewHandler(svc)

	g := rg.Group("/printers")

	// Per-action gating so a user with only reports:view can't create or
	// delete printers. Admins still bypass through the privileged-role check.
	g.GET("", middleware.RequirePermission(permChecker, "reports", "view"), h.List)
	g.POST("", middleware.RequirePermission(permChecker, "reports", "create"), h.Create)
	g.PATCH("/:id", middleware.RequirePermission(permChecker, "reports", "update"), h.Update)
	g.DELETE("/:id", middleware.RequirePermission(permChecker, "reports", "delete"), h.Delete)
	// TestPrint actually performs an action (sends bytes to a printer), so it
	// requires update rather than view.
	g.POST("/:id/test", middleware.RequirePermission(permChecker, "reports", "update"), h.TestPrint)

	return svc
}
