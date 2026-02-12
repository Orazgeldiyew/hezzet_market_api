package supplier

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	// All supplier endpoints: operator (admin bypass)
	g := rg.Group("/suppliers")
	g.Use(middleware.RequireRoles("operator"))
	{
		g.POST("", h.Create)
		g.GET("", middleware.PaginationMiddleware(), h.List)
		g.GET("/:id", h.Get)
		g.PATCH("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
	}
}
