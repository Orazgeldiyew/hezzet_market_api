// server/middleware.go
package server

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

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

		// Gin validation / JSON binding errors → 400
		switch err.(type) {
		case validator.ValidationErrors, *json.SyntaxError, *json.UnmarshalTypeError:
			response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}

		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}
