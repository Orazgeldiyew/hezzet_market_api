package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
			return
		}

		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}
