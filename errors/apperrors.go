package apperrors

import (
	"fmt"
	"net/http"
)

// AppError is a structured error type that carries an HTTP status code and a
// machine-readable code alongside the human-readable message. All service-layer
// functions return *AppError instead of plain errors, enabling handlers to
// extract status codes via errors.As() rather than fragile string matching.
type AppError struct {
	Code       string `json:"code"`        // Machine-readable: "NOT_FOUND", "CONFLICT", etc.
	Message    string `json:"message"`     // Human-readable message for the API consumer.
	StatusCode int    `json:"-"`           // HTTP status code to return.
	Err        error  `json:"-"`           // Wrapped original error for logging/debugging.
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap supports errors.Is and errors.As unwrapping.
func (e *AppError) Unwrap() error {
	return e.Err
}

// ── Constructors ────────────────────────────────────────────────────────────

// NotFound returns a 404 error indicating the requested resource does not exist.
func NotFound(resource, id string) *AppError {
	return &AppError{
		Code:       "NOT_FOUND",
		Message:    fmt.Sprintf("%s with id %s not found", resource, id),
		StatusCode: http.StatusNotFound,
	}
}

// Conflict returns a 409 error indicating a uniqueness or state conflict.
func Conflict(message string) *AppError {
	return &AppError{
		Code:       "CONFLICT",
		Message:    message,
		StatusCode: http.StatusConflict,
	}
}

// Validation returns a 400 error for invalid input or business rule violations.
func Validation(message string) *AppError {
	return &AppError{
		Code:       "VALIDATION_ERROR",
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
}

// Unauthorized returns a 401 error for authentication failures.
func Unauthorized(message string) *AppError {
	return &AppError{
		Code:       "UNAUTHORIZED",
		Message:    message,
		StatusCode: http.StatusUnauthorized,
	}
}

// Forbidden returns a 403 error for authorization failures.
func Forbidden(message string) *AppError {
	return &AppError{
		Code:       "FORBIDDEN",
		Message:    message,
		StatusCode: http.StatusForbidden,
	}
}

// Internal returns a 500 error wrapping an unexpected failure.
// The wrapped error is logged server-side but never exposed to the client.
func Internal(message string, err error) *AppError {
	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

// UnprocessableEntity returns a 422 error for semantically invalid requests.
func UnprocessableEntity(message string) *AppError {
	return &AppError{
		Code:       "UNPROCESSABLE_ENTITY",
		Message:    message,
		StatusCode: http.StatusUnprocessableEntity,
	}
}

// TooManyRequests returns a 429 error for rate limit violations.
func TooManyRequests(message string) *AppError {
	return &AppError{
		Code:       "RATE_LIMITED",
		Message:    message,
		StatusCode: http.StatusTooManyRequests,
	}
}
