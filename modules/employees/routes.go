package employees

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes wires the unified employees CRUD under /api/employees.
// Permissions are scoped to the "workers" module so existing role grants
// continue to work — admins/managers with workers:create can create any
// employee (worker, user, or both).
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	g := rg.Group("/employees")
	g.GET("",
		middleware.RequirePermission(permChecker, "workers", "view"),
		middleware.PaginationMiddleware(),
		h.List,
	)
	g.GET("/:id",
		middleware.RequirePermission(permChecker, "workers", "view"),
		h.Get,
	)
	g.POST("",
		middleware.RequirePermission(permChecker, "workers", "create"),
		h.Create,
	)
	g.PATCH("/:id",
		middleware.RequirePermission(permChecker, "workers", "update"),
		h.Update,
	)
	g.DELETE("/:id",
		middleware.RequirePermission(permChecker, "workers", "delete"),
		h.Delete,
	)
}
