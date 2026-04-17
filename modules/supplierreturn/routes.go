package supplierreturn

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	g := rg.Group("/supplier-returns")
	// Read (list/detail) — anyone with purchases:view
	g.GET("", middleware.RequirePermission(permChecker, "purchases", "view"), h.List)
	g.GET("/:id", middleware.RequirePermission(permChecker, "purchases", "view"), h.GetByID)

	// Write (create/confirm/cancel) — requires purchases:return permission (manager/admin)
	g.POST("", middleware.RequirePermission(permChecker, "purchases", "return"), h.Create)
	g.POST("/:id/confirm", middleware.RequirePermission(permChecker, "purchases", "return"), h.Confirm)
	g.POST("/:id/cancel", middleware.RequirePermission(permChecker, "purchases", "return"), h.Cancel)
}
