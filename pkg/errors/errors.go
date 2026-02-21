package errors

// AppError is the canonical application error type.
//
// Rules:
//   - Err    — the root cause; NEVER serialized or sent to clients; used for server-side logging.
//   - Details — optional structured data (e.g. per-field validation map); passed to the response
//     layer explicitly; NOT serialized from this struct directly.
//   - Code   — machine-readable string constant understood by the frontend.
//   - Message — safe, human-readable sentence; may be shown directly in UI.
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
	Details    any    `json:"-"`
	Err        error  `json:"-"`
}

func (e *AppError) Error() string { return e.Message }

// Unwrap enables errors.Is / errors.As to traverse the causal chain.
func (e *AppError) Unwrap() error { return e.Err }

// Validation returns 400 VALIDATION_ERROR.
// The optional second argument is forwarded as structured details (e.g. map[string]string)
// in the JSON response so the frontend can highlight individual fields.
// All existing call-sites that pass only msg continue to compile unchanged.
func Validation(msg string, details ...any) *AppError {
	var det any
	if len(details) > 0 {
		det = details[0]
	}
	return &AppError{Code: "VALIDATION_ERROR", Message: msg, HTTPStatus: 400, Details: det}
}

// Unauthorized returns 401 UNAUTHORIZED.
func Unauthorized(msg string) *AppError {
	return &AppError{Code: "UNAUTHORIZED", Message: msg, HTTPStatus: 401}
}

// Forbidden returns 403 FORBIDDEN.
func Forbidden(msg string) *AppError {
	return &AppError{Code: "FORBIDDEN", Message: msg, HTTPStatus: 403}
}

// NotFound returns 404 with a domain-specific code (e.g. "USER_NOT_FOUND").
func NotFound(code, msg string) *AppError {
	return &AppError{Code: code, Message: msg, HTTPStatus: 404}
}

// Conflict returns 409 with a domain-specific code (e.g. "DUPLICATE_USERNAME").
func Conflict(code, msg string) *AppError {
	return &AppError{Code: code, Message: msg, HTTPStatus: 409}
}

// Internal wraps a root-cause error and returns 500 INTERNAL_ERROR.
// The raw error is NEVER exposed to clients; it is stored in Err so the
// error middleware can log it with the request_id for server-side debugging.
func Internal(err error) *AppError {
	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    "internal error",
		HTTPStatus: 500,
		Err:        err,
	}
}
