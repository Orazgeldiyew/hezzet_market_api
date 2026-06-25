// middleware/rbac_middleware.go
package middleware

import (
	"context"

	"github.com/gin-gonic/gin"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

// PermissionChecker checks action-level permissions.
type PermissionChecker interface {
	IsAllowed(ctx context.Context, roleCodes []string, module, action string) (bool, error)
}

// ── RequirePermission (new, action-level) ──

// RequirePermission checks if the user's roles have granted=true for
// module+action. Admin always bypasses.
//
// On checker failure (Redis down + DB down, etc.) we FAIL CLOSED: return 503
// so the client retries instead of silently being granted access we couldn't
// verify. The Redis fallback path inside IsAllowed already handles partial
// outages by falling back to DB; we only get here if BOTH are dead.
func RequirePermission(checker PermissionChecker, module, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesVal, _ := c.Get("roles")
		userRoles, _ := rolesVal.([]string)

		for _, r := range userRoles {
			if r == "admin" {
				c.Next()
				return
			}
		}

		allowed, err := checker.IsAllowed(c.Request.Context(), userRoles, module, action)
		if err != nil {
			c.Error(apperr.ServiceUnavailable("permission service unavailable", err))
			c.Abort()
			return
		}
		if allowed {
			c.Next()
			return
		}

		c.Error(apperr.Forbidden("permission denied"))
		c.Abort()
	}
}

// ── HasAnyRole ──

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

// ── RequireRoles ──

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
