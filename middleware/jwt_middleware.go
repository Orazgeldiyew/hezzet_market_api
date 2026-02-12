// middleware/jwt_middleware.go
package middleware

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

func AuthRequired(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.Error(apperr.Unauthorized("unauthorized"))
			c.Abort()
			return
		}

		if cfg.JWTSecret == "" {
			c.Error(apperr.Internal(fmt.Errorf("JWT_SECRET is empty")))
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.Error(apperr.Unauthorized("unauthorized"))
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.Error(apperr.Unauthorized("unauthorized"))
			c.Abort()
			return
		}

		sub, _ := claims.GetSubject()
		userID, err := strconv.ParseInt(sub, 10, 64)
		if err != nil || userID <= 0 {
			c.Error(apperr.Unauthorized("unauthorized"))
			c.Abort()
			return
		}
		c.Set("user_id", userID)

		if username, ok := claims["username"].(string); ok {
			c.Set("username", username)
		}

		var roles []string
		if rolesRaw, ok := claims["roles"].([]interface{}); ok {
			for _, r := range rolesRaw {
				if s, ok := r.(string); ok {
					roles = append(roles, s)
				}
			}
		} else if rolesStr, ok := claims["roles"].([]string); ok {
			roles = append(roles, rolesStr...)
		}
		if roles == nil {
			roles = []string{}
		}
		c.Set("roles", roles)

		c.Next()
	}
}
