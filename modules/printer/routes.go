package printer

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, uploadsDir string) *Service {
	repo := NewRepository(db)
	svc := NewService(repo, db, uploadsDir)
	h := NewHandler(svc)

	g := rg.Group("/printers")
	g.Use(middleware.RequireRoles("admin"))

	g.POST("", h.Create)
	g.GET("", h.List)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
	g.POST("/:id/test", h.TestPrint)

	return svc
}
