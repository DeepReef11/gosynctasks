package formatters

import (
	"gosynctasks/backend"
	"strings"
)

// SummaryFormatter formats task summary field
type SummaryFormatter struct {
	ctx *FormatContext
}

// NewSummaryFormatter creates a new summary formatter
func NewSummaryFormatter(ctx *FormatContext) *SummaryFormatter {
	return &SummaryFormatter{ctx: ctx}
}

// Format formats the summary field according to the specified format
// Supported formats: full, truncate
func (f *SummaryFormatter) Format(task backend.Task, format string, width int, colorize bool) string {
	summary := task.Summary

	// Apply priority color if enabled
	if colorize && task.Priority > 0 && f.ctx.Backend != nil {
		priorityColor := f.ctx.Backend.GetPriorityColor(task.Priority)
		if format == "truncate" && width > 0 {
			summary = truncate(summary, width)
		}
		return priorityColor + "\033[1m" + summary + "\033[0m" // Bold + color
	}

	if format == "truncate" && width > 0 {
		return truncate(summary, width)
	}

	return summary
}

// DescriptionFormatter formats task description field
type DescriptionFormatter struct {
	ctx *FormatContext
}

// NewDescriptionFormatter creates a new description formatter
func NewDescriptionFormatter(ctx *FormatContext) *DescriptionFormatter {
	return &DescriptionFormatter{ctx: ctx}
}

// Format formats the description field according to the specified format
// Supported formats: full, truncate, first_line
func (f *DescriptionFormatter) Format(task backend.Task, format string, width int, colorize bool) string {
	if task.Description == "" {
		return ""
	}

	var result string

	switch format {
	case "full":
		result = task.Description
	case "truncate":
		result = f.formatTruncate(task.Description, width)
	case "first_line":
		result = f.formatFirstLine(task.Description, width)
	default:
		result = f.formatTruncate(task.Description, width)
	}

	// Description is typically shown in dim gray
	if colorize {
		return "\033[2m" + result + "\033[0m"
	}

	return result
}

// formatTruncate truncates description and replaces newlines with spaces
func (f *DescriptionFormatter) formatTruncate(description string, width int) string {
	// Replace newlines with spaces
	desc := strings.ReplaceAll(description, "\n", " ")
	desc = strings.ReplaceAll(desc, "\r", "")

	// Collapse multiple spaces
	desc = strings.Join(strings.Fields(desc), " ")

	if width > 0 && len(desc) > width {
		if width > 3 {
			return desc[:width-3] + "..."
		}
		return desc[:width]
	}

	return desc
}

// formatFirstLine returns only the first line of description
func (f *DescriptionFormatter) formatFirstLine(description string, width int) string {
	lines := strings.Split(description, "\n")
	if len(lines) == 0 {
		return ""
	}

	firstLine := strings.TrimSpace(lines[0])

	if width > 0 && len(firstLine) > width {
		if width > 3 {
			return firstLine[:width-3] + "..."
		}
		return firstLine[:width]
	}

	return firstLine
}

// TagsFormatter formats task tags/categories field
type TagsFormatter struct {
	ctx *FormatContext
}

// NewTagsFormatter creates a new tags formatter
func NewTagsFormatter(ctx *FormatContext) *TagsFormatter {
	return &TagsFormatter{ctx: ctx}
}

// Format formats the tags field according to the specified format
// Supported formats: list, comma, hash
func (f *TagsFormatter) Format(task backend.Task, format string, width int, colorize bool) string {
	if len(task.Categories) == 0 {
		return ""
	}

	var result string

	switch format {
	case "list":
		result = f.formatList(task.Categories)
	case "comma":
		result = f.formatComma(task.Categories)
	case "hash":
		result = f.formatHash(task.Categories)
	default:
		result = f.formatComma(task.Categories)
	}

	if width > 0 && len(result) > width {
		if width > 3 {
			result = result[:width-3] + "..."
		} else {
			result = result[:width]
		}
	}

	if colorize {
		return "\033[36m" + result + "\033[0m" // Cyan for tags
	}

	return result
}

// formatList formats tags as a bracketed list [tag1][tag2]
func (f *TagsFormatter) formatList(tags []string) string {
	if len(tags) == 0 {
		return ""
	}

	var parts []string
	for _, tag := range tags {
		parts = append(parts, "["+tag+"]")
	}
	return strings.Join(parts, "")
}

// formatComma formats tags as comma-separated list
func (f *TagsFormatter) formatComma(tags []string) string {
	return strings.Join(tags, ", ")
}

// formatHash formats tags with hash prefix #tag1 #tag2
func (f *TagsFormatter) formatHash(tags []string) string {
	if len(tags) == 0 {
		return ""
	}

	var parts []string
	for _, tag := range tags {
		parts = append(parts, "#"+tag)
	}
	return strings.Join(parts, " ")
}

// UIDFormatter formats task UID field
type UIDFormatter struct {
	ctx *FormatContext
}

// NewUIDFormatter creates a new UID formatter
func NewUIDFormatter(ctx *FormatContext) *UIDFormatter {
	return &UIDFormatter{ctx: ctx}
}

// Format formats the UID field according to the specified format
// Supported formats: full, short
func (f *UIDFormatter) Format(task backend.Task, format string, width int, colorize bool) string {
	if task.UID == "" {
		return ""
	}

	var result string

	switch format {
	case "full":
		result = task.UID
	case "short":
		result = f.formatShort(task.UID)
	default:
		result = f.formatShort(task.UID)
	}

	if width > 0 && len(result) > width {
		result = result[:width]
	}

	if colorize {
		return "\033[90m" + result + "\033[0m" // Gray for UID
	}

	return result
}

// formatShort returns abbreviated UID (first 8 characters)
func (f *UIDFormatter) formatShort(uid string) string {
	if len(uid) <= 8 {
		return uid
	}
	return uid[:8]
}
