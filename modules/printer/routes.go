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
	g.Use(middleware.RequirePermission(permChecker, "reports", "view"))

	g.GET("", h.List)
	g.POST("", h.Create)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
	g.POST("/:id/test", h.TestPrint)

	return svc
}
