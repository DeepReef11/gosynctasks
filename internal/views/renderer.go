package views

import (
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/internal/views/formatters"
	"strings"
)

// ViewRenderer renders tasks according to view configuration
type ViewRenderer struct {
	view    *View
	ctx     *formatters.FormatContext
	fmtMap  map[string]formatters.FieldFormatter
}

// NewViewRenderer creates a new view renderer
func NewViewRenderer(view *View, backend backend.TaskManager, dateFormat string) *ViewRenderer {
	if dateFormat == "" {
		dateFormat = view.Display.DateFormat
	}

	ctx := formatters.NewFormatContext(backend, dateFormat)

	renderer := &ViewRenderer{
		view:   view,
		ctx:    ctx,
		fmtMap: make(map[string]formatters.FieldFormatter),
	}

	// Initialize formatters for each field
	renderer.initializeFormatters()

	return renderer
}

// initializeFormatters creates formatter instances for all fields in the view
func (r *ViewRenderer) initializeFormatters() {
	for _, field := range r.view.Fields {
		var formatter formatters.FieldFormatter

		switch field.Name {
		case "status":
			formatter = formatters.NewStatusFormatter(r.ctx)
		case "priority":
			formatter = formatters.NewPriorityFormatter(r.ctx)
		case "summary":
			formatter = formatters.NewSummaryFormatter(r.ctx)
		case "description":
			formatter = formatters.NewDescriptionFormatter(r.ctx)
		case "due_date":
			formatter = formatters.NewDateFormatter(r.ctx, "due_date")
		case "start_date":
			formatter = formatters.NewDateFormatter(r.ctx, "start_date")
		case "created":
			formatter = formatters.NewDateFormatter(r.ctx, "created")
		case "modified":
			formatter = formatters.NewDateFormatter(r.ctx, "modified")
		case "completed":
			formatter = formatters.NewDateFormatter(r.ctx, "completed")
		case "tags":
			formatter = formatters.NewTagsFormatter(r.ctx)
		case "uid":
			formatter = formatters.NewUIDFormatter(r.ctx)
		case "parent":
			// Parent uses UID formatter
			formatter = formatters.NewUIDFormatter(r.ctx)
		}

		if formatter != nil {
			r.fmtMap[field.Name] = formatter
		}
	}
}

// RenderTask renders a single task according to the view configuration
func (r *ViewRenderer) RenderTask(task backend.Task) string {
	var result strings.Builder

	// Determine which fields to show and in what order
	fieldsToShow := r.getFieldsToShow()

	// Build field outputs
	fieldOutputs := make(map[string]string)
	for _, fieldName := range fieldsToShow {
		fieldConfig := r.getFieldConfig(fieldName)
		if fieldConfig == nil || (fieldConfig.Show != nil && !*fieldConfig.Show) {
			continue
		}

		formatter := r.fmtMap[fieldName]
		if formatter == nil {
			continue
		}

		output := formatter.Format(task, fieldConfig.Format, fieldConfig.Width, fieldConfig.Color)
		if output != "" {
			// Apply label if specified
			if fieldConfig.Label != "" {
				output = fieldConfig.Label + ": " + output
			}
			fieldOutputs[fieldName] = output
		}
	}

	// Render based on view type
	if r.view.Display.CompactMode {
		// Compact mode: all on one line
		result.WriteString("  ")
		parts := []string{}
		for _, fieldName := range fieldsToShow {
			if output, ok := fieldOutputs[fieldName]; ok && output != "" {
				parts = append(parts, output)
			}
		}
		result.WriteString(strings.Join(parts, " "))
		result.WriteString("\n")
	} else {
		// Standard mode: main line + optional description line + optional metadata line
		r.renderStandardMode(&result, fieldsToShow, fieldOutputs)
	}

	return result.String()
}

// renderStandardMode renders in standard (non-compact) mode
func (r *ViewRenderer) renderStandardMode(result *strings.Builder, fieldsToShow []string, fieldOutputs map[string]string) {
	// Main line: status + summary + dates
	mainFields := []string{"status", "summary", "start_date", "due_date"}
	mainParts := []string{}

	result.WriteString("  ")
	for _, fieldName := range mainFields {
		if output, ok := fieldOutputs[fieldName]; ok && output != "" {
			mainParts = append(mainParts, output)
		}
	}
	result.WriteString(strings.Join(mainParts, " "))
	result.WriteString("\n")

	// Description line (if present and not already shown)
	if desc, ok := fieldOutputs["description"]; ok && desc != "" {
		result.WriteString(fmt.Sprintf("     %s\n", desc))
	}

	// Metadata line: other fields (priority, tags, created, modified, etc.)
	metadataFields := []string{"created", "modified", "priority", "tags", "uid", "completed", "parent"}
	metadataParts := []string{}

	for _, fieldName := range metadataFields {
		if output, ok := fieldOutputs[fieldName]; ok && output != "" {
			// Skip if not in fieldsToShow
			inShow := false
			for _, f := range fieldsToShow {
				if f == fieldName {
					inShow = true
					break
				}
			}
			if inShow {
				metadataParts = append(metadataParts, output)
			}
		}
	}

	if len(metadataParts) > 0 {
		result.WriteString(fmt.Sprintf("     %s\n", strings.Join(metadataParts, " | ")))
	}
}

// getFieldsToShow returns the list of fields to display in order
func (r *ViewRenderer) getFieldsToShow() []string {
	// If field_order is specified, use it
	if len(r.view.FieldOrder) > 0 {
		return r.view.FieldOrder
	}

	// Otherwise, use the order from Fields array
	fields := []string{}
	for _, field := range r.view.Fields {
		// Show if: Show is nil (default to true) OR Show is true
		if field.Show == nil || *field.Show {
			fields = append(fields, field.Name)
		}
	}

	return fields
}

// getFieldConfig returns the field configuration by name
func (r *ViewRenderer) getFieldConfig(fieldName string) *FieldConfig {
	for i := range r.view.Fields {
		if r.view.Fields[i].Name == fieldName {
			return &r.view.Fields[i]
		}
	}
	return nil
}

// RenderTasks renders multiple tasks
func (r *ViewRenderer) RenderTasks(tasks []backend.Task) string {
	var result strings.Builder

	for _, task := range tasks {
		result.WriteString(r.RenderTask(task))
	}

	return result.String()
}

// GetFilters returns the view's filter configuration
func (r *ViewRenderer) GetFilters() *ViewFilters {
	return r.view.Filters
}

// GetSortConfig returns the view's sort configuration
func (r *ViewRenderer) GetSortConfig() (string, string) {
	return r.view.Display.SortBy, r.view.Display.SortOrder
}

// RenderTaskHierarchical renders a single task with hierarchical indentation
// prefix is the tree prefix (e.g., "├─ " or "└─ ")
func (r *ViewRenderer) RenderTaskHierarchical(task backend.Task, nodePrefix, childPrefix string) string {
	var result strings.Builder

	// Render the task normally
	taskOutput := r.RenderTask(task)

	// Add indentation to each line of the task output
	if nodePrefix != "" {
		lines := strings.Split(strings.TrimRight(taskOutput, "\n"), "\n")
		for j, line := range lines {
			if j == 0 {
				result.WriteString(nodePrefix)
			} else {
				// Continuation lines use the child prefix
				result.WriteString(childPrefix)
			}
			result.WriteString(line)
			result.WriteString("\n")
		}
	} else {
		result.WriteString(taskOutput)
	}

	return result.String()
}
