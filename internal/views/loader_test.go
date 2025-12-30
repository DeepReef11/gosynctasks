package views

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadView(t *testing.T) {
	// Test loading minimal.yaml
	viewPath := filepath.Join("testdata", "views", "minimal.yaml")
	view, err := LoadView(viewPath)
	if err != nil {
		t.Fatalf("Failed to load minimal.yaml: %v", err)
	}

	if view.Name != "minimal" {
		t.Errorf("Expected name 'minimal', got '%s'", view.Name)
	}

	if len(view.Fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(view.Fields))
	}

	// Check that formats were set to defaults or specified values
	for _, field := range view.Fields {
		if field.Format == "" {
			t.Errorf("Field %s has empty format (should be set to default)", field.Name)
		}
	}
}

func TestLoadView_Detailed(t *testing.T) {
	viewPath := filepath.Join("testdata", "views", "detailed.yaml")
	view, err := LoadView(viewPath)
	if err != nil {
		t.Fatalf("Failed to load detailed.yaml: %v", err)
	}

	if view.Name != "detailed" {
		t.Errorf("Expected name 'detailed', got '%s'", view.Name)
	}

	// Check field_order
	if len(view.FieldOrder) == 0 {
		t.Error("Expected field_order to be populated")
	}

	// Verify field_order references existing fields
	fieldMap := make(map[string]bool)
	for _, field := range view.Fields {
		fieldMap[field.Name] = true
	}

	for _, fieldName := range view.FieldOrder {
		if !fieldMap[fieldName] {
			t.Errorf("field_order references non-existent field: %s", fieldName)
		}
	}
}

func TestLoadViewFromBytes_Valid(t *testing.T) {
	yaml := `
name: test_view
description: Test view for unit testing
fields:
  - name: status
    format: symbol
    show: true
  - name: summary
    format: full
    show: true
display:
  show_header: true
  show_border: true
`

	view, err := LoadViewFromBytes([]byte(yaml), "test_view")
	if err != nil {
		t.Fatalf("Failed to load view from bytes: %v", err)
	}

	if view.Name != "test_view" {
		t.Errorf("Expected name 'test_view', got '%s'", view.Name)
	}

	if len(view.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(view.Fields))
	}
}

func TestLoadViewFromBytes_InvalidField(t *testing.T) {
	yaml := `
name: invalid
fields:
  - name: invalid_field_name
    format: symbol
`

	_, err := LoadViewFromBytes([]byte(yaml), "invalid")
	if err == nil {
		t.Error("Expected error for invalid field name, got nil")
	}

	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

func TestLoadViewFromBytes_InvalidFormat(t *testing.T) {
	yaml := `
name: invalid_format
fields:
  - name: status
    format: invalid_format
    show: true
`

	_, err := LoadViewFromBytes([]byte(yaml), "invalid_format")
	if err == nil {
		t.Error("Expected error for invalid format, got nil")
	}

	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("Expected 'invalid format' error, got: %v", err)
	}
}

func TestLoadViewFromBytes_MissingName(t *testing.T) {
	yaml := `
fields:
  - name: status
    format: symbol
`

	// Name should be set from parameter
	view, err := LoadViewFromBytes([]byte(yaml), "from_param")
	if err != nil {
		t.Fatalf("Failed to load view: %v", err)
	}

	if view.Name != "from_param" {
		t.Errorf("Expected name 'from_param', got '%s'", view.Name)
	}
}

func TestLoadViewFromBytes_NoFields(t *testing.T) {
	yaml := `
name: no_fields
description: View with no fields
`

	_, err := LoadViewFromBytes([]byte(yaml), "no_fields")
	if err == nil {
		t.Error("Expected error for view with no fields, got nil")
	}

	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("Expected validation error for missing fields, got: %v", err)
	}
}

func TestLoadViewFromBytes_InvalidFieldOrder(t *testing.T) {
	yaml := `
name: invalid_order
fields:
  - name: status
    format: symbol
field_order:
  - status
  - non_existent_field
`

	_, err := LoadViewFromBytes([]byte(yaml), "invalid_order")
	if err == nil {
		t.Error("Expected error for invalid field_order, got nil")
	}

	if !strings.Contains(err.Error(), "field_order references undefined field") {
		t.Errorf("Expected error about undefined field in field_order, got: %v", err)
	}
}

func TestLoadViewFromBytes_DefaultValues(t *testing.T) {
	yaml := `
name: defaults
fields:
  - name: status
    show: true
  - name: summary
    show: true
`

	view, err := LoadViewFromBytes([]byte(yaml), "defaults")
	if err != nil {
		t.Fatalf("Failed to load view: %v", err)
	}

	// Check that default formats were applied
	for _, field := range view.Fields {
		defaultFormat := GetDefaultFormat(field.Name)
		if field.Format != defaultFormat {
			t.Errorf("Field %s: expected default format '%s', got '%s'",
				field.Name, defaultFormat, field.Format)
		}
	}

	// Check default date format
	if view.Display.DateFormat != "2006-01-02" {
		t.Errorf("Expected default date format '2006-01-02', got '%s'", view.Display.DateFormat)
	}
}

func TestLoadView_FileNotFound(t *testing.T) {
	_, err := LoadView("non_existent_file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}

	if !strings.Contains(err.Error(), "failed to read view file") {
		t.Errorf("Expected 'failed to read view file' error, got: %v", err)
	}
}

func TestLoadView_InvalidYAML(t *testing.T) {
	// Create temp file with invalid YAML
	tmpfile, err := os.CreateTemp("", "invalid*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	invalidYAML := `
name: invalid
fields:
  - name: status
    format: symbol
  invalid yaml structure here:::
`
	if _, err := tmpfile.Write([]byte(invalidYAML)); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpfile.Close()

	_, err = LoadView(tmpfile.Name())
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}

	if !strings.Contains(err.Error(), "failed to parse YAML") {
		t.Errorf("Expected YAML parse error, got: %v", err)
	}
}
