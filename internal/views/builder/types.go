// Package builder provides an interactive terminal UI for creating custom task views.
//
// The builder uses a state machine to guide users through the view creation process:
//   1. Welcome screen with overview
//   2. Basic info (view description)
//   3. Field selection (which task fields to display)
//   4. Field ordering (arrange field display order)
//   5. Field configuration (customize formats, colors, widths)
//   6. Display options (headers, borders, sorting)
//   7. Confirmation (review and save)
//
// Usage:
//
//	view, err := builder.Run("my-view")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// Use the created view...
//
// The builder validates input at each step, preventing invalid configurations
// and providing helpful error messages. All validation is performed against
// the views.FieldRegistry to ensure only valid field names and formats are used.
package builder

import (
	"gosynctasks/internal/views"
)

// BuilderState represents the current state in the view builder state machine.
// The builder progresses through a sequence of states, collecting configuration
// at each step. State transitions are triggered by user actions (Enter key).
// At any non-terminal state, the user can cancel with Ctrl+C or Esc.
type BuilderState int

const (
	// StateWelcome is the initial welcome screen showing an overview
	StateWelcome BuilderState = iota

	// StateBasicInfo collects the view description (optional text input)
	StateBasicInfo

	// StateFieldSelection allows users to select which task fields to display.
	// At least one field must be selected to proceed.
	StateFieldSelection

	// StateFieldOrdering allows users to arrange the display order of selected fields.
	// Fields can be moved up/down with Ctrl+Up/Down keys.
	StateFieldOrdering

	// StateFieldConfig allows customization of each field's display settings:
	// color coding, custom width, and display format.
	StateFieldConfig

	// StateDisplayOptions configures overall display settings:
	// show header, show border, compact mode, date format, and sorting.
	StateDisplayOptions

	// StateConfirm shows a preview and asks for final confirmation (Y/N).
	StateConfirm

	// StateDone is reached when the user confirms and the view is successfully built.
	// This is a terminal state.
	StateDone

	// StateCancelled is reached when the user cancels or an unrecoverable error occurs.
	// This is a terminal state.
	StateCancelled
)

// String returns the string representation of the state
func (s BuilderState) String() string {
	switch s {
		case StateWelcome:
			return "Welcome"
		case StateBasicInfo:
			return "Basic Info"
		case StateFieldSelection:
			return "Field Selection"
		case StateFieldOrdering:
			return "Field Ordering"
		case StateFieldConfig:
			return "Field Configuration"
		case StateDisplayOptions:
			return "Display Options"
		case StateConfirm:
			return "Confirm"
		case StateDone:
			return "Done"
		case StateCancelled:
			return "Cancelled"
		default:
			return "Unknown"
	}
}

// FieldItem represents a task field with its display configuration and selection state.
// FieldItems are initialized from views.FieldRegistry and track both user selections
// and display customizations.
type FieldItem struct {
	// Name is the field identifier (e.g., "status", "summary", "priority").
	// Must match a key in views.FieldRegistry.
	Name string

	// Description provides user-friendly explanation of what the field represents.
	Description string

	// Selected indicates whether this field is included in the view.
	Selected bool

	// Format specifies the display format for this field (e.g., "symbol", "full", "truncate").
	// Must be a valid format for this field type according to views.FieldRegistry.
	// If empty, the default format for the field will be used.
	Format string

	// Width specifies the maximum width for this field in characters (0 = no limit).
	// Valid range is 0-200.
	Width int

	// Color enables or disables color coding for this field.
	// Color meaning depends on the field type (e.g., priority uses backend-specific colors).
	Color bool

	// Label provides a custom display label for this field.
	// If empty, the field name will be used as the label.
	Label string
}

