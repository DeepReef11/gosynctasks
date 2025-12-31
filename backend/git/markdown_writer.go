package git

import (
	"gosynctasks/backend"
	"fmt"
	"strings"
)

// MarkdownWriter writes backend.Task structures back to markdown format.
type MarkdownWriter struct{}

// NewMarkdownWriter creates a new markdown writer.
func NewMarkdownWriter() *MarkdownWriter {
	return &MarkdownWriter{}
}

// Write converts task lists to markdown format.
func (w *MarkdownWriter) Write(taskLists map[string][]backend.Task) string {
	var builder strings.Builder

	// Write marker at the top
	builder.WriteString(gitBackendMarker)
	builder.WriteString("\n\n")

	// Write each task list
	// Note: iteration order is non-deterministic, but that's okay for now
	for listName, tasks := range taskLists {
		// Write list header
		builder.WriteString(fmt.Sprintf("## %s\n", listName))

		// Write each task
		for _, task := range tasks {
			// Write checkbox with status
			checkbox := w.formatCheckbox(task.Status)
			builder.WriteString(fmt.Sprintf("- %s %s", checkbox, task.Summary))

			// Write tags
			tags := w.formatTags(task)
			if tags != "" {
				builder.WriteString(" " + tags)
			}

			builder.WriteString("\n")

			// Write description if present
			if task.Description != "" {
				// Indent description lines
				descLines := strings.Split(task.Description, "\n")
				for _, line := range descLines {
					builder.WriteString(fmt.Sprintf("  %s\n", line))
				}
			}
		}

		builder.WriteString("\n")
	}

	return builder.String()
}

// formatCheckbox converts task status to markdown checkbox.
func (w *MarkdownWriter) formatCheckbox(status string) string {
	switch status {
	case "DONE":
		return "[x]"
	case "PROCESSING":
		return "[>]"
	case "CANCELLED":
		return "[-]"
	default:
		return "[ ]"
	}
}

// formatTags formats task metadata as @tag:value pairs.
func (w *MarkdownWriter) formatTags(task backend.Task) string {
	var tags []string

	// Always include UID
	if task.UID != "" {
		tags = append(tags, fmt.Sprintf("@uid:%s", task.UID))
	}

	// Include priority if set
	if task.Priority > 0 {
		tags = append(tags, fmt.Sprintf("@priority:%d", task.Priority))
	}

	// Include dates if set
	if task.DueDate != nil {
		if !task.DueDate.IsZero() {
			tags = append(tags, fmt.Sprintf("@due:%s", task.DueDate.Format("2006-01-02")))
		}
	}

	if task.StartDate != nil {
		if !task.StartDate.IsZero() {
			tags = append(tags, fmt.Sprintf("@start:%s", task.StartDate.Format("2006-01-02")))
		}
	}

	if !task.Created.IsZero() {
		tags = append(tags, fmt.Sprintf("@created:%s", task.Created.Format("2006-01-02")))
	}

	if task.Completed != nil {
		if !task.Completed.IsZero() {
			tags = append(tags, fmt.Sprintf("@completed:%s", task.Completed.Format("2006-01-02")))
		}
	}

	return strings.Join(tags, " ")
}
