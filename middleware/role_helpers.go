package middleware

import "github.com/gin-gonic/gin"

// IsPrivileged returns true if any of the caller's roles is admin/manager/operator.
// Used by service-level ownership checks: privileged users see all records;
// non-privileged (cashier) can only see records they created.
func IsPrivileged(roles []string) bool {
	for _, r := range roles {
		if r == "admin" || r == "manager" || r == "operator" {
			return true
		}
	}
	return false
}

// CallerRoles pulls the roles slice the JWT middleware put on the gin context.
// Returns nil if absent — callers should treat that as "unprivileged".
func CallerRoles(c *gin.Context) []string {
	v, _ := c.Get("roles")
	r, _ := v.([]string)
	return r
}

// CallerUserID pulls the user_id the JWT middleware put on the gin context.
// Returns 0 if absent.
func CallerUserID(c *gin.Context) int64 {
	v, _ := c.Get("user_id")
	uid, _ := v.(int64)
	return uid
}
