package domain

import (
	"errors"
	"fmt"
)

// Domain-specific errors for better error handling and user feedback
var (
	// ErrURLNotFound is returned when a short code doesn't exist
	ErrURLNotFound = errors.New("URL not found")
	
	// ErrURLExpired is returned when accessing an expired URL
	ErrURLExpired = errors.New("URL has expired")
	
	// ErrInvalidURL is returned when the provided URL is invalid
	ErrInvalidURL = errors.New("invalid URL format")
	
	// ErrShortCodeTaken is returned when a custom alias is already in use
	ErrShortCodeTaken = errors.New("short code already exists")
	
	// ErrShortCodeInvalid is returned when a short code has invalid characters
	ErrShortCodeInvalid = errors.New("short code contains invalid characters")
	
	// ErrRateLimitExceeded is returned when rate limit is hit
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	
	// ErrDatabaseConnection is returned for database connectivity issues
	ErrDatabaseConnection = errors.New("database connection error")
	
	// ErrCacheUnavailable is returned when cache operations fail
	ErrCacheUnavailable = errors.New("cache temporarily unavailable")
)

// AppError wraps errors with additional context for better debugging
type AppError struct {
	Err        error  // Original error
	Message    string // User-friendly message
	StatusCode int    // HTTP status code
	Internal   bool   // Whether to log as internal error
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

// Unwrap returns the wrapped error for errors.Is and errors.As
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new application error with context
func NewAppError(err error, message string, statusCode int, internal bool) *AppError {
	return &AppError{
		Err:        err,
		Message:    message,
		StatusCode: statusCode,
		Internal:   internal,
	}
}

// NewNotFoundError creates a 404 error
func NewNotFoundError(resource string) *AppError {
	return &AppError{
		Err:        ErrURLNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		StatusCode: 404,
		Internal:   false,
	}
}

// NewValidationError creates a 400 validation error
func NewValidationError(message string) *AppError {
	return &AppError{
		Err:        ErrInvalidURL,
		Message:    message,
		StatusCode: 400,
		Internal:   false,
	}
}

// NewInternalError creates a 500 internal server error
func NewInternalError(err error) *AppError {
	return &AppError{
		Err:        err,
		Message:    "Internal server error occurred",
		StatusCode: 500,
		Internal:   true, // Log this error
	}
}