// middleware/rbac_middleware.go
package middleware

import (
	"context"

	"github.com/gin-gonic/gin"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

// ModuleChecker is implemented by permissions.Repository.
type ModuleChecker interface {
	IsEnabled(ctx context.Context, role, module string) (bool, error)
}

// RequireModule checks if the user's role has access to the given module.
// Admin always bypasses. If Redis/DB is unavailable, defaults to allow.
func RequireModule(repo ModuleChecker, module string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesVal, _ := c.Get("roles")
		userRoles, _ := rolesVal.([]string)

		for _, r := range userRoles {
			if r == "admin" {
				c.Next()
				return
			}
		}

		for _, role := range userRoles {
			enabled, err := repo.IsEnabled(c.Request.Context(), role, module)
			if err != nil || enabled {
				// err → fail-open (allow if DB/Redis unavailable)
				c.Next()
				return
			}
		}

		c.Error(apperr.Forbidden("module access denied"))
		c.Abort()
	}
}

// HasAnyRole returns true if the request context contains at least one of the given roles (admin always passes).
func HasAnyRole(c *gin.Context, roles ...string) bool {
	rolesVal, exists := c.Get("roles")
	if !exists {
		return false
	}
	userRoles, ok := rolesVal.([]string)
	if !ok {
		return false
	}
	for _, r := range userRoles {
		if r == "admin" {
			return true
		}
		for _, required := range roles {
			if r == required {
				return true
			}
		}
	}
	return false
}

// RequireRoles: admin bypass always.
// If roles is empty => admin only.
func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesVal, exists := c.Get("roles")
		if !exists {
			c.Error(apperr.Forbidden("forbidden"))
			c.Abort()
			return
		}

		userRoles, ok := rolesVal.([]string)
		if !ok {
			c.Error(apperr.Forbidden("forbidden"))
			c.Abort()
			return
		}

		for _, r := range userRoles {
			if r == "admin" {
				c.Next()
				return
			}
		}

		if len(roles) == 0 {
			c.Error(apperr.Forbidden("forbidden"))
			c.Abort()
			return
		}

		for _, required := range roles {
			for _, userRole := range userRoles {
				if required == userRole {
					c.Next()
					return
				}
			}
		}

		c.Error(apperr.Forbidden("forbidden"))
		c.Abort()
	}
}
