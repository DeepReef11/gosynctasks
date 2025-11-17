package main

import (
	"gosynctasks/internal/views"
	"testing"
)

func TestGetViewTemplate(t *testing.T) {
	templates := []string{"minimal", "full", "kanban", "timeline", "compact"}

	for _, templateName := range templates {
		t.Run(templateName, func(t *testing.T) {
			view, err := views.ResolveView(templateName)
			if err != nil {
				t.Fatalf("Failed to get template '%s': %v", templateName, err)
			}

			if view.Name != templateName {
				t.Errorf("Expected template name '%s', got '%s'", templateName, view.Name)
			}

			if view.Description == "" {
				t.Errorf("Template '%s' has no description", templateName)
			}

			if len(view.Fields) == 0 {
				t.Errorf("Template '%s' has no fields", templateName)
			}

			// Check all fields have valid configurations
			for _, field := range view.Fields {
				if field.Name == "" {
					t.Errorf("Template '%s' has field with empty name", templateName)
				}

				if field.Format == "" {
					t.Errorf("Template '%s' field '%s' has empty format", templateName, field.Name)
				}
			}

			// Check display options are set
			if view.Display.DateFormat == "" {
				t.Errorf("Template '%s' has no date format", templateName)
			}
		})
	}
}

func TestGetViewTemplate_Invalid(t *testing.T) {
	_, err := views.ResolveView("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent template, got nil")
	}
}

func TestGetViewTemplate_Minimal(t *testing.T) {
	view, err := views.ResolveView("minimal")
	if err != nil {
		t.Fatalf("Failed to get minimal template: %v", err)
	}

	// Minimal should be compact
	if !view.Display.CompactMode {
		t.Error("Minimal template should have compact mode enabled")
	}

	// Minimal should have few fields
	if len(view.Fields) > 5 {
		t.Errorf("Minimal template should have few fields, got %d", len(view.Fields))
	}
}

func TestGetViewTemplate_Full(t *testing.T) {
	view, err := views.ResolveView("full")
	if err != nil {
		t.Fatalf("Failed to get full template: %v", err)
	}

	// Full should have many fields
	if len(view.Fields) < 8 {
		t.Errorf("Full template should have many fields, got %d", len(view.Fields))
	}

	// Full should have field order specified
	if len(view.FieldOrder) == 0 {
		t.Error("Full template should have field_order specified")
	}

	// Field order should match number of fields
	if len(view.FieldOrder) != len(view.Fields) {
		t.Errorf("Field order length (%d) doesn't match fields length (%d)",
			len(view.FieldOrder), len(view.Fields))
	}
}

func TestGetViewTemplate_Kanban(t *testing.T) {
	view, err := views.ResolveView("kanban")
	if err != nil {
		t.Fatalf("Failed to get kanban template: %v", err)
	}

	// Kanban should use emoji for status
	hasEmojiStatus := false
	for _, field := range view.Fields {
		if field.Name == "status" && field.Format == "emoji" {
			hasEmojiStatus = true
			break
		}
	}

	if !hasEmojiStatus {
		t.Error("Kanban template should use emoji format for status")
	}

	// Kanban should be compact
	if !view.Display.CompactMode {
		t.Error("Kanban template should have compact mode enabled")
	}
}

func TestGetViewTemplate_Timeline(t *testing.T) {
	view, err := views.ResolveView("timeline")
	if err != nil {
		t.Fatalf("Failed to get timeline template: %v", err)
	}

	// Timeline should have start_date and due_date
	hasStartDate := false
	hasDueDate := false

	for _, field := range view.Fields {
		if field.Name == "start_date" {
			hasStartDate = true
		}
		if field.Name == "due_date" {
			hasDueDate = true
		}
	}

	if !hasStartDate {
		t.Error("Timeline template should have start_date field")
	}

	if !hasDueDate {
		t.Error("Timeline template should have due_date field")
	}

	// Timeline should sort by start_date
	if view.Display.SortBy != "start_date" {
		t.Errorf("Timeline should sort by start_date, got '%s'", view.Display.SortBy)
	}
}

func TestGetViewTemplate_Compact(t *testing.T) {
	view, err := views.ResolveView("compact")
	if err != nil {
		t.Fatalf("Failed to get compact template: %v", err)
	}

	// Compact should be in compact mode
	if !view.Display.CompactMode {
		t.Error("Compact template should have compact mode enabled")
	}

	// Compact should hide borders
	if view.Display.ShowHeader {
		t.Error("Compact template should hide header")
	}

	if view.Display.ShowBorder {
		t.Error("Compact template should hide border")
	}
}

func TestTemplateNameValidation(t *testing.T) {
	templates := map[string]bool{
		"minimal":  true,
		"full":     true,
		"kanban":   true,
		"timeline": true,
		"compact":  true,
		"invalid":  false,
		"":         false,
	}

	for name, shouldExist := range templates {
		_, err := views.ResolveView(name)

		if shouldExist && err != nil {
			t.Errorf("Template '%s' should exist but got error: %v", name, err)
		}

		if !shouldExist && err == nil {
			t.Errorf("Template '%s' should not exist but got no error", name)
		}
	}
}
