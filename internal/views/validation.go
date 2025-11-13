package views

import (
	"fmt"
	"strings"
)

// ValidationError represents a validation error with details
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// ValidateView validates a view configuration
func ValidateView(view *View) error {
	if view == nil {
		return &ValidationError{Message: "view cannot be nil"}
	}

	// Validate view name
	if err := ValidateViewName(view.Name); err != nil {
		return err
	}

	// Validate fields
	if err := ValidateFields(view.Fields); err != nil {
		return err
	}

	// Validate field order
	if err := ValidateFieldOrder(view.Fields, view.FieldOrder); err != nil {
		return err
	}

	// Validate display options
	if err := ValidateDisplayOptions(&view.Display); err != nil {
		return err
	}

	return nil
}

// ValidateViewName validates a view name
func ValidateViewName(name string) error {
	if name == "" {
		return &ValidationError{Field: "name", Message: "view name is required"}
	}

	if len(name) < 1 || len(name) > 50 {
		return &ValidationError{Field: "name", Message: "view name must be between 1 and 50 characters"}
	}

	// Check alphanum_underscore constraint
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return &ValidationError{
				Field:   "name",
				Message: "view name can only contain letters, numbers, underscores, and hyphens",
			}
		}
	}

	return nil
}

// ValidateFields validates field configurations
func ValidateFields(fields []FieldConfig) error {
	if len(fields) == 0 {
		return &ValidationError{Field: "fields", Message: "at least one field must be selected"}
	}

	for i, field := range fields {
		if err := ValidateField(&field); err != nil {
			return &ValidationError{
				Field:   fmt.Sprintf("fields[%d]", i),
				Message: err.Error(),
			}
		}
	}

	return nil
}

// ValidateField validates a single field configuration
func ValidateField(field *FieldConfig) error {
	if field == nil {
		return &ValidationError{Message: "field cannot be nil"}
	}

	// Validate field name exists in registry
	def, ok := GetFieldDefinition(field.Name)
	if !ok {
		return &ValidationError{
			Field:   "name",
			Message: fmt.Sprintf("unknown field '%s'", field.Name),
		}
	}

	// Validate format if specified
	if field.Format != "" && !ValidateFieldFormat(field.Name, field.Format) {
		return &ValidationError{
			Field: "format",
			Message: fmt.Sprintf("invalid format '%s' for field '%s' (valid: %s)",
				field.Format, field.Name, strings.Join(def.Formats, ", ")),
		}
	}

	// Validate width
	if field.Width < 0 || field.Width > 200 {
		return &ValidationError{
			Field:   "width",
			Message: "field width must be between 0 and 200",
		}
	}

	return nil
}

// ValidateFieldOrder validates field order against field list
func ValidateFieldOrder(fields []FieldConfig, fieldOrder []string) error {
	if len(fieldOrder) == 0 {
		// Empty field order is valid - will use fields array order
		return nil
	}

	// Create a map of field names for quick lookup
	fieldMap := make(map[string]bool)
	for _, field := range fields {
		fieldMap[field.Name] = true
	}

	// Check that all field order entries exist in fields
	for i, fieldName := range fieldOrder {
		if !fieldMap[fieldName] {
			return &ValidationError{
				Field: fmt.Sprintf("field_order[%d]", i),
				Message: fmt.Sprintf("field '%s' in field_order does not exist in fields list",
					fieldName),
			}
		}
	}

	return nil
}

// ValidateDisplayOptions validates display options
func ValidateDisplayOptions(opts *DisplayOptions) error {
	if opts == nil {
		return nil // Display options are optional
	}

	// Validate sort_by if specified
	if opts.SortBy != "" {
		validSortFields := []string{"status", "summary", "priority", "due_date", "start_date", "created", "modified"}
		valid := false
		for _, validField := range validSortFields {
			if opts.SortBy == validField {
				valid = true
				break
			}
		}
		if !valid {
			return &ValidationError{
				Field: "display.sort_by",
				Message: fmt.Sprintf("invalid sort_by field '%s' (valid: %s)",
					opts.SortBy, strings.Join(validSortFields, ", ")),
			}
		}
	}

	// Validate sort_order if specified
	if opts.SortOrder != "" && opts.SortOrder != "asc" && opts.SortOrder != "desc" {
		return &ValidationError{
			Field:   "display.sort_order",
			Message: "sort_order must be 'asc' or 'desc'",
		}
	}

	return nil
}
