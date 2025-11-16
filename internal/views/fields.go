package views

// FieldRegistry maps field names to their definitions
var FieldRegistry = map[string]FieldDefinition{
	"status": {
		Name:          "status",
		Description:   "Task completion status",
		Formats:       []string{"symbol", "text", "emoji", "short"},
		DefaultFormat: "symbol",
	},
	"summary": {
		Name:          "summary",
		Description:   "Task title/summary",
		Formats:       []string{"full", "truncate"},
		DefaultFormat: "full",
	},
	"description": {
		Name:          "description",
		Description:   "Task detailed description",
		Formats:       []string{"full", "truncate", "first_line"},
		DefaultFormat: "truncate",
	},
	"priority": {
		Name:            "priority",
		Description:     "Task priority (0-9)",
		Formats:         []string{"number", "text", "stars", "color"},
		DefaultFormat:   "number",
		RequiresBackend: true, // For priority color
	},
	"due_date": {
		Name:          "due_date",
		Description:   "Task due date",
		Formats:       []string{"full", "relative", "short"},
		DefaultFormat: "full",
	},
	"start_date": {
		Name:          "start_date",
		Description:   "Task start date",
		Formats:       []string{"full", "relative", "short"},
		DefaultFormat: "full",
	},
	"created": {
		Name:          "created",
		Description:   "Task creation timestamp",
		Formats:       []string{"full", "relative", "date_only"},
		DefaultFormat: "full",
	},
	"modified": {
		Name:          "modified",
		Description:   "Task last modified timestamp",
		Formats:       []string{"full", "relative", "date_only"},
		DefaultFormat: "full",
	},
	"completed": {
		Name:          "completed",
		Description:   "Task completion timestamp",
		Formats:       []string{"full", "relative", "date_only", "short"},
		DefaultFormat: "full",
	},
	"tags": {
		Name:          "tags",
		Description:   "Task categories/labels",
		Formats:       []string{"list", "comma", "hash"},
		DefaultFormat: "comma",
	},
	"uid": {
		Name:          "uid",
		Description:   "Unique task identifier",
		Formats:       []string{"full", "short"},
		DefaultFormat: "short",
	},
	"parent": {
		Name:          "parent",
		Description:   "Parent task UID (for subtasks)",
		Formats:       []string{"full", "short"},
		DefaultFormat: "short",
	},
}

// GetFieldDefinition returns the definition for a field name
func GetFieldDefinition(name string) (FieldDefinition, bool) {
	def, ok := FieldRegistry[name]
	return def, ok
}

// ValidateFieldFormat checks if a format is valid for a field
func ValidateFieldFormat(fieldName, format string) bool {
	def, ok := GetFieldDefinition(fieldName)
	if !ok {
		return false
	}

	// Empty format is valid (will use default)
	if format == "" {
		return true
	}

	// Check if format is in the list of valid formats
	for _, validFormat := range def.Formats {
		if format == validFormat {
			return true
		}
	}

	return false
}

// GetDefaultFormat returns the default format for a field
func GetDefaultFormat(fieldName string) string {
	def, ok := GetFieldDefinition(fieldName)
	if !ok {
		return ""
	}
	return def.DefaultFormat
}
