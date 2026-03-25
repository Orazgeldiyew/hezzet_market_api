package permissions

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes registers role & permission management endpoints and returns the Repository
// so it can be used by RequirePermission middleware in router.go.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, rdb *redis.Client) *Repository {
	repo := NewRepository(db, rdb)
	svc := NewService(repo)
	h := NewHandler(svc)

	// ── Roles CRUD ──
	roles := rg.Group("/roles")
	roles.GET("", middleware.RequireRoles("manager"), h.ListRoles)
	roles.GET("/:id", middleware.RequireRoles("manager"), h.GetRole)
	roles.POST("", middleware.RequireRoles(), h.CreateRole)   // admin only
	roles.PATCH("/:id", middleware.RequireRoles(), h.UpdateRole) // admin only
	roles.DELETE("/:id", middleware.RequireRoles(), h.DeleteRole) // admin only

	// ── Permissions ──
	perms := rg.Group("/permissions")
	perms.GET("", middleware.RequireRoles("manager"), h.ListPermissions)
	perms.GET("/matrix", middleware.RequireRoles("manager"), h.Matrix)
	perms.PUT("/:role_id/:module/:action", middleware.RequireRoles("manager"), h.UpdatePermission)
	perms.PUT("/:role_id/bulk", middleware.RequireRoles("manager"), h.BulkUpdate)

	return repo
}
