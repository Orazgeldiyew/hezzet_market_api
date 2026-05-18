package category

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	// Per-action permissions so admins can grant categories editing to
	// whatever role they like via the /roles matrix. Manager was silently
	// blocked under the previous "operator only" rule.
	read := rg.Group("/categories")
	read.Use(middleware.RequirePermission(permChecker, "categories", "view"))
	{
		read.GET("", middleware.PaginationMiddleware(), h.List)
		read.GET("/tree", h.Tree)
		read.GET("/:id", h.Get)
	}

	write := rg.Group("/categories")
	{
		write.POST("", middleware.RequirePermission(permChecker, "categories", "create"), h.Create)
		write.PATCH("/:id", middleware.RequirePermission(permChecker, "categories", "update"), h.Update)
		write.DELETE("/:id", middleware.RequirePermission(permChecker, "categories", "delete"), h.Delete)
	}
}
