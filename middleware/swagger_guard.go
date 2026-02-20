package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func SwaggerGuard(allowIPs []string, user, pass string) gin.HandlerFunc {
	allowed := make([]*net.IPNet, 0, len(allowIPs))
	for _, cidr := range allowIPs {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		_, ipnet, err := net.ParseCIDR(cidr)
		if err == nil && ipnet != nil {
			allowed = append(allowed, ipnet)
		}
	}

	return func(c *gin.Context) {
		// IP allowlist
		if len(allowed) > 0 {
			ipStr := clientIP(c)
			ip := net.ParseIP(ipStr)
			if ip == nil {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			ok := false
			for _, n := range allowed {
				if n.Contains(ip) {
					ok = true
					break
				}
			}
			if !ok {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
		}

		// Basic auth
		if user != "" && pass != "" {
			u, p, ok := c.Request.BasicAuth()
			if !ok || u != user || p != pass {
				c.Header("WWW-Authenticate", `Basic realm="swagger"`)
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
		}

		c.Next()
	}
}

func clientIP(c *gin.Context) string {
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	return strings.TrimSpace(c.ClientIP())
}