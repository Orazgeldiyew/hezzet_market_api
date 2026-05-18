package warehouse

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	// Per-action permissions. Manager-only create previously blocked operators
	// who legitimately need to register a new register location.
	warehouses := rg.Group("/warehouses")
	{
		warehouses.GET("",
			middleware.RequirePermission(permChecker, "warehouses", "view"),
			middleware.PaginationMiddleware(),
			h.List,
		)
		warehouses.GET("/:id",
			middleware.RequirePermission(permChecker, "warehouses", "view"),
			h.Get,
		)
		warehouses.POST("",
			middleware.RequirePermission(permChecker, "warehouses", "create"),
			h.Create,
		)
	}
}
