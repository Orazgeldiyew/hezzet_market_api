package middleware

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

// AuthFromQuery reads JWT from ?token= query parameter (for browser access).
func AuthFromQuery(cfg config.Config, getTokenVersion TokenVersionFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.Query("token")
		if tokenStr == "" {
			c.Error(apperr.Unauthorized("token query parameter required"))
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

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.Error(apperr.Unauthorized("invalid token"))
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.Error(apperr.Unauthorized("invalid token"))
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
			c.Error(apperr.Unauthorized("invalid token"))
			c.Abort()
			return
		}

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

		dbVersion, err := getTokenVersion(c.Request.Context(), userID)
		if err != nil {
			c.Error(apperr.Unauthorized("unauthorized"))
			c.Abort()
			return
		}
		if int(tvFloat) != dbVersion {
			c.Error(apperr.Unauthorized("token revoked"))
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		if username, ok := claims["username"].(string); ok {
			c.Set("username", username)
		}

		c.Next()
	}
}
