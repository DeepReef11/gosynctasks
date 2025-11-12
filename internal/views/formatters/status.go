package formatters

import (
	"gosynctasks/backend"
	"strings"
)

// StatusFormatter formats task status field
type StatusFormatter struct {
	ctx *FormatContext
}

// NewStatusFormatter creates a new status formatter
func NewStatusFormatter(ctx *FormatContext) *StatusFormatter {
	return &StatusFormatter{ctx: ctx}
}

// Format formats the status field according to the specified format
// Supported formats: symbol, text, emoji, short
func (f *StatusFormatter) Format(task backend.Task, format string, width int, color bool) string {
	var result string

	switch format {
	case "symbol":
		result = f.formatSymbol(task.Status, color)
	case "text":
		result = f.formatText(task.Status, color)
	case "emoji":
		result = f.formatEmoji(task.Status)
	case "short":
		result = f.formatShort(task.Status, color)
	default:
		result = f.formatSymbol(task.Status, color)
	}

	return truncate(result, width)
}

// formatSymbol returns colored symbols for each status
func (f *StatusFormatter) formatSymbol(status string, color bool) string {
	var statusColor string
	var statusSymbol string

	switch status {
	case "COMPLETED":
		statusColor = "\033[32m" // Green
		statusSymbol = "✓"
	case "IN-PROCESS":
		statusColor = "\033[33m" // Yellow
		statusSymbol = "●"
	case "CANCELLED":
		statusColor = "\033[31m" // Red
		statusSymbol = "✗"
	default: // NEEDS-ACTION
		statusColor = "\033[37m" // White
		statusSymbol = "○"
	}

	if color {
		return statusColor + statusSymbol + "\033[0m"
	}
	return statusSymbol
}

// formatText returns full text status with optional color
func (f *StatusFormatter) formatText(status string, color bool) string {
	var statusColor string
	var statusText string

	switch status {
	case "COMPLETED":
		statusColor = "\033[32m"
		statusText = "COMPLETED"
	case "IN-PROCESS":
		statusColor = "\033[33m"
		statusText = "IN-PROCESS"
	case "CANCELLED":
		statusColor = "\033[31m"
		statusText = "CANCELLED"
	default: // NEEDS-ACTION
		statusColor = "\033[37m"
		statusText = "TODO"
	}

	if color {
		return statusColor + statusText + "\033[0m"
	}
	return statusText
}

// formatEmoji returns emoji representations
func (f *StatusFormatter) formatEmoji(status string) string {
	switch status {
	case "COMPLETED":
		return "✅"
	case "IN-PROCESS":
		return "🔄"
	case "CANCELLED":
		return "❌"
	default: // NEEDS-ACTION
		return "⭕"
	}
}

// formatShort returns abbreviated status
func (f *StatusFormatter) formatShort(status string, color bool) string {
	var statusColor string
	var statusText string

	switch status {
	case "COMPLETED":
		statusColor = "\033[32m"
		statusText = "D"
	case "IN-PROCESS":
		statusColor = "\033[33m"
		statusText = "P"
	case "CANCELLED":
		statusColor = "\033[31m"
		statusText = "C"
	default: // NEEDS-ACTION
		statusColor = "\033[37m"
		statusText = "T"
	}

	if color {
		return statusColor + statusText + "\033[0m"
	}
	return statusText
}

// truncate truncates a string to the specified width (accounting for ANSI codes)
func truncate(s string, width int) string {
	if width <= 0 {
		return s
	}

	// Strip ANSI codes to measure actual visible length
	visibleLen := len(stripAnsi(s))
	if visibleLen <= width {
		return s
	}

	// Truncate visible content
	visible := stripAnsi(s)
	if width > 3 {
		visible = visible[:width-3] + "..."
	} else {
		visible = visible[:width]
	}

	// If original had color, try to preserve it (simplified)
	if strings.Contains(s, "\033[") {
		// Extract first color code
		colorStart := strings.Index(s, "\033[")
		colorEnd := strings.Index(s[colorStart:], "m")
		if colorEnd > 0 {
			colorCode := s[colorStart : colorStart+colorEnd+1]
			return colorCode + visible + "\033[0m"
		}
	}

	return visible
}

// stripAnsi removes ANSI color codes from a string
func stripAnsi(s string) string {
	var result []rune
	inEscape := false

	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}

		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}

		result = append(result, r)
	}

	return string(result)
}
