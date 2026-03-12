package permissions

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes registers permission management endpoints and returns the Repository
// so it can be used by RequireModule middleware in router.go.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, rdb *redis.Client) *Repository {
	repo := NewRepository(db, rdb)
	svc := NewService(repo)
	h := NewHandler(svc)

	perms := rg.Group("/permissions")
	perms.GET("", middleware.RequireRoles("manager"), h.List)
	perms.PUT("/:role/:module", middleware.RequireRoles("manager"), h.Update)

	return repo
}
