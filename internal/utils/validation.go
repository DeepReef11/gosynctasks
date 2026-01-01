package utils

import (
	"fmt"
	"time"
)

// ValidatePriority checks if priority is within valid range (0-9)
func ValidatePriority(priority int) error {
	if priority < 0 || priority > 9 {
		return ErrInvalidPriority(priority)
	}
	return nil
}

// ParseDateFlag parses a date string in ISO format (YYYY-MM-DD).
// Returns nil for empty strings (used to clear dates).
// Returns error for invalid formats or dates.
func ParseDateFlag(dateStr string) (*time.Time, error) {
	// Empty string means clear the date
	if dateStr == "" {
		return nil, nil
	}

	// Parse ISO date format (YYYY-MM-DD) in local timezone
	parsedDate, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		return nil, ErrInvalidDate(dateStr)
	}

	return &parsedDate, nil
}

// ValidateDates checks that start and due dates are logically consistent.
// If both are provided, start date must be before or equal to due date.
func ValidateDates(startDate, dueDate *time.Time) error {
	// If either is nil, no validation needed
	if startDate == nil || dueDate == nil {
		return nil
	}

	// Start date must be before or equal to due date
	if startDate.After(*dueDate) {
		return WrapWithSuggestion(
			fmt.Errorf("start date (%s) cannot be after due date (%s)",
				startDate.Format("2006-01-02"),
				dueDate.Format("2006-01-02")),
			"Make sure the start date is before or on the same day as the due date",
		)
	}

	return nil
}
