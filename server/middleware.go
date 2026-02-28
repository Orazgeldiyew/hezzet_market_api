// server/middleware.go
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

// ErrorMiddleware is a Gin post-handler middleware that converts any error pushed via
// c.Error(err) into a unified JSON error envelope and writes exactly one response.
//
// Registration order in the router matters: ErrorMiddleware must be registered
// AFTER RequestID so that the request ID is already in the context when errors
// are processed.
//
// Classification priority (first match wins):
//  1. *apperr.AppError — already classified by the service / handler layer.
//     Special case: if the AppError wraps a pg unique-violation the middleware
//     auto-promotes it to 409 CONFLICT so services don't have to detect it twice.
//  2. context.Canceled         → 499  REQUEST_CANCELED
//  3. context.DeadlineExceeded → 504  GATEWAY_TIMEOUT
//  4. pgconn.PgError 23505     → 409  CONFLICT  (unique violation)
//     pgconn.PgError 23503     → 409  CONFLICT  (fk violation)
//  5. validator.ValidationErrors → 400 VALIDATION_ERROR with per-field details
//  6. json.SyntaxError / json.UnmarshalTypeError → 400 VALIDATION_ERROR
//  7. jwt sentinel errors      → 401  UNAUTHORIZED  (safety-net; JWT middleware
//     normally converts these to AppError before they reach here)
//  8. Everything else          → 500  INTERNAL_ERROR (raw error logged, never exposed)
//
// All errors are logged server-side with the request_id. Raw pg / internal error
// details are NEVER included in the client-facing response.
func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		// Do not double-write if a handler already flushed a response
		// (e.g. file download, SSE stream).
		if c.Writer.Written() {
			return
		}

		requestID := middleware.GetRequestID(c)

		// Log every error. The first is the primary (used for the response);
		// subsequent ones are secondary (logged only).
		for i, ginErr := range c.Errors {
			label := "primary"
			if i > 0 {
				label = "secondary"
			}

			// For AppError wrapping an internal cause, log the root cause
			// so that 500s are debuggable from server logs.
			errMsg := ginErr.Err.Error()
			var appErr *apperr.AppError
			if errors.As(ginErr.Err, &appErr) && appErr.Err != nil {
				errMsg = appErr.Err.Error()
			}

			slog.ErrorContext(
				c.Request.Context(),
				"request error",
				slog.String("request_id", requestID),
				slog.String("error_label", label),
				slog.String("method", c.Request.Method),
				slog.String("path", c.Request.URL.Path),
				slog.String("client_ip", c.ClientIP()),
				slog.String("error", errMsg),
			)
		}

		r := classifyError(c.Errors[0].Err)

		c.JSON(r.status, response.APIResponse{
			Success:    false,
			StatusCode: r.status,
			Error: &response.APIError{
				Code:      r.code,
				Message:   r.message,
				Details:   r.details,
				RequestID: requestID,
			},
		})
	}
}

// ─── internal classification ─────────────────────────────────────────────────

// errResp is the middleware-internal classification result.
// It is never serialized directly; its fields are copied into response.APIError.
type errResp struct {
	status  int
	code    string
	message string
	details any
}

// classifyError maps any error to an errResp without leaking implementation details.
func classifyError(err error) errResp {
	// ── 1. AppError ──────────────────────────────────────────────────────────
	var appErr *apperr.AppError
	if errors.As(err, &appErr) {
		// If the service used Internal(pgErr) without detecting the pg error type,
		// auto-promote to a friendlier status so clients get 409 instead of 500.
		if appErr.Code == "INTERNAL_ERROR" && appErr.Err != nil {
			if r, ok := classifyPgError(appErr.Err); ok {
				return r
			}
		}
		return errResp{
			status:  appErr.HTTPStatus,
			code:    appErr.Code,
			message: appErr.Message,
			details: appErr.Details,
		}
	}

	// ── 2. Context cancellation / deadline ───────────────────────────────────
	if errors.Is(err, context.Canceled) {
		// 499 is a de-facto standard ("Client Closed Request") used by nginx/GCP.
		// Not in net/http constants; use the literal value.
		return errResp{status: 499, code: "REQUEST_CANCELED", message: "request canceled"}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errResp{
			status:  http.StatusGatewayTimeout,
			code:    "GATEWAY_TIMEOUT",
			message: "request timed out",
		}
	}

	// ── 3. PostgreSQL errors ──────────────────────────────────────────────────
	if r, ok := classifyPgError(err); ok {
		return r
	}

	// ── 4. go-playground/validator struct validation errors ───────────────────
	var valErrs validator.ValidationErrors
	if errors.As(err, &valErrs) {
		return errResp{
			status:  http.StatusBadRequest,
			code:    "VALIDATION_ERROR",
			message: "request validation failed",
			details: buildValidationDetails(valErrs),
		}
	}

	// ── 5. JSON binding errors (malformed request body) ───────────────────────
	var jsonSyntax *json.SyntaxError
	if errors.As(err, &jsonSyntax) {
		return errResp{
			status:  http.StatusBadRequest,
			code:    "VALIDATION_ERROR",
			message: "malformed JSON in request body",
		}
	}
	var jsonType *json.UnmarshalTypeError
	if errors.As(err, &jsonType) {
		return errResp{
			status:  http.StatusBadRequest,
			code:    "VALIDATION_ERROR",
			message: fmt.Sprintf("field %q expects type %s", jsonType.Field, jsonType.Type.String()),
		}
	}

	// ── 6. JWT library errors (safety-net) ────────────────────────────────────
	// The JWT middleware already converts these to AppError (step 1 above).
	// This branch handles the rare case where a raw jwt error reaches the middleware.
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return errResp{status: http.StatusUnauthorized, code: "UNAUTHORIZED", message: "token expired"}
	case errors.Is(err, jwt.ErrTokenSignatureInvalid),
		errors.Is(err, jwt.ErrTokenMalformed),
		errors.Is(err, jwt.ErrTokenNotValidYet),
		errors.Is(err, jwt.ErrTokenUnverifiable),
		errors.Is(err, jwt.ErrTokenInvalidClaims):
		return errResp{status: http.StatusUnauthorized, code: "UNAUTHORIZED", message: "unauthorized"}
	}

	// ── 7. Unknown / unhandled errors → never expose details ─────────────────
	return errResp{
		status:  http.StatusInternalServerError,
		code:    "INTERNAL_ERROR",
		message: "internal error",
	}
}

