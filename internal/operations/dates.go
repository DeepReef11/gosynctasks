package operations

import (
	"fmt"
	"time"
)

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
		return nil, fmt.Errorf("invalid date format '%s': expected YYYY-MM-DD (e.g., 2025-01-31)", dateStr)
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
		return fmt.Errorf("start date (%s) cannot be after due date (%s)",
			startDate.Format("2006-01-02"),
			dueDate.Format("2006-01-02"))
	}

	return nil
}
