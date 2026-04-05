package inventory

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool) {
	repo := NewRepository(db)
	h := NewHandler(repo)

	inv := rg.Group("/inventory")
	inv.Use(middleware.RequireRoles("manager"))
	{
		inv.POST("", h.Create)
		inv.GET("", h.List)
		inv.GET("/:id", h.Get)
		inv.PUT("/:id/items", h.UpdateItems)
		inv.POST("/:id/confirm", h.Confirm)
		inv.POST("/:id/cancel", h.Cancel)
	}
}
