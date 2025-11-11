package backend

import "fmt"

// BackendError represents an error from a backend operation
// It provides structured error information including HTTP status codes,
// operation context, and the underlying error message
type BackendError struct {
	Operation  string // e.g., "DeleteTask", "GetTasks", "UpdateTask"
	StatusCode int    // HTTP status code (0 if not an HTTP error)
	Message    string // Human-readable error message
	TaskUID    string // Optional: affected task UID
	ListID     string // Optional: affected list ID
	Body       string // Optional: response body for debugging
	Err        error  // Optional: underlying error
}

// Error implements the error interface
func (e *BackendError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s failed with status %d: %s", e.Operation, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("%s failed: %s", e.Operation, e.Message)
}

// Unwrap returns the underlying error for error wrapping
func (e *BackendError) Unwrap() error {
	return e.Err
}

// IsNotFound returns true if the error is a 404 Not Found
func (e *BackendError) IsNotFound() bool {
	return e.StatusCode == 404
}

// IsUnauthorized returns true if the error is a 401 Unauthorized or 403 Forbidden
func (e *BackendError) IsUnauthorized() bool {
	return e.StatusCode == 401 || e.StatusCode == 403
}

// IsServerError returns true if the error is a 5xx server error
func (e *BackendError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// NewBackendError creates a new BackendError
func NewBackendError(operation string, statusCode int, message string) *BackendError {
	return &BackendError{
		Operation:  operation,
		StatusCode: statusCode,
		Message:    message,
	}
}

// WithTaskUID adds task UID to the error for context
func (e *BackendError) WithTaskUID(uid string) *BackendError {
	e.TaskUID = uid
	return e
}

// WithListID adds list ID to the error for context
func (e *BackendError) WithListID(listID string) *BackendError {
	e.ListID = listID
	return e
}

// WithBody adds the response body to the error for debugging
func (e *BackendError) WithBody(body string) *BackendError {
	e.Body = body
	return e
}

// WithError wraps an underlying error
func (e *BackendError) WithError(err error) *BackendError {
	e.Err = err
	return e
}
