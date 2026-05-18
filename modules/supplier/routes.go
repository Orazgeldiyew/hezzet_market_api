package supplier

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	// Per-action permissions. Previously the whole module was operator-only,
	// which silently blocked manager even from reading the supplier list.
	g := rg.Group("/suppliers")
	{
		g.GET("", middleware.RequirePermission(permChecker, "suppliers", "view"), middleware.PaginationMiddleware(), h.List)
		g.GET("/:id", middleware.RequirePermission(permChecker, "suppliers", "view"), h.Get)
		g.POST("", middleware.RequirePermission(permChecker, "suppliers", "create"), h.Create)
		g.PATCH("/:id", middleware.RequirePermission(permChecker, "suppliers", "update"), h.Update)
		g.DELETE("/:id", middleware.RequirePermission(permChecker, "suppliers", "delete"), h.Delete)
	}
}
