package errors

type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
	Err        error  `json:"-"`
}

func (e *AppError) Error() string { return e.Message }

// полезно для errors.Is / errors.As
func (e *AppError) Unwrap() error { return e.Err }

func NotFound(code, msg string) *AppError {
	return &AppError{Code: code, Message: msg, HTTPStatus: 404}
}

func Validation(msg string) *AppError {
	return &AppError{Code: "VALIDATION_ERROR", Message: msg, HTTPStatus: 400}
}

func Internal(err error) *AppError {
	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    "internal error",
		HTTPStatus: 500,
		Err:        err,
	}
}

func Unauthorized(msg string) *AppError {
	return &AppError{Code: "UNAUTHORIZED", Message: msg, HTTPStatus: 401}
}

func Forbidden(msg string) *AppError {
	return &AppError{Code: "FORBIDDEN", Message: msg, HTTPStatus: 403}
}
func Conflict(code, msg string) *AppError {
	return &AppError{Code: code, Message: msg, HTTPStatus: 409}
}