// ViewBuilder holds the state for building a view through the interactive builder.
// It maintains all configuration collected during the build process and manages
// the state machine transitions.
//
// ViewBuilder is not thread-safe and should only be used from a single goroutine
// (typically the bubbletea UI event loop).
type ViewBuilder struct {
	// ViewName is the unique identifier for the view being created.
	// Must be 1-50 characters, alphanumeric with underscores and hyphens only.
	ViewName string

	// ViewDescription is an optional human-readable description of the view's purpose.
	ViewDescription string

	// AvailableFields contains all possible task fields with their current configuration.
	// Initialized from views.FieldRegistry in NewViewBuilder.
	AvailableFields []FieldItem

	// SelectedFields contains the names of fields the user has selected to display.
	// Updated by calling UpdateSelectedFields().
	SelectedFields []string

	// FieldOrder specifies the display order of selected fields.
	// Updated by calling UpdateFieldOrder().
	FieldOrder []string

	// CurrentFieldIndex tracks which field is being configured in StateFieldConfig.
	// Not used in other states.
	CurrentFieldIndex int

	// ShowHeader controls whether the task list header is displayed.
	ShowHeader bool

	// ShowBorder controls whether borders are drawn around the task list.
	ShowBorder bool

	// CompactMode reduces spacing between tasks when enabled.
	CompactMode bool

	// DateFormat specifies the Go time format string for dates (e.g., "2006-01-02").
	DateFormat string

	// SortBy specifies which field to sort tasks by.
	// Must be one of: status, summary, priority, due_date, start_date, created, modified.
	SortBy string

	// SortOrder specifies the sort direction: "asc" or "desc".
	SortOrder string

	// CurrentState tracks the builder's position in the state machine.
	CurrentState BuilderState

	// View contains the final built view when CurrentState == StateDone.
	// Nil until BuildView() is called successfully.
	View *views.View

	// Err contains any error that occurred during the build process.
	// Non-nil when CurrentState == StateCancelled due to error.
	Err error
}

// NewViewBuilder creates a new view builder with the given name.
//
// The builder is initialized with:
//   - All available fields from views.FieldRegistry
//   - Status and summary fields pre-selected
//   - Default display options (header on, border on, compact off)
//   - Default date format ("2006-01-02")
//   - Initial state set to StateWelcome
//
// The name parameter becomes the view's unique identifier. It must be validated
// before use - call ValidateViewName() or proceed through the builder UI which
// validates automatically.
//
// Example:
//
//	builder := NewViewBuilder("my-tasks")
//	// Interact with builder through UI or programmatically...
func NewViewBuilder(name string) *ViewBuilder {
	// Initialize available fields from field registry
	availableFields := []FieldItem{
		{Name: "status", Description: "Task completion status (TODO, DONE, PROCESSING, CANCELLED)", Selected: true, Format: "symbol"},
		{Name: "summary", Description: "Task title/summary", Selected: true, Format: "full"},
		{Name: "description", Description: "Task detailed description", Selected: false, Format: "truncate", Width: 70},
		{Name: "priority", Description: "Task priority (0-9, 1=highest)", Selected: false, Format: "number", Color: true},
		{Name: "due_date", Description: "Task due date/deadline", Selected: false, Format: "full", Color: true},
		{Name: "start_date", Description: "Task start date", Selected: false, Format: "full", Color: true},
		{Name: "created", Description: "Task creation timestamp", Selected: false, Format: "full"},
		{Name: "modified", Description: "Task last modified timestamp", Selected: false, Format: "full"},
		{Name: "completed", Description: "Task completion timestamp", Selected: false, Format: "full"},
		{Name: "tags", Description: "Task categories/labels", Selected: false, Format: "comma"},
		{Name: "uid", Description: "Unique task identifier", Selected: false, Format: "short"},
		{Name: "parent", Description: "Parent task UID (for subtasks)", Selected: false, Format: "short"},
	}

	return &ViewBuilder{
		ViewName:        name,
		ViewDescription: "",
		AvailableFields: availableFields,
		SelectedFields:  []string{},
		FieldOrder:      []string{},
		CurrentState:    StateWelcome,
		ShowHeader:      true,
		ShowBorder:      true,
		CompactMode:     false,
		DateFormat:      "2006-01-02",
		SortBy:          "",
		SortOrder:       "asc",
	}
}

