// modules/auth/routes.go
package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"
	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes sets up all auth routes and returns the token-version
// checker so the caller can reuse it for other protected route groups.
// auditMiddleware is optional — when provided it is applied to every
// authenticated write route so those actions appear in the audit log.
// RegisterResult bundles the return values of RegisterRoutes.
type RegisterResult struct {
	TokenVersionFunc middleware.TokenVersionFunc
	Service          *Service
}

func RegisterRoutes(r *gin.Engine, db *pgxpool.Pool, cfg config.Config, auditMiddleware ...gin.HandlerFunc) RegisterResult {
	repo := NewRepository(db)
	svc := NewService(repo, cfg)
	h := NewHandler(svc)

	g := r.Group("/api/auth")
	{
		// Public
		g.POST("/login", h.Login)
		g.POST("/refresh", h.RefreshToken)
		g.POST("/register", h.Register)

		// Admin-only user management
		admin := g.Group("/users")
		admin.Use(middleware.AuthRequired(cfg, repo.GetTokenVersion))
		admin.Use(middleware.RequireRoles("manager")) // admin + manager
		admin.Use(auditMiddleware...)
		{
			admin.POST("", h.CreateUser)
			admin.GET("", middleware.PaginationMiddleware(), h.ListUsers)
			admin.GET("/:id", h.GetUser)
			admin.PATCH("/:id", h.UpdateUser)
			admin.DELETE("/:id", h.DeleteUser)
			admin.POST("/:id/block", h.BlockUser)
			admin.POST("/:id/unblock", h.UnblockUser)
		}

		// Authenticated (admin or self)
		self := g.Group("/users")
		self.Use(middleware.AuthRequired(cfg, repo.GetTokenVersion))
		self.Use(auditMiddleware...)
		{
			self.POST("/:id/password", h.ChangePassword)
		}
	}

	return RegisterResult{
		TokenVersionFunc: repo.GetTokenVersion,
		Service:          svc,
	}
}
