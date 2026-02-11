package product

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(r *gin.Engine, db *pgxpool.Pool) {
	repo := NewRepository(db)
	stock := NewStockRepository(db)
	svc := NewService(repo, stock)
	h := NewHandler(svc)

	g := r.Group("/products")
	{
		g.POST("", h.Create)
		g.GET("", middleware.PaginationMiddleware(), h.List)
		g.GET("/:id", h.Get)
		g.PATCH("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
		g.GET("/:id/card", h.GetCard)
		g.GET("/:id/categories", h.GetCategories)
		g.PUT("/:id/categories", h.SetCategories)
		g.DELETE("/:id/categories/:categoryId", h.RemoveCategory)
	}
}
