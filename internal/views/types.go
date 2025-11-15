package views

import "time"

// View represents a custom view configuration for displaying tasks.
// Views are stored as YAML files in ~/.config/gosynctasks/views/
type View struct {
	// Name is the unique identifier for this view (filename without .yaml extension)
	Name string `yaml:"name" validate:"required,min=1,max=50,alphanum_underscore"`

	// Description provides a human-readable explanation of the view's purpose
	Description string `yaml:"description,omitempty"`

	// Fields defines which task fields to display and how to format them
	Fields []FieldConfig `yaml:"fields" validate:"required,min=1,dive"`

	// FieldOrder specifies the display order of fields (field names)
	// If empty, fields are displayed in the order they appear in Fields
	FieldOrder []string `yaml:"field_order,omitempty"`

	// Filters defines default filtering rules for this view
	Filters *ViewFilters `yaml:"filters,omitempty"`

	// Display contains overall presentation options
	Display DisplayOptions `yaml:"display,omitempty"`
}

// FieldConfig specifies how to display a single task field
type FieldConfig struct {
	// Name is the field identifier (e.g., "status", "summary", "priority")
	Name string `yaml:"name" validate:"required,oneof=status summary description priority due_date start_date created modified completed tags uid parent"`

	// Format specifies the display format for this field
	// Available formats depend on the field type (see FieldDefinition)
	Format string `yaml:"format,omitempty"`

	// Label is the display label for this field (overrides default)
	Label string `yaml:"label,omitempty"`

	// Width specifies the maximum width for this field (0 = no limit)
	Width int `yaml:"width,omitempty" validate:"min=0,max=200"`

	// Color enables/disables color coding for this field
	Color bool `yaml:"color,omitempty"`

	// Show controls whether this field is displayed
	// nil = default to true, true = show, false = hide
	Show *bool `yaml:"show,omitempty"`
}

// ViewFilters defines default filtering rules for a view
type ViewFilters struct {
	// Status filters tasks by status (e.g., ["TODO", "PROCESSING"])
	Status []string `yaml:"status,omitempty"`

	// Priority filters tasks by priority range
	PriorityMin int `yaml:"priority_min,omitempty" validate:"min=0,max=9"`
	PriorityMax int `yaml:"priority_max,omitempty" validate:"min=0,max=9"`

	// Tags filters tasks that have all specified tags
	Tags []string `yaml:"tags,omitempty"`

	// DueBefore filters tasks due before this date
	DueBefore *time.Time `yaml:"due_before,omitempty"`

	// DueAfter filters tasks due after this date
	DueAfter *time.Time `yaml:"due_after,omitempty"`

	// StartBefore filters tasks starting before this date
	StartBefore *time.Time `yaml:"start_before,omitempty"`

	// StartAfter filters tasks starting after this date
	StartAfter *time.Time `yaml:"start_after,omitempty"`
}

// DisplayOptions controls overall presentation behavior
type DisplayOptions struct {
	// ShowHeader enables/disables the list header
	ShowHeader bool `yaml:"show_header"`

	// ShowBorder enables/disables borders around task list
	ShowBorder bool `yaml:"show_border"`

	// CompactMode reduces spacing between tasks
	CompactMode bool `yaml:"compact_mode"`

	// DateFormat specifies the Go time format string for dates
	DateFormat string `yaml:"date_format,omitempty"`

	// Sorting specifies the field to sort by
	SortBy string `yaml:"sort_by,omitempty" validate:"omitempty,oneof=status summary priority due_date start_date created modified"`

	// SortOrder specifies ascending or descending order
	SortOrder string `yaml:"sort_order,omitempty" validate:"omitempty,oneof=asc desc"`
}

// FieldDefinition describes a task field's available formats
type FieldDefinition struct {
	// Name is the field identifier
	Name string

	// Description explains what this field represents
	Description string

	// Formats lists available display formats for this field
	Formats []string

	// DefaultFormat is the format used when none is specified
	DefaultFormat string

	// RequiresBackend indicates if this field needs backend-specific rendering
	RequiresBackend bool
}
