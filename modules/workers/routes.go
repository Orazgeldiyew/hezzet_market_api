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

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	workers := rg.Group("/workers")

	// Read: manager/operator (admin bypass)
	workers.GET("",
		middleware.RequireRoles("manager", "operator"),
		middleware.PaginationMiddleware(),
		h.List,
	)
	workers.GET("/:id",
		middleware.RequireRoles("manager", "operator"),
		h.Get,
	)

	// Write: manager (admin bypass)
	workers.POST("",
		middleware.RequireRoles("manager"),
		h.Create,
	)
	workers.PATCH("/:id",
		middleware.RequireRoles("manager"),
		h.Update,
	)

	// Delete: admin only
	workers.DELETE("/:id",
		middleware.RequireRoles(), // empty => admin only (admin bypass still works)
		h.Delete,
	)
}