// BuildView constructs the final View from the builder state.
//
// This method converts the ViewBuilder's configuration into a views.View struct,
// performing comprehensive validation of the complete view configuration.
//
// BuildView should be called after all configuration steps are complete, typically
// when the builder reaches StateConfirm and the user accepts.
//
// Validation performed:
//   - View name format and length
//   - At least one field is selected
//   - All field formats are valid for their types
//   - All field widths are in valid range (0-200)
//   - Field order is consistent with selected fields
//   - Display options are valid
//
// Returns the created view on success, or a views.ValidationError on validation failure.
//
// Example:
//
//	view, err := builder.BuildView()
//	if err != nil {
//	    var validationErr *views.ValidationError
//	    if errors.As(err, &validationErr) {
//	        fmt.Printf("Validation failed: %s\n", validationErr.Message)
//	    }
//	    return err
//	}
//	// Save or use the view...
func (b *ViewBuilder) BuildView() (*views.View, error) {
	// Collect selected fields
	var fields []views.FieldConfig

	for _, fieldName := range b.FieldOrder {
		// Find the field item
		var item *FieldItem
		for i := range b.AvailableFields {
			if b.AvailableFields[i].Name == fieldName {
				item = &b.AvailableFields[i]
				break
			}
		}

		if item == nil || !item.Selected {
			continue
		}

		trueVal := true
		field := views.FieldConfig{
			Name:   item.Name,
			Format: item.Format,
			Show:   &trueVal,
			Color:  item.Color,
		}

		if item.Width > 0 {
			field.Width = item.Width
		}

		if item.Label != "" {
			field.Label = item.Label
		}

		fields = append(fields, field)
	}

	view := &views.View{
		Name:        b.ViewName,
		Description: b.ViewDescription,
		Fields:      fields,
		FieldOrder:  b.FieldOrder,
		Display: views.DisplayOptions{
			ShowHeader:  b.ShowHeader,
			ShowBorder:  b.ShowBorder,
			CompactMode: b.CompactMode,
			DateFormat:  b.DateFormat,
			SortBy:      b.SortBy,
			SortOrder:   b.SortOrder,
		},
	}

	// Validate the complete view
	if err := views.ValidateView(view); err != nil {
		return nil, err
	}

	return view, nil
}

// UpdateSelectedFields updates the SelectedFields slice to match the current
// selection state of AvailableFields.
//
// This method should be called after any changes to field selection (toggling
// fields in StateFieldSelection) and before UpdateFieldOrder().
//
// The method clears SelectedFields and rebuilds it by iterating through
// AvailableFields and collecting the names of all fields where Selected == true.
//
// Example:
//
//	builder.AvailableFields[2].Selected = true  // Select a field
//	builder.UpdateSelectedFields()               // Sync SelectedFields
//	fmt.Println(builder.SelectedFields)          // Contains the new selection
func (b *ViewBuilder) UpdateSelectedFields() {
	b.SelectedFields = []string{}
	for _, field := range b.AvailableFields {
		if field.Selected {
			b.SelectedFields = append(b.SelectedFields, field.Name)
		}
	}
}

