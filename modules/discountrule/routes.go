package discountrule

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool) *Repository {
	repo := NewRepository(db)
	h := NewHandler(repo)

	g := rg.Group("/discount-rules")
	g.Use(middleware.RequireRoles("manager"))
	{
		g.GET("", h.List)
		g.POST("", h.Create)
		g.GET("/:id", h.GetByID)
		g.PATCH("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
	}

	return repo
}
