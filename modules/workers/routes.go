// package workers

// import (
// 	"github.com/gin-gonic/gin"
// 	"github.com/jackc/pgx/v5/pgxpool"

// 	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
// )

// func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool) {
// 	repo := NewRepository(db)
// 	svc := NewService(repo)
// 	h := NewHandler(svc)

// 	// Read: manager, operator (admin bypass)
// 	read := rg.Group("/workers")
// 	read.Use(middleware.RequireRoles("manager", "operator"))
// 	{
// 		read.GET("", middleware.PaginationMiddleware(), h.List)
// 		read.GET("/:id", h.Get)
// 	}

// 	// Write: manager (admin bypass)
// 	write := rg.Group("/workers")
// 	write.Use(middleware.RequireRoles("manager"))
// 	{
// 		write.POST("", h.Create)
// 		write.PATCH("/:id", h.Update)
// 	}

// 	// Delete: admin only
// 	admin := rg.Group("/workers")
// 	admin.Use(middleware.RequireRoles())
// 	{
// 		admin.DELETE("/:id", h.Delete)
// 	}
// }

package workers

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes wires worker CRUD endpoints under the `workers` permission
// module so admins can grant operator the right to add/edit workers via
// /roles without code changes.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	workers := rg.Group("/workers")

	workers.GET("",
		middleware.RequirePermission(permChecker, "workers", "view"),
		middleware.PaginationMiddleware(),
		h.List,
	)
	workers.GET("/:id",
		middleware.RequirePermission(permChecker, "workers", "view"),
		h.Get,
	)
	workers.POST("",
		middleware.RequirePermission(permChecker, "workers", "create"),
		h.Create,
	)
	workers.PATCH("/:id",
		middleware.RequirePermission(permChecker, "workers", "update"),
		h.Update,
	)
	workers.DELETE("/:id",
		middleware.RequirePermission(permChecker, "workers", "delete"),
		h.Delete,
	)
}
