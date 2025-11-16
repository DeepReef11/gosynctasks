package views

import (
	"strings"
	"testing"
)

func TestValidateViewComprehensive(t *testing.T) {
	tests := []struct {
		name          string
		view          *View
		expectErrors  bool
		errorCount    int
		errorContains []string
	}{
		{
			name:         "nil view",
			view:         nil,
			expectErrors: true,
			errorCount:   1,
		},
		{
			name: "valid view",
			view: &View{
				Name:        "test",
				Description: "Test view",
				Fields: []FieldConfig{
					{Name: "status", Format: "symbol"},
					{Name: "summary", Format: "full"},
				},
				Display: DisplayOptions{
					SortBy:    "priority",
					SortOrder: "asc",
				},
			},
			expectErrors: false,
		},
		{
			name: "multiple errors",
			view: &View{
				Name:        "test-view!@#", // Invalid characters
				Description: "Test",
				Fields: []FieldConfig{
					{Name: "invalid_field", Format: "full"}, // Unknown field
					{Name: "status", Format: "invalid"},     // Invalid format
					{Name: "summary", Width: 300},           // Width too large
				},
				Display: DisplayOptions{
					SortBy:    "invalid_field", // Invalid sort field
					SortOrder: "middle",        // Invalid sort order
				},
			},
			expectErrors: true,
			errorCount:   5, // name, field name, format, width, sort_by, sort_order
			errorContains: []string{
				"name",
				"invalid_field",
				"invalid",
				"width",
				"sort_by",
			},
		},
		{
			name: "invalid field format",
			view: &View{
				Name: "test",
				Fields: []FieldConfig{
					{Name: "status", Format: "nonexistent"},
				},
			},
			expectErrors:  true,
			errorCount:    1,
			errorContains: []string{"format", "status"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateViewComprehensive(tt.view)

			if tt.expectErrors {
				if errors == nil {
					t.Errorf("Expected errors but got none")
					return
				}
				if len(errors.Errors) != tt.errorCount {
					t.Errorf("Expected %d errors, got %d: %v", tt.errorCount, len(errors.Errors), errors)
				}
				// Check error messages contain expected strings
				errMsg := errors.Error()
				for _, expectedStr := range tt.errorContains {
					if !strings.Contains(errMsg, expectedStr) {
						t.Errorf("Expected error to contain '%s', got: %s", expectedStr, errMsg)
					}
				}
			} else {
				if errors != nil {
					t.Errorf("Expected no errors but got: %v", errors)
				}
			}
		})
	}
}

func TestAnnotateYAMLWithErrors(t *testing.T) {
	yamlContent := `name: test
description: Test view
fields:
  - name: status
    format: invalid
  - name: summary
    format: full
display:
  sort_by: invalid_field
  sort_order: asc
`

	errors := &ValidationErrors{
		Errors: []ValidationError{
			{
				Field:   "fields[0].format",
				Message: "invalid format 'invalid' for field 'status'",
				Value:   "invalid",
				Hint:    "Valid formats for 'status': symbol, text, emoji, short",
			},
			{
				Field:   "display.sort_by",
				Message: "invalid sort_by field 'invalid_field'",
				Value:   "invalid_field",
				Hint:    "Valid sort fields: status, summary, priority, due_date, start_date, created, modified",
			},
		},
	}

	annotated := AnnotateYAMLWithErrors(yamlContent, errors)

	// Check that error header is added
	if !strings.Contains(annotated, "VALIDATION ERRORS") {
		t.Error("Expected error header in annotated YAML")
	}

	// Check that error count is shown
	if !strings.Contains(annotated, "Found 2 error(s)") {
		t.Error("Expected error count in annotated YAML")
	}

	// Check that original content is preserved
	if !strings.Contains(annotated, "name: test") {
		t.Error("Expected original content to be preserved")
	}

	// Check that hints are included
	if !strings.Contains(annotated, "Valid formats for 'status'") {
		t.Error("Expected hint in annotated YAML")
	}
}

func TestValidationErrorsError(t *testing.T) {
	tests := []struct {
		name     string
		errors   ValidationErrors
		expected string
	}{
		{
			name:     "no errors",
			errors:   ValidationErrors{Errors: []ValidationError{}},
			expected: "validation failed",
		},
		{
			name: "single error",
			errors: ValidationErrors{
				Errors: []ValidationError{
					{Field: "name", Message: "required"},
				},
			},
			expected: "name: required",
		},
		{
			name: "multiple errors",
			errors: ValidationErrors{
				Errors: []ValidationError{
					{Field: "name", Message: "required"},
					{Field: "fields", Message: "at least one field required"},
				},
			},
			expected: "validation failed with 2 errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.errors.Error()
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected error message to contain '%s', got: %s", tt.expected, result)
			}
		})
	}
}
