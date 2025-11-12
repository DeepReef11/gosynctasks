package views

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	// Register custom validation for alphanum_underscore
	validate.RegisterValidation("alphanum_underscore", func(fl validator.FieldLevel) bool {
		str := fl.Field().String()
		for _, r := range str {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
				return false
			}
		}
		return true
	})
}

// LoadView loads a view configuration from a YAML file
func LoadView(path string) (*View, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read view file %s: %w", path, err)
	}

	// Parse YAML
	var view View
	if err := yaml.Unmarshal(data, &view); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", path, err)
	}

	// Set name from filename if not specified in YAML
	if view.Name == "" {
		view.Name = filepath.Base(path)
		// Remove .yaml or .yml extension
		if ext := filepath.Ext(view.Name); ext == ".yaml" || ext == ".yml" {
			view.Name = view.Name[:len(view.Name)-len(ext)]
		}
	}

	// Validate structure
	if err := validate.Struct(&view); err != nil {
		return nil, fmt.Errorf("validation failed for view %s: %w", path, formatValidationError(err))
	}

	// Validate field formats
	for i, field := range view.Fields {
		if !ValidateFieldFormat(field.Name, field.Format) {
			def, _ := GetFieldDefinition(field.Name)
			return nil, fmt.Errorf("invalid format '%s' for field '%s' (valid formats: %v)",
				field.Format, field.Name, def.Formats)
		}

		// Set default format if not specified
		if field.Format == "" {
			view.Fields[i].Format = GetDefaultFormat(field.Name)
		}

		// Set default Show to true if not explicitly set
		if !field.Show {
			view.Fields[i].Show = true
		}
	}

	// Validate field_order references existing fields
	if len(view.FieldOrder) > 0 {
		fieldMap := make(map[string]bool)
		for _, field := range view.Fields {
			fieldMap[field.Name] = true
		}

		for _, fieldName := range view.FieldOrder {
			if !fieldMap[fieldName] {
				return nil, fmt.Errorf("field_order references undefined field: %s", fieldName)
			}
		}
	}

	// Set default display options
	if view.Display.DateFormat == "" {
		view.Display.DateFormat = "2006-01-02"
	}

	return &view, nil
}

// LoadViewFromBytes loads a view configuration from YAML bytes (used for testing)
func LoadViewFromBytes(data []byte, name string) (*View, error) {
	var view View
	if err := yaml.Unmarshal(data, &view); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if view.Name == "" {
		view.Name = name
	}

	// Validate and set defaults (same as LoadView)
	if err := validate.Struct(&view); err != nil {
		return nil, fmt.Errorf("validation failed: %w", formatValidationError(err))
	}

	for i, field := range view.Fields {
		if !ValidateFieldFormat(field.Name, field.Format) {
			def, _ := GetFieldDefinition(field.Name)
			return nil, fmt.Errorf("invalid format '%s' for field '%s' (valid formats: %v)",
				field.Format, field.Name, def.Formats)
		}

		if field.Format == "" {
			view.Fields[i].Format = GetDefaultFormat(field.Name)
		}

		if !field.Show {
			view.Fields[i].Show = true
		}
	}

	// Validate field_order references existing fields
	if len(view.FieldOrder) > 0 {
		fieldMap := make(map[string]bool)
		for _, field := range view.Fields {
			fieldMap[field.Name] = true
		}

		for _, fieldName := range view.FieldOrder {
			if !fieldMap[fieldName] {
				return nil, fmt.Errorf("field_order references undefined field: %s", fieldName)
			}
		}
	}

	if view.Display.DateFormat == "" {
		view.Display.DateFormat = "2006-01-02"
	}

	return &view, nil
}

// formatValidationError converts validator errors to user-friendly messages
func formatValidationError(err error) error {
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrs {
			switch e.Tag() {
			case "required":
				return fmt.Errorf("field '%s' is required", e.Field())
			case "min":
				return fmt.Errorf("field '%s' must have at least %s items/characters", e.Field(), e.Param())
			case "max":
				return fmt.Errorf("field '%s' must have at most %s items/characters", e.Field(), e.Param())
			case "oneof":
				return fmt.Errorf("field '%s' must be one of: %s", e.Field(), e.Param())
			case "alphanum_underscore":
				return fmt.Errorf("field '%s' must contain only letters, numbers, underscores, and hyphens", e.Field())
			}
		}
	}
	return err
}
