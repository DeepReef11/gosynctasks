package builder

import (
	"testing"
)

func TestNewViewBuilder(t *testing.T) {
	builder := NewViewBuilder("test-view")

	if builder.ViewName != "test-view" {
		t.Errorf("Expected ViewName 'test-view', got '%s'", builder.ViewName)
	}

	if len(builder.AvailableFields) != 12 {
		t.Errorf("Expected 12 available fields, got %d", len(builder.AvailableFields))
	}

	// Check that status and summary are pre-selected
	statusSelected := false
	summarySelected := false
	for _, field := range builder.AvailableFields {
		if field.Name == "status" && field.Selected {
			statusSelected = true
		}
		if field.Name == "summary" && field.Selected {
			summarySelected = true
		}
	}

	if !statusSelected {
		t.Error("Expected 'status' field to be pre-selected")
	}
	if !summarySelected {
		t.Error("Expected 'summary' field to be pre-selected")
	}

	// Check defaults
	if builder.CurrentState != StateWelcome {
		t.Errorf("Expected initial state to be StateWelcome, got %v", builder.CurrentState)
	}

	if !builder.ShowHeader || !builder.ShowBorder {
		t.Error("Expected ShowHeader and ShowBorder to be true by default")
	}

	if builder.DateFormat != "2006-01-02" {
		t.Errorf("Expected default DateFormat '2006-01-02', got '%s'", builder.DateFormat)
	}
}

func TestUpdateSelectedFields(t *testing.T) {
	builder := NewViewBuilder("test")

	// Initially, status and summary should be selected
	builder.UpdateSelectedFields()
	if len(builder.SelectedFields) != 2 {
		t.Errorf("Expected 2 selected fields initially, got %d", len(builder.SelectedFields))
	}

	// Select additional field
	builder.AvailableFields[2].Selected = true // description
	builder.UpdateSelectedFields()

	if len(builder.SelectedFields) != 3 {
		t.Errorf("Expected 3 selected fields, got %d", len(builder.SelectedFields))
	}

	// Deselect a field
	builder.AvailableFields[0].Selected = false // status
	builder.UpdateSelectedFields()

	if len(builder.SelectedFields) != 2 {
		t.Errorf("Expected 2 selected fields after deselection, got %d", len(builder.SelectedFields))
	}
}

func TestUpdateFieldOrder_FirstTime(t *testing.T) {
	builder := NewViewBuilder("test")
	builder.AvailableFields[0].Selected = true  // status
	builder.AvailableFields[1].Selected = true  // summary
	builder.AvailableFields[2].Selected = false // description
	builder.UpdateSelectedFields()

	// First time: should initialize from selected fields
	builder.UpdateFieldOrder()

	if len(builder.FieldOrder) != 2 {
		t.Errorf("Expected 2 fields in order, got %d", len(builder.FieldOrder))
	}

	// Order should match AvailableFields order
	if builder.FieldOrder[0] != builder.SelectedFields[0] ||
		builder.FieldOrder[1] != builder.SelectedFields[1] {
		t.Error("Field order doesn't match selected fields on first initialization")
	}
}

func TestUpdateFieldOrder_PreserveOrder(t *testing.T) {
	builder := NewViewBuilder("test")

	// Set up initial state: 3 selected fields in custom order
	builder.AvailableFields[0].Selected = true // status
	builder.AvailableFields[1].Selected = true // summary
	builder.AvailableFields[4].Selected = true // due_date
	builder.UpdateSelectedFields()

	// Set custom order
	builder.FieldOrder = []string{"due_date", "status", "summary"}

	// Now deselect one
	builder.AvailableFields[1].Selected = false // summary
	builder.UpdateSelectedFields()
	builder.UpdateFieldOrder()

	// Order should be preserved for remaining fields
	if len(builder.FieldOrder) != 2 {
		t.Errorf("Expected 2 fields in order, got %d", len(builder.FieldOrder))
	}

	if builder.FieldOrder[0] != "due_date" || builder.FieldOrder[1] != "status" {
		t.Errorf("Order not preserved: got %v", builder.FieldOrder)
	}
}

