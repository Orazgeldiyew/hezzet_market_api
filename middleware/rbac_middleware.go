// middleware/rbac_middleware.go
package middleware

import (
	"github.com/gin-gonic/gin"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

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
