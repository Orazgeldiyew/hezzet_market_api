package workercard

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool) {
	repo := NewRepository(db)
	h := NewHandler(repo)

	g := rg.Group("/worker-cards")

	// Lookup by card code — cashier/operator/manager
	g.GET("/by-card/:code", middleware.RequireRoles("cashier", "operator", "manager"), h.GetByCard)

	// List all cards (optionally filtered by worker_id) — manager
	g.GET("", middleware.RequireRoles("manager"), h.List)

	// Card management — manager only
	g.POST("", middleware.RequireRoles("manager"), h.Add)
	g.PUT("/:id", middleware.RequireRoles("manager"), h.Update)
	g.DELETE("/:id", middleware.RequireRoles("manager"), h.Delete)
}
