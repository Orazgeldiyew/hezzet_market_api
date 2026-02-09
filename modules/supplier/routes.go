package supplier

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r *gin.Engine, db *pgxpool.Pool) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	g := r.Group("/suppliers")
	{
		g.POST("", h.Create)
		g.GET("", h.List)
		g.GET("/:id", h.Get)
		g.PATCH("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
	}
}
