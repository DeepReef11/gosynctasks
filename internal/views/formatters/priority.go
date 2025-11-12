package formatters

import (
	"fmt"
	"gosynctasks/backend"
	"strings"
)

// PriorityFormatter formats task priority field
type PriorityFormatter struct {
	ctx *FormatContext
}

// NewPriorityFormatter creates a new priority formatter
func NewPriorityFormatter(ctx *FormatContext) *PriorityFormatter {
	return &PriorityFormatter{ctx: ctx}
}

// Format formats the priority field according to the specified format
// Supported formats: number, text, stars, color
func (f *PriorityFormatter) Format(task backend.Task, format string, width int, colorize bool) string {
	var result string

	switch format {
	case "number":
		result = f.formatNumber(task.Priority, colorize)
	case "text":
		result = f.formatText(task.Priority, colorize)
	case "stars":
		result = f.formatStars(task.Priority, colorize)
	case "color":
		result = f.formatColorBar(task.Priority)
	default:
		result = f.formatNumber(task.Priority, colorize)
	}

	return truncate(result, width)
}

// formatNumber returns priority as a number with optional color
func (f *PriorityFormatter) formatNumber(priority int, colorize bool) string {
	if priority == 0 {
		return "-"
	}

	numStr := fmt.Sprintf("%d", priority)

	if colorize && f.ctx.Backend != nil {
		color := f.ctx.Backend.GetPriorityColor(priority)
		return color + numStr + "\033[0m"
	}

	return numStr
}

// formatText returns priority as text (High/Medium/Low)
func (f *PriorityFormatter) formatText(priority int, colorize bool) string {
	var text string
	var color string

	if priority == 0 {
		return "None"
	}

	// Priority mapping: 1-3=High, 4-6=Medium, 7-9=Low
	switch {
	case priority >= 1 && priority <= 3:
		text = "High"
		color = "\033[31m" // Red
	case priority >= 4 && priority <= 6:
		text = "Medium"
		color = "\033[33m" // Yellow
	case priority >= 7 && priority <= 9:
		text = "Low"
		color = "\033[34m" // Blue
	default:
		text = fmt.Sprintf("P%d", priority)
		color = "\033[37m" // White
	}

	if colorize {
		return color + text + "\033[0m"
	}
	return text
}

// formatStars returns priority as stars (more stars = higher priority)
// Priority 1-3: ★★★ (high)
// Priority 4-6: ★★ (medium)
// Priority 7-9: ★ (low)
func (f *PriorityFormatter) formatStars(priority int, colorize bool) string {
	if priority == 0 {
		return "☆"
	}

	var stars string
	var color string

	switch {
	case priority >= 1 && priority <= 3:
		stars = "★★★"
		color = "\033[31m" // Red
	case priority >= 4 && priority <= 6:
		stars = "★★"
		color = "\033[33m" // Yellow
	case priority >= 7 && priority <= 9:
		stars = "★"
		color = "\033[34m" // Blue
	default:
		stars = "?"
		color = "\033[37m"
	}

	if colorize {
		return color + stars + "\033[0m"
	}
	return stars
}

// formatColorBar returns priority as a colored bar
func (f *PriorityFormatter) formatColorBar(priority int) string {
	if priority == 0 {
		return "\033[90m━\033[0m" // Gray bar for no priority
	}

	// Invert priority for bar length (1=highest=longest, 9=lowest=shortest)
	barLength := 10 - priority
	if barLength < 1 {
		barLength = 1
	}

	var color string
	switch {
	case priority >= 1 && priority <= 3:
		color = "\033[31m" // Red
	case priority >= 4 && priority <= 6:
		color = "\033[33m" // Yellow
	case priority >= 7 && priority <= 9:
		color = "\033[34m" // Blue
	default:
		color = "\033[37m"
	}

	bar := strings.Repeat("█", barLength)
	return color + bar + "\033[0m"
}
