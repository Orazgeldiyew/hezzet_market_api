package warehouse

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	warehouses := rg.Group("/warehouses")

	warehouses.POST("",
		middleware.RequireRoles("manager"),
		h.Create,
	)
	warehouses.GET("",
		middleware.RequireRoles("operator", "cashier", "manager"),
		middleware.PaginationMiddleware(),
		h.List,
	)
	warehouses.GET("/:id",
		middleware.RequireRoles("operator", "cashier", "manager"),
		h.Get,
	)
}
