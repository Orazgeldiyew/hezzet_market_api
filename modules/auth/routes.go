// modules/auth/routes.go
package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"
	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(r *gin.Engine, db *pgxpool.Pool, cfg config.Config) {
	repo := NewRepository(db)
	svc := NewService(repo, cfg)
	h := NewHandler(svc)

	g := r.Group("/auth")
	{
		g.POST("/login", h.Login)

		admin := g.Group("/users")
		admin.Use(middleware.AuthRequired(cfg))
		admin.Use(middleware.RequireRoles()) // empty = admin only
		{
			admin.POST("", h.CreateUser)
			admin.GET("", middleware.PaginationMiddleware(), h.ListUsers)
			admin.GET("/:id", h.GetUser)
			admin.PATCH("/:id", h.UpdateUser)
			admin.DELETE("/:id", h.DeleteUser)
		}

		self := g.Group("/users")
		self.Use(middleware.AuthRequired(cfg))
		{
			self.POST("/:id/password", h.ChangePassword)
		}
	}
}
