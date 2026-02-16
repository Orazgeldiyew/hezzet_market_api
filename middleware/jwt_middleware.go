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

		secret := cfg.AccessTokenSecret
		if secret == "" {
			secret = cfg.JWTSecret
		}
		if secret == "" {
			c.Error(apperr.Internal(fmt.Errorf("ACCESS_TOKEN_SECRET is empty")))
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secret), nil
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

		if tokenType, _ := claims["type"].(string); tokenType != "access" {
			c.Error(apperr.Unauthorized("invalid token type"))
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
		if rolesRaw, ok := claims["role"].([]interface{}); ok {
			for _, r := range rolesRaw {
				if s, ok := r.(string); ok {
					roles = append(roles, s)
				}
			}
		} else if rolesStr, ok := claims["role"].([]string); ok {
			roles = append(roles, rolesStr...)
		}
		if roles == nil {
			roles = []string{}
		}
		c.Set("roles", roles)

		c.Next()
	}
}
