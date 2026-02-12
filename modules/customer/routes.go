package customer

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	// Read: cashier, operator, manager (admin bypass)
	read := rg.Group("/customers")
	read.Use(middleware.RequireRoles("cashier", "operator", "manager"))
	{
		read.GET("", middleware.PaginationMiddleware(), h.List)
		read.GET("/:id", h.Get)
	}

	// Write: cashier (admin bypass)
	write := rg.Group("/customers")
	write.Use(middleware.RequireRoles("cashier"))
	{
		write.POST("", h.Create)
		write.POST("/:id/spent", h.AddSpent)
	}

	// Update/Delete: admin only
	admin := rg.Group("/customers")
	admin.Use(middleware.RequireRoles())
	{
		admin.PATCH("/:id", h.Update)
		admin.DELETE("/:id", h.Delete)
	}
}
