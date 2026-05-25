package auditlog

import (
	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes wires audit-log read endpoints under reports:view (same
// audience as the rest of the reports module — manager/admin).
func RegisterRoutes(rg *gin.RouterGroup, repo *Repository, permChecker middleware.PermissionChecker) {
	h := NewHandler(repo)

	rg.GET("/audit-logs",
		middleware.RequirePermission(permChecker, "reports", "view"),
		h.List,
	)
	rg.GET("/audit-logs/stats",
		middleware.RequirePermission(permChecker, "reports", "view"),
		h.Stats,
	)
}
