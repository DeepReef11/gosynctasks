package views

import "testing"

func TestGetFieldDefinition(t *testing.T) {
	tests := []struct {
		name          string
		fieldName     string
		expectFound   bool
		expectFormats int // Expected number of formats
	}{
		{"status field", "status", true, 4},
		{"summary field", "summary", true, 2},
		{"priority field", "priority", true, 4},
		{"due_date field", "due_date", true, 3},
		{"invalid field", "nonexistent", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, found := GetFieldDefinition(tt.fieldName)
			if found != tt.expectFound {
				t.Errorf("GetFieldDefinition(%q) found = %v, want %v", tt.fieldName, found, tt.expectFound)
			}

			if found {
				if def.Name != tt.fieldName {
					t.Errorf("Expected field name %q, got %q", tt.fieldName, def.Name)
				}

				if len(def.Formats) != tt.expectFormats {
					t.Errorf("Expected %d formats for %q, got %d", tt.expectFormats, tt.fieldName, len(def.Formats))
				}

				if def.DefaultFormat == "" {
					t.Errorf("Expected default format to be set for %q", tt.fieldName)
				}

				if def.Description == "" {
					t.Errorf("Expected description to be set for %q", tt.fieldName)
				}
			}
		})
	}
}

func TestValidateFieldFormat(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		format    string
		valid     bool
	}{
		{"status symbol", "status", "symbol", true},
		{"status text", "status", "text", true},
		{"status emoji", "status", "emoji", true},
		{"status invalid", "status", "invalid_format", false},
		{"status empty (uses default)", "status", "", true},
		{"summary full", "summary", "full", true},
		{"summary truncate", "summary", "truncate", true},
		{"summary invalid", "summary", "invalid", false},
		{"invalid field", "nonexistent", "any", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateFieldFormat(tt.fieldName, tt.format)
			if result != tt.valid {
				t.Errorf("ValidateFieldFormat(%q, %q) = %v, want %v",
					tt.fieldName, tt.format, result, tt.valid)
			}
		})
	}
}

func TestGetDefaultFormat(t *testing.T) {
	tests := []struct {
		fieldName      string
		expectedFormat string
	}{
		{"status", "symbol"},
		{"summary", "full"},
		{"description", "truncate"},
		{"priority", "number"},
		{"due_date", "full"},
		{"start_date", "full"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			format := GetDefaultFormat(tt.fieldName)
			if format != tt.expectedFormat {
				t.Errorf("GetDefaultFormat(%q) = %q, want %q",
					tt.fieldName, format, tt.expectedFormat)
			}
		})
	}
}

func TestGetDefaultFormat_Invalid(t *testing.T) {
	format := GetDefaultFormat("nonexistent_field")
	if format != "" {
		t.Errorf("Expected empty string for invalid field, got %q", format)
	}
}

func TestFieldRegistryCompleteness(t *testing.T) {
	// Ensure all fields in the registry have required properties
	for name, def := range FieldRegistry {
		if def.Name != name {
			t.Errorf("Field %q has mismatched name in definition: %q", name, def.Name)
		}

		if def.Description == "" {
			t.Errorf("Field %q missing description", name)
		}

		if len(def.Formats) == 0 {
			t.Errorf("Field %q has no formats defined", name)
		}

		if def.DefaultFormat == "" {
			t.Errorf("Field %q has no default format", name)
		}

		// Verify default format is in the list of valid formats
		validDefault := false
		for _, format := range def.Formats {
			if format == def.DefaultFormat {
				validDefault = true
				break
			}
		}

		if !validDefault {
			t.Errorf("Field %q default format %q not in valid formats list: %v",
				name, def.DefaultFormat, def.Formats)
		}
	}
}

func TestRequiredFields(t *testing.T) {
	// Ensure key fields exist in the registry
	requiredFields := []string{
		"status",
		"summary",
		"description",
		"priority",
		"due_date",
		"start_date",
		"created",
		"modified",
		"completed",
		"tags",
		"uid",
		"parent",
	}

	for _, fieldName := range requiredFields {
		if _, found := GetFieldDefinition(fieldName); !found {
			t.Errorf("Required field %q not found in registry", fieldName)
		}
	}
}