// UpdateFieldOrder updates the FieldOrder to reflect the current SelectedFields,
// preserving any custom ordering that was already set.
//
// Behavior:
//   - If FieldOrder is empty: Initialize it with SelectedFields in their current order
//   - If FieldOrder exists: Remove unselected fields, add newly selected fields at the end,
//     and preserve the relative order of fields that remain selected
//
// This method should be called after UpdateSelectedFields() and before proceeding
// to StateFieldOrdering or StateFieldConfig.
//
// Example - Initial setup:
//
//	builder.UpdateSelectedFields()  // SelectedFields = ["status", "summary"]
//	builder.UpdateFieldOrder()      // FieldOrder = ["status", "summary"]
//
// Example - Preserving custom order:
//
//	// User arranged: FieldOrder = ["summary", "priority", "status"]
//	builder.AvailableFields[1].Selected = false  // Deselect priority
//	builder.UpdateSelectedFields()                // SelectedFields = ["status", "summary"]
//	builder.UpdateFieldOrder()                    // FieldOrder = ["summary", "status"] (order preserved)
func (b *ViewBuilder) UpdateFieldOrder() {
	if len(b.FieldOrder) == 0 {
		// Initialize field order with currently selected fields
		b.FieldOrder = append([]string{}, b.SelectedFields...)
	} else {
		// Remove unselected fields from order
		newOrder := []string{}
		for _, fieldName := range b.FieldOrder {
			for _, selected := range b.SelectedFields {
				if fieldName == selected {
					newOrder = append(newOrder, fieldName)
					break
				}
			}
		}

		// Add newly selected fields that aren't in order yet
		for _, selected := range b.SelectedFields {
			found := false
			for _, ordered := range newOrder {
				if selected == ordered {
					found = true
					break
				}
			}
			if !found {
				newOrder = append(newOrder, selected)
			}
		}

		b.FieldOrder = newOrder
	}
}

// Validation methods

// ValidateViewName validates the view name against naming constraints.
//
// Requirements:
//   - Name must not be empty
//   - Length must be between 1 and 50 characters
//   - Only alphanumeric characters, underscores, and hyphens allowed
//
// Returns nil if valid, or a views.ValidationError describing the problem.
//
// This validation is automatically performed during state transitions in the
// interactive builder, but can also be called manually for programmatic use.
func (b *ViewBuilder) ValidateViewName() error {
	return views.ValidateViewName(b.ViewName)
}

// ValidateFieldSelection validates that at least one field is selected.
//
// This ensures the view will display at least some task information. A view
// with no fields would be empty and useless.
//
// Returns nil if one or more fields are selected, or a views.ValidationError
// if no fields are selected.
//
// This validation is automatically performed when transitioning from
// StateFieldSelection to StateFieldOrdering in the interactive builder.
func (b *ViewBuilder) ValidateFieldSelection() error {
	selectedCount := 0
	for _, field := range b.AvailableFields {
		if field.Selected {
			selectedCount++
		}
	}

	if selectedCount == 0 {
		return &views.ValidationError{
			Field:   "fields",
			Message: "at least one field must be selected",
		}
	}

	return nil
}

// ValidateFieldConfigs validates all selected field configurations against the field registry.
//
// For each selected field, this method validates:
//   - Field name exists in views.FieldRegistry
//   - Format (if specified) is valid for the field type
//   - Width is in valid range (0-200)
//
// Returns nil if all selected fields are valid, or the first validation error encountered.
//
// This validation is automatically performed when transitioning from
// StateFieldConfig to StateDisplayOptions in the interactive builder.
//
// Example:
//
//	builder.AvailableFields[0].Selected = true
//	builder.AvailableFields[0].Format = "invalid_format"
//	err := builder.ValidateFieldConfigs()
//	// err will be a ValidationError about invalid format
func (b *ViewBuilder) ValidateFieldConfigs() error {
	for _, field := range b.AvailableFields {
		if !field.Selected {
			continue
		}

		// Validate field name exists in registry
		_, ok := views.GetFieldDefinition(field.Name)
		if !ok {
			return &views.ValidationError{
				Field:   field.Name,
				Message: "unknown field",
			}
		}

		// Validate format if specified
		if field.Format != "" && !views.ValidateFieldFormat(field.Name, field.Format) {
			def, _ := views.GetFieldDefinition(field.Name)
			return &views.ValidationError{
				Field:   field.Name,
				Message: "invalid format '" + field.Format + "' (valid: " + joinFormats(def.Formats) + ")",
			}
		}

		// Validate width
		if field.Width < 0 || field.Width > 200 {
			return &views.ValidationError{
				Field:   field.Name,
				Message: "width must be between 0 and 200",
			}
		}
	}

	return nil
}

// Helper function to join format strings
func joinFormats(formats []string) string {
	result := ""
	for i, f := range formats {
		if i > 0 {
			result += ", "
		}
		result += f
	}
	return result
}