// classifyPgError converts known PostgreSQL SQLSTATE codes into friendly errResps.
// Returns (errResp, true) on a match; (zero, false) if err is not a *pgconn.PgError
// or the SQLSTATE is not handled here (falls through to INTERNAL_ERROR).
func classifyPgError(err error) (errResp, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return errResp{}, false
	}

	switch pgErr.Code {
	case "23505": // unique_violation
		return errResp{
			status:  http.StatusConflict,
			code:    "CONFLICT",
			message: pgUniqueMessage(pgErr.ConstraintName),
		}, true

	case "23503": // foreign_key_violation
		return errResp{
			status:  http.StatusConflict,
			code:    "CONFLICT",
			message: "related resource does not exist",
		}, true

	case "23502": // not_null_violation
		return errResp{
			status:  http.StatusBadRequest,
			code:    "VALIDATION_ERROR",
			message: fmt.Sprintf("field %q is required", pgErr.ColumnName),
		}, true
	}

	// Any other pg error is an unhandled internal error.
	return errResp{}, false
}

// ─── validation helpers ───────────────────────────────────────────────────────

// buildValidationDetails converts validator.ValidationErrors into a
// map[field → human-readable message] suitable for the response Details field.
//
// Field names are lowercased struct field names. To get JSON tag names instead
// (e.g. "user_id" rather than "userid"), register a custom TagNameFunc on Gin's
// validator engine at startup:
//
//	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
//	    v.RegisterTagNameFunc(func(fld reflect.StructField) string {
//	        name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
//	        if name == "-" { return "" }
//	        return name
//	    })
//	}
func buildValidationDetails(errs validator.ValidationErrors) map[string]string {
	out := make(map[string]string, len(errs))
	for _, e := range errs {
		out[validationFieldName(e)] = validationFieldMessage(e)
	}
	return out
}

// validationFieldName extracts the bare field name from the validator namespace
// (e.g. "LoginRequest.Username" → "username").
func validationFieldName(e validator.FieldError) string {
	ns := e.Namespace()
	if idx := strings.LastIndex(ns, "."); idx >= 0 {
		ns = ns[idx+1:]
	}
	return strings.ToLower(ns)
}

// validationFieldMessage returns a human-readable sentence for a FieldError.
func validationFieldMessage(e validator.FieldError) string {
	p := e.Param()
	switch e.Tag() {
	case "required":
		return "field is required"
	case "required_if", "required_with", "required_without":
		return "field is required"
	case "min":
		return fmt.Sprintf("must be at least %s characters long", p)
	case "max":
		return fmt.Sprintf("must be at most %s characters long", p)
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", p)
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", p)
	case "gt":
		return fmt.Sprintf("must be greater than %s", p)
	case "lt":
		return fmt.Sprintf("must be less than %s", p)
	case "len":
		return fmt.Sprintf("must be exactly %s characters long", p)
	case "email":
		return "must be a valid email address"
	case "url", "uri":
		return "must be a valid URL"
	case "uuid", "uuid4":
		return "must be a valid UUID"
	case "e164":
		return "must be a valid phone number in E.164 format"
	case "oneof":
		return fmt.Sprintf("must be one of: %s", strings.ReplaceAll(p, " ", ", "))
	case "alphanum":
		return "must contain only alphanumeric characters"
	case "alpha":
		return "must contain only alphabetic characters"
	case "numeric":
		return "must be a numeric value"
	case "boolean":
		return "must be a boolean value"
	case "unique":
		return "must contain unique values"
	default:
		return fmt.Sprintf("failed validation rule %q", e.Tag())
	}
}

// ─── PostgreSQL message helpers ───────────────────────────────────────────────

// pgUniqueMessage derives a friendly message from a PostgreSQL constraint name.
//
// PostgreSQL auto-names unique constraints as <table>_<column(s)>_key.
// Examples:
//
//	"users_username_key"   → "username already exists"
//	"products_sku_key"     → "sku already exists"
//	"roles_code_key"       → "code already exists"
func pgUniqueMessage(constraintName string) string {
	name := constraintName

	// Strip the table prefix (everything up to and including the first "_").
	if idx := strings.Index(name, "_"); idx >= 0 {
		name = name[idx+1:]
	}

	// Strip common PostgreSQL / explicit constraint suffixes.
	for _, suffix := range []string{"_key", "_unique", "_uniq", "_idx"} {
		name = strings.TrimSuffix(name, suffix)
	}

	name = strings.ReplaceAll(name, "_", " ")
	name = strings.TrimSpace(name)

	if name == "" {
		return "duplicate entry"
	}
	return fmt.Sprintf("%s already exists", name)
}
