package formatters

import (
	"fmt"
	"gosynctasks/backend"
	"time"
)

// DateFormatter formats date fields
type DateFormatter struct {
	ctx       *FormatContext
	fieldName string // "due_date", "start_date", "created", "modified", "completed"
}

// NewDateFormatter creates a new date formatter
func NewDateFormatter(ctx *FormatContext, fieldName string) *DateFormatter {
	return &DateFormatter{
		ctx:       ctx,
		fieldName: fieldName,
	}
}

// Format formats the date field according to the specified format
// Supported formats: full, relative, short, date_only
func (f *DateFormatter) Format(task backend.Task, format string, width int, colorize bool) string {
	date := f.getDate(task)
	if date == nil {
		return ""
	}

	var result string

	switch format {
	case "full":
		result = f.formatFull(*date, colorize)
	case "relative":
		result = f.formatRelative(*date, colorize)
	case "short":
		result = f.formatShort(*date, colorize)
	case "date_only":
		result = f.formatDateOnly(*date, colorize)
	default:
		result = f.formatFull(*date, colorize)
	}

	return truncate(result, width)
}

// getDate extracts the appropriate date from the task
func (f *DateFormatter) getDate(task backend.Task) *time.Time {
	switch f.fieldName {
	case "due_date":
		return task.DueDate
	case "start_date":
		return task.StartDate
	case "created":
		if task.Created.IsZero() {
			return nil
		}
		return &task.Created
	case "modified":
		if task.Modified.IsZero() {
			return nil
		}
		return &task.Modified
	case "completed":
		return task.Completed
	default:
		return nil
	}
}

// formatFull returns full date with color coding based on date type
func (f *DateFormatter) formatFull(date time.Time, colorize bool) string {
	dateStr := date.Format(f.ctx.DateFormat)

	if !colorize {
		return dateStr
	}

	// Apply color based on field type and date relative to now
	color := f.getDateColor(date)
	if color != "" {
		return color + dateStr + "\033[0m"
	}

	return dateStr
}

// formatRelative returns relative time (e.g., "2 days ago", "in 3 days")
func (f *DateFormatter) formatRelative(date time.Time, colorize bool) string {
	duration := f.ctx.Now.Sub(date)

	var result string

	if duration < 0 {
		// Future date
		duration = -duration
		result = f.humanizeDuration(duration, "in ")
	} else {
		// Past date
		result = f.humanizeDuration(duration, "") + " ago"
	}

	if !colorize {
		return result
	}

	color := f.getDateColor(date)
	if color != "" {
		return color + result + "\033[0m"
	}

	return result
}

// formatShort returns short date format (e.g., "01/15", "Jan 15")
func (f *DateFormatter) formatShort(date time.Time, colorize bool) string {
	// Use short format from context or default
	shortFormat := "01/02"
	if f.ctx.DateFormat == "2006-01-02 15:04" {
		shortFormat = "01/02 15:04"
	}

	dateStr := date.Format(shortFormat)

	if !colorize {
		return dateStr
	}

	color := f.getDateColor(date)
	if color != "" {
		return color + dateStr + "\033[0m"
	}

	return dateStr
}

// formatDateOnly returns just the date part (no time)
func (f *DateFormatter) formatDateOnly(date time.Time, colorize bool) string {
	dateStr := date.Format("2006-01-02")

	if !colorize {
		return dateStr
	}

	color := f.getDateColor(date)
	if color != "" {
		return color + dateStr + "\033[0m"
	}

	return dateStr
}

// getDateColor returns the appropriate color for a date based on field type
func (f *DateFormatter) getDateColor(date time.Time) string {
	switch f.fieldName {
	case "due_date":
		return f.getDueDateColor(date)
	case "start_date":
		return f.getStartDateColor(date)
	default:
		return "" // No color for other date fields
	}
}

// getDueDateColor returns color for due dates
func (f *DateFormatter) getDueDateColor(date time.Time) string {
	if date.Before(f.ctx.Now) {
		return "\033[31m" // Red (overdue)
	} else if date.Sub(f.ctx.Now).Hours() < 24 {
		return "\033[33m" // Yellow (due soon)
	}
	return "\033[90m" // Gray (future)
}

// getStartDateColor returns color for start dates
func (f *DateFormatter) getStartDateColor(date time.Time) string {
	hoursDiff := date.Sub(f.ctx.Now).Hours()

	if date.Before(f.ctx.Now) {
		return "\033[36m" // Cyan (past - should have started)
	} else if hoursDiff < 72 { // Within 3 days
		return "\033[33m" // Yellow (starting soon)
	}
	return "\033[90m" // Gray (future)
}

// humanizeDuration converts duration to human-readable format
func (f *DateFormatter) humanizeDuration(d time.Duration, prefix string) string {
	seconds := int(d.Seconds())
	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24
	weeks := days / 7
	months := days / 30
	years := days / 365

	if years > 0 {
		return fmt.Sprintf("%s%dy", prefix, years)
	} else if months > 0 {
		return fmt.Sprintf("%s%dmo", prefix, months)
	} else if weeks > 0 {
		return fmt.Sprintf("%s%dw", prefix, weeks)
	} else if days > 0 {
		return fmt.Sprintf("%s%dd", prefix, days)
	} else if hours > 0 {
		return fmt.Sprintf("%s%dh", prefix, hours)
	} else if minutes > 0 {
		return fmt.Sprintf("%s%dm", prefix, minutes)
	}
	return fmt.Sprintf("%s%ds", prefix, seconds)
}
