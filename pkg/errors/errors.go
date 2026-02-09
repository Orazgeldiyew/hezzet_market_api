package errors

type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
	Err        error  `json:"-"`
}

func (e *AppError) Error() string {
	return e.Message
}

func NotFound(code, msg string) *AppError {
	return &AppError{Code: code, Message: msg, HTTPStatus: 404}
}

func Validation(msg string) *AppError {
	return &AppError{Code: "VALIDATION_ERROR", Message: msg, HTTPStatus: 400}
}

func Internal(err error) *AppError {
	return &AppError{Code: "INTERNAL_ERROR", Message: "internal error", HTTPStatus: 500, Err: err}
}
