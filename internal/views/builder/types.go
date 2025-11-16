// Package builder provides an interactive TUI for creating custom task views.
// It uses the Charm Bubbletea framework to guide users through a step-by-step
// wizard for configuring view options, selecting fields, and customizing display.
package builder

import (
	"fmt"
	"gosynctasks/internal/views"
)

// BuilderState represents the current state in the view builder state machine.
// The builder progresses through these states in sequence from Welcome to Done/Cancelled.
type BuilderState int

const (
	// StateWelcome is the initial welcome screen
	StateWelcome BuilderState = iota
	// StateBasicInfo collects view name and description
	StateBasicInfo
	// StateFieldSelection allows selecting which fields to display
	StateFieldSelection
	// StateFieldOrdering allows reordering selected fields
	StateFieldOrdering
	// StateFieldConfig configures format and color for each field
	StateFieldConfig
	// StateDisplayOptions configures view-level display options
	StateDisplayOptions
	// StateFilterConfig allows configuration of default filters
	StateFilterConfig
	// StateConfirm shows final configuration for confirmation
	StateConfirm
	// StateDone indicates successful completion
	StateDone
	// StateCancelled indicates user cancelled the builder
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
		case StateFilterConfig:
			return "Filter Configuration"
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

// FieldItem represents a field with its selection and configuration state.
// It tracks whether a field is selected for display and its formatting options.
type FieldItem struct {
	Name        string // Field name (e.g., "status", "summary")
	Description string // Human-readable description from field registry
	Selected    bool   // Whether this field is selected for display
	Format      string // Display format (e.g., "symbol", "text")
	Width       int    // Display width (for truncation)
	Color       bool   // Whether to use color
	Label       string // Custom label override
}

// ViewBuilder holds the state for building a custom view through the interactive wizard.
// It maintains all configuration choices across the different builder states and produces
// the final View when complete.
type ViewBuilder struct {
	// View being built
	ViewName        string // Name of the view being created
	ViewDescription string // Optional description

	// Field selection and ordering
	AvailableFields []FieldItem // All available fields from registry
	SelectedFields  []string    // Names of selected fields
	FieldOrder      []string    // Order of selected fields

	// Current field being configured
	CurrentFieldIndex int // Index for field configuration state

	// Display options
	ShowHeader  bool   // Whether to show column headers
	ShowBorder  bool   // Whether to show borders
	CompactMode bool   // Whether to use compact single-line display
	DateFormat  string // Date format string
	SortBy      string // Field to sort by
	SortOrder   string // Sort order ("asc" or "desc")

	// Filter options
	FilterStatus []string // Status filters (e.g., "NEEDS-ACTION", "COMPLETED")

	// State management
	CurrentState BuilderState // Current state in the wizard

	// Result
	View *views.View // Built view (populated on success)
	Err  error       // Error encountered during building
}

// NewViewBuilder creates a new view builder with the given view name.
// It initializes all available fields from the field registry with sensible defaults.
// Status and summary fields are pre-selected as they are the most commonly used.
func NewViewBuilder(name string) *ViewBuilder {
	// Initialize available fields from field registry
	// This ensures single source of truth and maintains consistency
	fieldOrder := []string{"status", "summary", "description", "priority",
		"due_date", "start_date", "created", "modified", "completed",
		"tags", "uid", "parent"}

	availableFields := make([]FieldItem, 0, len(fieldOrder))

	for _, fieldName := range fieldOrder {
		def, ok := views.GetFieldDefinition(fieldName)
		if !ok {
			continue // Skip fields not in registry
		}

		// Pre-select status and summary as they're most commonly used
		selected := (fieldName == "status" || fieldName == "summary")

		// Set sensible defaults
		item := FieldItem{
			Name:        fieldName,
			Description: def.Description,
			Selected:    selected,
			Format:      def.DefaultFormat,
			Width:       0,  // Will use default
			Color:       false, // User can toggle in config
			Label:       "",  // Will use default
		}

		// Description field gets special treatment for width
		if fieldName == "description" {
			item.Width = 70
		}

		availableFields = append(availableFields, item)
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
		FilterStatus:    []string{"NEEDS-ACTION", "IN-PROCESS"}, // Default: filter out completed tasks
	}
}

// BuildView constructs the final View from the builder state.
// It validates the configuration and returns an error if invalid.
// At minimum, at least one field must be selected.
func (b *ViewBuilder) BuildView() (*views.View, error) {
	// Validate: must have at least one selected field
	if len(b.FieldOrder) == 0 {
		return nil, fmt.Errorf("at least one field must be selected")
	}

	// Collect selected fields
	var fields []views.FieldConfig

	for _, fieldName := range b.FieldOrder {
		// Find the field item using helper
		item := b.getFieldItem(fieldName)
		if item == nil || !item.Selected {
			continue
		}

		// Validate format against field registry
		if !views.ValidateFieldFormat(item.Name, item.Format) {
			return nil, fmt.Errorf("invalid format %q for field %q", item.Format, item.Name)
		}

		showTrue := true
		field := views.FieldConfig{
			Name:   item.Name,
			Format: item.Format,
			Show:   &showTrue,
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

	// Final validation: ensure we have at least one field configured
	if len(fields) == 0 {
		return nil, fmt.Errorf("no valid fields configured")
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

	return view, nil
}

// UpdateSelectedFields updates the list of selected field names from AvailableFields.
// This should be called after field selection changes.
func (b *ViewBuilder) UpdateSelectedFields() {
	b.SelectedFields = []string{}
	for _, field := range b.AvailableFields {
		if field.Selected {
			b.SelectedFields = append(b.SelectedFields, field.Name)
		}
	}
}

// UpdateFieldOrder sets the field order from selected fields.
// It preserves existing order for fields that remain selected and appends newly selected fields.
// This is optimized to O(n) using a map for lookups.
func (b *ViewBuilder) UpdateFieldOrder() {
	if len(b.FieldOrder) == 0 {
		// Initialize field order with currently selected fields
		b.FieldOrder = append([]string{}, b.SelectedFields...)
		return
	}

	// Build a map of selected fields for O(1) lookup
	selectedMap := make(map[string]bool, len(b.SelectedFields))
	for _, name := range b.SelectedFields {
		selectedMap[name] = true
	}

	// Keep fields that are still selected, preserving order
	newOrder := make([]string, 0, len(b.SelectedFields))
	for _, fieldName := range b.FieldOrder {
		if selectedMap[fieldName] {
			newOrder = append(newOrder, fieldName)
			delete(selectedMap, fieldName) // Mark as processed
		}
	}

	// Append newly selected fields that weren't in the old order
	for _, selected := range b.SelectedFields {
		if selectedMap[selected] { // If still in map, it's new
			newOrder = append(newOrder, selected)
		}
	}

	b.FieldOrder = newOrder
}

// Validate checks if the current builder state is valid.
// Returns an error describing what is invalid, or nil if valid.
func (b *ViewBuilder) Validate() error {
	// View name required
	if b.ViewName == "" {
		return fmt.Errorf("view name is required")
	}

	// At least one field must be selected
	if len(b.SelectedFields) == 0 {
		return fmt.Errorf("at least one field must be selected")
	}

	// Validate each selected field's format
	for _, fieldName := range b.SelectedFields {
		item := b.getFieldItem(fieldName)
		if item == nil {
			continue
		}

		if !views.ValidateFieldFormat(item.Name, item.Format) {
			return fmt.Errorf("invalid format %q for field %q", item.Format, item.Name)
		}
	}

	return nil
}

// getFieldItem is a helper that finds a FieldItem by name.
// Returns nil if not found. This reduces code duplication.
func (b *ViewBuilder) getFieldItem(name string) *FieldItem {
	for i := range b.AvailableFields {
		if b.AvailableFields[i].Name == name {
			return &b.AvailableFields[i]
		}
	}
	return nil
}
