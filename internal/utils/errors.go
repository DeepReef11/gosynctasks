package utils

import (
	"fmt"
	"strings"
)

// ErrorWithSuggestion wraps an error with a helpful suggestion for the user
type ErrorWithSuggestion struct {
	Err        error
	Suggestion string
}

// Error implements the error interface
func (e *ErrorWithSuggestion) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("%v\n\nSuggestion: %s", e.Err, e.Suggestion)
	}
	return e.Err.Error()
}

// Unwrap allows errors.Is and errors.As to work
func (e *ErrorWithSuggestion) Unwrap() error {
	return e.Err
}

// Common error constructors with suggestions

// ErrTaskNotFound creates an error when a task is not found
func ErrTaskNotFound(searchTerm string) error {
	return &ErrorWithSuggestion{
		Err:        fmt.Errorf("no tasks found matching '%s'", searchTerm),
		Suggestion: "Try using a different search term or run 'gosynctasks <list>' to see all tasks",
	}
}

// ErrListNotFound creates an error when a list is not found
func ErrListNotFound(listName string) error {
	return &ErrorWithSuggestion{
		Err:        fmt.Errorf("list '%s' not found", listName),
		Suggestion: "Run 'gosynctasks list' to see available lists",
	}
}

// ErrNoListsAvailable creates an error when no lists are available
func ErrNoListsAvailable() error {
	return &ErrorWithSuggestion{
		Err:        fmt.Errorf("no task lists available"),
		Suggestion: "Create a new list with 'gosynctasks list create <name>'",
	}
}

// ErrSyncNotEnabled creates an error when sync operations are attempted but sync is disabled
func ErrSyncNotEnabled() error {
	return &ErrorWithSuggestion{
		Err:        fmt.Errorf("sync is not enabled in configuration"),
		Suggestion: "Enable sync in ~/.config/gosynctasks/config.yaml by setting 'sync.enabled: true'",
	}
}

// ErrBackendNotConfigured creates an error when a backend is not configured
func ErrBackendNotConfigured(backendName string) error {
	return &ErrorWithSuggestion{
		Err:        fmt.Errorf("backend '%s' is not configured", backendName),
		Suggestion: fmt.Sprintf("Add backend configuration to ~/.config/gosynctasks/config.yaml under 'backends.%s'", backendName),
	}
}

// ErrBackendOffline creates an error when a backend is offline
func ErrBackendOffline(backendName, reason string) error {
	suggestion := "Check your internet connection and try again"
	if strings.Contains(reason, "DNS") {
		suggestion = "Check your DNS settings and internet connection"
	} else if strings.Contains(reason, "refused") {
		suggestion = "Check if the server is running and accessible"
	} else if strings.Contains(reason, "timeout") {
		suggestion = "The server may be slow or unreachable. Try again later"
	}

	return &ErrorWithSuggestion{
		Err:        fmt.Errorf("backend '%s' is offline: %s", backendName, reason),
		Suggestion: suggestion,
	}
}

// ErrInvalidPriority creates an error for invalid priority values
func ErrInvalidPriority(priority int) error {
	return &ErrorWithSuggestion{
		Err:        fmt.Errorf("invalid priority %d", priority),
		Suggestion: "Priority must be between 0 (no priority) and 9 (highest priority)",
	}
}

// ErrInvalidDate creates an error for invalid date formats
func ErrInvalidDate(dateStr string) error {
	return &ErrorWithSuggestion{
		Err:        fmt.Errorf("invalid date format: %s", dateStr),
		Suggestion: "Use YYYY-MM-DD format (e.g., 2026-01-15)",
	}
}

// ErrInvalidStatus creates an error for invalid status values
func ErrInvalidStatus(status string, validStatuses []string) error {
	return &ErrorWithSuggestion{
		Err:        fmt.Errorf("invalid status: %s", status),
		Suggestion: fmt.Sprintf("Valid statuses: %s", strings.Join(validStatuses, ", ")),
	}
}

// ErrCredentialsNotFound creates an error when credentials are not found
func ErrCredentialsNotFound(backend, username string) error {
	return &ErrorWithSuggestion{
		Err:        fmt.Errorf("credentials not found for %s (user: %s)", backend, username),
		Suggestion: fmt.Sprintf("Store credentials with 'gosynctasks credentials set %s %s --prompt'", backend, username),
	}
}

// ErrAuthenticationFailed creates an error when authentication fails
func ErrAuthenticationFailed(backend string) error {
	return &ErrorWithSuggestion{
		Err:        fmt.Errorf("authentication failed for %s", backend),
		Suggestion: "Check your credentials with 'gosynctasks credentials get <backend> <user>' and update if needed",
	}
}

// ErrConfigFileNotFound creates an error when config file is not found
func ErrConfigFileNotFound(path string) error {
	return &ErrorWithSuggestion{
		Err:        fmt.Errorf("config file not found at %s", path),
		Suggestion: "Run gosynctasks to create a default configuration file",
	}
}

// ErrInvalidConfig creates an error for invalid configuration
func ErrInvalidConfig(field string, reason string) error {
	return &ErrorWithSuggestion{
		Err:        fmt.Errorf("invalid configuration for '%s': %s", field, reason),
		Suggestion: fmt.Sprintf("Check ~/.config/gosynctasks/config.yaml and fix the '%s' field", field),
	}
}

// WrapWithSuggestion wraps an existing error with a suggestion
func WrapWithSuggestion(err error, suggestion string) error {
	if err == nil {
		return nil
	}
	return &ErrorWithSuggestion{
		Err:        err,
		Suggestion: suggestion,
	}
}
