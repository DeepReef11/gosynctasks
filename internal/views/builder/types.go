package builder

import (
	"gosynctasks/internal/views"
)

// BuilderState represents the current state in the view builder state machine
type BuilderState int

const (
	StateWelcome BuilderState = iota
	StateBasicInfo
	StateFieldSelection
	StateFieldOrdering
	StateFieldConfig
	StateDisplayOptions
	StateConfirm
	StateDone
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

// FieldItem represents a field with its selection state
type FieldItem struct {
	Name        string
	Description string
	Selected    bool
	Format      string
	Width       int
	Color       bool
	Label       string
}

// ViewBuilder holds the state for building a view
type ViewBuilder struct {
	// View being built
	ViewName        string
	ViewDescription string

	// Field selection and ordering
	AvailableFields []FieldItem
	SelectedFields  []string
	FieldOrder      []string

	// Current field being configured
	CurrentFieldIndex int

	// Display options
	ShowHeader  bool
	ShowBorder  bool
	CompactMode bool
	DateFormat  string
	SortBy      string
	SortOrder   string

	// State management
	CurrentState BuilderState

	// Result
	View *views.View
	Err  error
}

// NewViewBuilder creates a new view builder
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

// BuildView constructs the final View from the builder state
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

		field := views.FieldConfig{
			Name:   item.Name,
			Format: item.Format,
			Show:   true,
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

// UpdateSelectedFields updates the list of selected field names
func (b *ViewBuilder) UpdateSelectedFields() {
	b.SelectedFields = []string{}
	for _, field := range b.AvailableFields {
		if field.Selected {
			b.SelectedFields = append(b.SelectedFields, field.Name)
		}
	}
}

// UpdateFieldOrder sets the field order from selected fields
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

// ValidateViewName validates the view name
func (b *ViewBuilder) ValidateViewName() error {
	return views.ValidateViewName(b.ViewName)
}

// ValidateFieldSelection validates that at least one field is selected
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

// ValidateFieldConfigs validates field configurations against the field registry
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
