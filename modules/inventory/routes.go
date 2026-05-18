package inventory

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, finRepo *finance.Repository, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db, finRepo)
	h := NewHandler(repo)

	// Inventory counts adjust stock levels, so they fall under the "stock"
	// permission group. Previously gated to manager only — operators were
	// blocked even from listing past counts.
	inv := rg.Group("/inventory")
	{
		inv.GET("", middleware.RequirePermission(permChecker, "stock", "view"), h.List)
		inv.GET("/:id", middleware.RequirePermission(permChecker, "stock", "view"), h.Get)
		inv.POST("", middleware.RequirePermission(permChecker, "stock", "create"), h.Create)
		inv.PUT("/:id/items", middleware.RequirePermission(permChecker, "stock", "update"), h.UpdateItems)
		inv.POST("/:id/confirm", middleware.RequirePermission(permChecker, "stock", "update"), h.Confirm)
		inv.POST("/:id/cancel", middleware.RequirePermission(permChecker, "stock", "delete"), h.Cancel)
	}
}
