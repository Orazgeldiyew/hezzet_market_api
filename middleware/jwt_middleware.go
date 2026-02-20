// middleware/jwt_middleware.go
package middleware

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

// TokenVersionFunc loads the user's current token_version from the DB.
// It must return an error if the user is deleted, disabled, or blocked.
type TokenVersionFunc func(ctx context.Context, userID int64) (int, error)

func AuthRequired(cfg config.Config, getTokenVersion TokenVersionFunc) gin.HandlerFunc {
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

		// token_version: FAIL-CLOSED (must exist and be numeric)
		rawTV, exists := claims["token_version"]
		if !exists {
			c.Error(apperr.Unauthorized("token missing token_version"))
			c.Abort()
			return
		}
		tvFloat, ok := rawTV.(float64)
		if !ok {
			c.Error(apperr.Unauthorized("invalid token_version type"))
			c.Abort()
			return
		}
		claimVersion := int(tvFloat)

		// Verify token_version against DB (also checks blocked/disabled/deleted)
		dbVersion, err := getTokenVersion(c.Request.Context(), userID)
		if err != nil {
			c.Error(apperr.Unauthorized("unauthorized"))
			c.Abort()
			return
		}
		if claimVersion != dbVersion {
			c.Error(apperr.Unauthorized("token revoked"))
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
				if s, ok := r.(string); ok && s != "" {
					roles = append(roles, s)
				}
			}
		} else if rolesStr, ok := claims["role"].([]string); ok {
			for _, s := range rolesStr {
				if s != "" {
					roles = append(roles, s)
				}
			}
		}
		if roles == nil {
			roles = []string{}
		}
		c.Set("roles", roles)

		c.Next()
	}
}