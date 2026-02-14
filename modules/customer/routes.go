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

	customers := rg.Group("/customers")

	// Read: cashier/operator/manager (admin bypass)
	customers.GET("", middleware.RequireRoles("cashier", "operator", "manager"), middleware.PaginationMiddleware(), h.List)
	customers.GET("/:id", middleware.RequireRoles("cashier", "operator", "manager"), h.Get)

	// Create + spent: cashier/manager (admin bypass)
	customers.POST("", middleware.RequireRoles("cashier", "manager"), h.Create)
	customers.POST("/:id/spent", middleware.RequireRoles("cashier", "manager"), h.AddSpent)

	// Contact update: cashier/operator/manager (admin bypass)
	customers.PATCH("/:id/contact", middleware.RequireRoles("cashier", "operator", "manager"), h.UpdateContact)

	// Business update: manager/admin (admin bypass already works)
	customers.PATCH("/:id/admin", middleware.RequireRoles("manager"), h.UpdateAdmin)

	// Delete: admin only
	customers.DELETE("/:id", middleware.RequireRoles(), h.Delete)
}