func TestUpdateFieldOrder_AddNewField(t *testing.T) {
	builder := NewViewBuilder("test")

	// Start with 2 fields
	builder.AvailableFields[0].Selected = true // status
	builder.AvailableFields[1].Selected = true // summary
	builder.UpdateSelectedFields()
	builder.UpdateFieldOrder()

	// Add a new field
	builder.AvailableFields[4].Selected = true // due_date
	builder.UpdateSelectedFields()
	builder.UpdateFieldOrder()

	if len(builder.FieldOrder) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(builder.FieldOrder))
	}

	// New field should be appended
	if builder.FieldOrder[2] != "due_date" {
		t.Errorf("Expected 'due_date' at end, got '%s'", builder.FieldOrder[2])
	}
}

func TestBuildView_Basic(t *testing.T) {
	builder := NewViewBuilder("test-view")
	builder.ViewDescription = "Test view description"

	// Select two fields
	builder.AvailableFields[0].Selected = true // status
	builder.AvailableFields[1].Selected = true // summary
	builder.UpdateSelectedFields()
	builder.UpdateFieldOrder()

	view, err := builder.BuildView()
	if err != nil {
		t.Fatalf("BuildView failed: %v", err)
	}

	if view.Name != "test-view" {
		t.Errorf("Expected name 'test-view', got '%s'", view.Name)
	}

	if view.Description != "Test view description" {
		t.Errorf("Expected description 'Test view description', got '%s'", view.Description)
	}

	if len(view.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(view.Fields))
	}

	if len(view.FieldOrder) != 2 {
		t.Errorf("Expected 2 field order entries, got %d", len(view.FieldOrder))
	}

	// Check display options
	if !view.Display.ShowHeader || !view.Display.ShowBorder {
		t.Error("Expected ShowHeader and ShowBorder to be true")
	}
}

func TestBuildView_EmptySelection(t *testing.T) {
	builder := NewViewBuilder("test")

	// Deselect all fields
	for i := range builder.AvailableFields {
		builder.AvailableFields[i].Selected = false
	}
	builder.UpdateSelectedFields()
	builder.UpdateFieldOrder()

	view, err := builder.BuildView()

	// Currently this doesn't return an error, but creates an empty view
	// This is a known issue that should be addressed in validation
	if err == nil && len(view.Fields) == 0 {
		// This is the current behavior
		// TODO: Should return error or warning
	}
}

func TestBuildView_WithAllOptions(t *testing.T) {
	builder := NewViewBuilder("full-test")
	builder.ViewDescription = "Full featured view"

	// Deselect all first
	for i := range builder.AvailableFields {
		builder.AvailableFields[i].Selected = false
	}

	// Select and configure one field
	builder.AvailableFields[0].Selected = true
	builder.AvailableFields[0].Color = true
	builder.AvailableFields[0].Width = 50
	builder.AvailableFields[0].Label = "Custom Status"

	builder.UpdateSelectedFields()
	builder.UpdateFieldOrder()

	// Set display options
	builder.ShowHeader = false
	builder.CompactMode = true
	builder.DateFormat = "2006-01-02"
	builder.SortBy = "priority"
	builder.SortOrder = "desc"

	view, err := builder.BuildView()
	if err != nil {
		t.Fatalf("BuildView failed: %v", err)
	}

	// Check field configuration
	if len(view.Fields) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(view.Fields))
	}

	field := view.Fields[0]
	if !field.Color {
		t.Error("Expected Color to be true")
	}
	if field.Width != 50 {
		t.Errorf("Expected Width 50, got %d", field.Width)
	}
	if field.Label != "Custom Status" {
		t.Errorf("Expected Label 'Custom Status', got '%s'", field.Label)
	}

	// Check display options
	if view.Display.ShowHeader {
		t.Error("Expected ShowHeader to be false")
	}
	if !view.Display.CompactMode {
		t.Error("Expected CompactMode to be true")
	}
	if view.Display.SortBy != "priority" {
		t.Errorf("Expected SortBy 'priority', got '%s'", view.Display.SortBy)
	}
}

func TestBuilderState_String(t *testing.T) {
	tests := []struct {
		state    BuilderState
		expected string
	}{
		{StateWelcome, "Welcome"},
		{StateBasicInfo, "Basic Info"},
		{StateFieldSelection, "Field Selection"},
		{StateFieldOrdering, "Field Ordering"},
		{StateFieldConfig, "Field Configuration"},
		{StateDisplayOptions, "Display Options"},
		{StateConfirm, "Confirm"},
		{StateDone, "Done"},
		{StateCancelled, "Cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.state.String() != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, tt.state.String())
			}
		})
	}
}
