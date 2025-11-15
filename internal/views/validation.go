package views

import (
	"fmt"
	"strings"
)

// ValidationError represents a validation error with details
type ValidationError struct {
	Field   string
	Message string
	Value   string // The invalid value that caused the error
	Hint    string // Suggestion for fixing the error
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// ValidationErrors is a collection of multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError
}

func (e ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("validation failed with %d errors:\n", len(e.Errors)))
	for i, err := range e.Errors {
		b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return b.String()
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

// ValidateViewComprehensive validates a view and collects ALL errors instead of stopping at first
func ValidateViewComprehensive(view *View) *ValidationErrors {
	var errors []ValidationError

	if view == nil {
		return &ValidationErrors{
			Errors: []ValidationError{{Message: "view cannot be nil"}},
		}
	}

	// Validate view name
	if err := ValidateViewName(view.Name); err != nil {
		if ve, ok := err.(*ValidationError); ok {
			errors = append(errors, *ve)
		} else {
			errors = append(errors, ValidationError{Field: "name", Message: err.Error()})
		}
	}

	// Validate fields - collect all field errors
	if len(view.Fields) == 0 {
		errors = append(errors, ValidationError{
			Field:   "fields",
			Message: "at least one field must be selected",
			Hint:    "Add at least one field from: status, summary, description, priority, due_date, start_date, created, modified, completed, tags, uid, parent",
		})
	} else {
		for i, field := range view.Fields {
			if fieldErr := ValidateField(&field); fieldErr != nil {
				if ve, ok := fieldErr.(*ValidationError); ok {
					// Prefix with array index
					ve.Field = fmt.Sprintf("fields[%d].%s", i, ve.Field)
					errors = append(errors, *ve)
				}
			}
		}
	}

	// Validate field order
	if len(view.FieldOrder) > 0 {
		fieldMap := make(map[string]bool)
		for _, field := range view.Fields {
			fieldMap[field.Name] = true
		}

		for i, fieldName := range view.FieldOrder {
			if !fieldMap[fieldName] {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("field_order[%d]", i),
					Message: fmt.Sprintf("field '%s' does not exist in fields list", fieldName),
					Value:   fieldName,
					Hint:    "field_order can only reference fields defined in the 'fields' array",
				})
			}
		}
	}

	// Validate display options
	if err := ValidateDisplayOptions(&view.Display); err != nil {
		if ve, ok := err.(*ValidationError); ok {
			errors = append(errors, *ve)
		}
	}

	if len(errors) > 0 {
		return &ValidationErrors{Errors: errors}
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
		validFields := []string{}
		for name := range FieldRegistry {
			validFields = append(validFields, name)
		}
		return &ValidationError{
			Field:   "name",
			Message: fmt.Sprintf("unknown field '%s'", field.Name),
			Value:   field.Name,
			Hint:    fmt.Sprintf("Valid fields: %s", strings.Join(validFields, ", ")),
		}
	}

	// Validate format if specified
	if field.Format != "" && !ValidateFieldFormat(field.Name, field.Format) {
		return &ValidationError{
			Field:   "format",
			Message: fmt.Sprintf("invalid format '%s' for field '%s'", field.Format, field.Name),
			Value:   field.Format,
			Hint:    fmt.Sprintf("Valid formats for '%s': %s", field.Name, strings.Join(def.Formats, ", ")),
		}
	}

	// Validate width
	if field.Width < 0 || field.Width > 200 {
		return &ValidationError{
			Field:   "width",
			Message: "field width must be between 0 and 200",
			Value:   fmt.Sprintf("%d", field.Width),
			Hint:    "Set width to a value between 0-200, or 0 for no limit",
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
				Field:   "display.sort_by",
				Message: fmt.Sprintf("invalid sort_by field '%s'", opts.SortBy),
				Value:   opts.SortBy,
				Hint:    fmt.Sprintf("Valid sort fields: %s", strings.Join(validSortFields, ", ")),
			}
		}
	}

	// Validate sort_order if specified
	if opts.SortOrder != "" && opts.SortOrder != "asc" && opts.SortOrder != "desc" {
		return &ValidationError{
			Field:   "display.sort_order",
			Message: "sort_order must be 'asc' or 'desc'",
			Value:   opts.SortOrder,
			Hint:    "Use 'asc' for ascending or 'desc' for descending",
		}
	}

	return nil
}

// AnnotateYAMLWithErrors adds inline error comments to YAML content
func AnnotateYAMLWithErrors(yamlContent string, errors *ValidationErrors) string {
	if errors == nil || len(errors.Errors) == 0 {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")

	// Build error header
	var header strings.Builder
	header.WriteString("# ========================================\n")
	header.WriteString("# VALIDATION ERRORS - Please fix the issues below\n")
	header.WriteString(fmt.Sprintf("# Found %d error(s)\n", len(errors.Errors)))
	header.WriteString("# ========================================\n")
	for i, err := range errors.Errors {
		header.WriteString(fmt.Sprintf("# %d. %s: %s\n", i+1, err.Field, err.Message))
		if err.Hint != "" {
			header.WriteString(fmt.Sprintf("#    Hint: %s\n", err.Hint))
		}
	}
	header.WriteString("# ========================================\n\n")

	// Insert errors inline at relevant locations
	result := make([]string, 0, len(lines)+len(errors.Errors)*3)
	result = append(result, strings.Split(header.String(), "\n")...)

	for i, line := range lines {
		result = append(result, line)

		// Check if this line matches any error field
		for _, err := range errors.Errors {
			if matchesErrorField(line, err.Field) {
				// Add inline error comment
				result = append(result, fmt.Sprintf("# ERROR: %s", err.Message))
				if err.Value != "" {
					result = append(result, fmt.Sprintf("# Invalid value: %s", err.Value))
				}
				if err.Hint != "" {
					result = append(result, fmt.Sprintf("# %s", err.Hint))
				}
			}
		}

		// Special handling for array items (fields[0], fields[1], etc.)
		for _, err := range errors.Errors {
			if strings.Contains(err.Field, "fields[") && strings.Contains(line, "- name:") {
				// Extract array index from error field
				var arrayIndex int
				if _, scanErr := fmt.Sscanf(err.Field, "fields[%d]", &arrayIndex); scanErr == nil {
					// Count how many "- name:" we've seen so far
					nameCount := 0
					for j := 0; j <= i; j++ {
						if strings.Contains(lines[j], "- name:") {
							if nameCount == arrayIndex {
								// This is the field with the error
								result = append(result, fmt.Sprintf("  # ERROR in this field: %s", err.Message))
								if err.Hint != "" {
									result = append(result, fmt.Sprintf("  # %s", err.Hint))
								}
								break
							}
							nameCount++
						}
					}
				}
			}
		}
	}

	return strings.Join(result, "\n")
}

// matchesErrorField checks if a YAML line matches the error field path
func matchesErrorField(line, fieldPath string) bool {
	line = strings.TrimSpace(line)

	// Handle nested fields like "display.sort_by"
	parts := strings.Split(fieldPath, ".")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]

		// Check if line starts with the field name
		if strings.HasPrefix(line, lastPart+":") {
			return true
		}
		if strings.HasPrefix(line, "- "+lastPart+":") {
			return true
		}
	}

	// Direct field match
	if strings.HasPrefix(line, fieldPath+":") {
		return true
	}

	return false
}
