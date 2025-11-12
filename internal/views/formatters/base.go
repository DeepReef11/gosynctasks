package formatters

import (
	"gosynctasks/backend"
	"time"
)

// FieldFormatter is the base interface for all field formatters
type FieldFormatter interface {
	// Format returns the formatted string representation of a field value
	Format(task backend.Task, format string, width int, color bool) string
}

// FormatContext provides additional context for formatting
type FormatContext struct {
	// DateFormat is the Go time format string for date display
	DateFormat string

	// Backend provides backend-specific functionality (e.g., priority colors)
	Backend backend.TaskManager

	// Now is the current time (useful for relative date calculations)
	Now time.Time
}

// NewFormatContext creates a new format context with default values
func NewFormatContext(backend backend.TaskManager, dateFormat string) *FormatContext {
	if dateFormat == "" {
		dateFormat = "2006-01-02"
	}

	return &FormatContext{
		DateFormat: dateFormat,
		Backend:    backend,
		Now:        time.Now(),
	}
}
