package favorite

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool) {
	repo := NewRepository(db)
	h := NewHandler(repo)

	g := rg.Group("/favorites")
	{
		g.GET("", h.List)
		g.POST("", h.Add)
		g.PUT("/reorder", h.Reorder)
		g.DELETE("/:product_id", h.Remove)
	}
}
