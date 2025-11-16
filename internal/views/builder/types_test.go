package builder

import (
	"gosynctasks/internal/views"
	"testing"
)

// TestNewViewBuilder verifies that NewViewBuilder properly initializes from field registry
func TestNewViewBuilder(t *testing.T) {
	builder := NewViewBuilder("test-view")

	// Check basic fields
	if builder.ViewName != "test-view" {
		t.Errorf("Expected ViewName 'test-view', got %q", builder.ViewName)
	}

	if builder.CurrentState != StateWelcome {
		t.Errorf("Expected CurrentState StateWelcome, got %v", builder.CurrentState)
	}

	// Check that fields are loaded from registry
	if len(builder.AvailableFields) == 0 {
		t.Fatal("No fields loaded from registry")
	}

	// Verify fields come from registry with correct defaults
	expectedFields := map[string]string{
		"status":      "symbol",
		"summary":     "full",
		"description": "truncate",
		"priority":    "number",
		"due_date":    "full",
	}

	for name, expectedFormat := range expectedFields {
		found := false
		for _, field := range builder.AvailableFields {
			if field.Name == name {
				found = true
				// Check format matches registry default
				if field.Format != expectedFormat {
					t.Errorf("Field %q has format %q, expected %q", name, field.Format, expectedFormat)
				}
				// Verify description comes from registry
				regDef, ok := views.GetFieldDefinition(name)
				if ok && field.Description != regDef.Description {
					t.Errorf("Field %q description doesn't match registry", name)
				}
				break
			}
		}
		if !found {
			t.Errorf("Expected field %q not found in AvailableFields", name)
		}
	}

	// Check pre-selection
	selectedCount := 0
	for _, field := range builder.AvailableFields {
		if field.Selected {
			selectedCount++
			if field.Name != "status" && field.Name != "summary" {
				t.Errorf("Unexpected pre-selected field: %q", field.Name)
			}
		}
	}
	if selectedCount != 2 {
		t.Errorf("Expected 2 pre-selected fields (status, summary), got %d", selectedCount)
	}
}

// TestBuildView_Success verifies successful view building
func TestBuildView_Success(t *testing.T) {
	builder := NewViewBuilder("test-view")
	builder.ViewDescription = "Test description"

	// Select a few fields
	builder.AvailableFields[0].Selected = true // status
	builder.AvailableFields[1].Selected = true // summary
	builder.UpdateSelectedFields()
	builder.UpdateFieldOrder()

	view, err := builder.BuildView()
	if err != nil {
		t.Fatalf("BuildView failed: %v", err)
	}

	if view.Name != "test-view" {
		t.Errorf("Expected view name 'test-view', got %q", view.Name)
	}

	if view.Description != "Test description" {
		t.Errorf("Expected description 'Test description', got %q", view.Description)
	}

	if len(view.Fields) < 2 {
		t.Errorf("Expected at least 2 fields, got %d", len(view.Fields))
	}
}

// TestBuildView_NoFieldsSelected verifies validation catches empty selection
func TestBuildView_NoFieldsSelected(t *testing.T) {
	builder := NewViewBuilder("test-view")

	// Deselect all fields
	for i := range builder.AvailableFields {
		builder.AvailableFields[i].Selected = false
	}
	builder.UpdateSelectedFields()
	builder.UpdateFieldOrder()

	_, err := builder.BuildView()
	if err == nil {
		t.Error("Expected error for no selected fields, got nil")
	}

	expectedMsg := "at least one field must be selected"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

// TestBuildView_InvalidFormat verifies format validation
func TestBuildView_InvalidFormat(t *testing.T) {
	builder := NewViewBuilder("test-view")

	// Set invalid format
	builder.AvailableFields[0].Selected = true
	builder.AvailableFields[0].Format = "invalid_format"
	builder.UpdateSelectedFields()
	builder.UpdateFieldOrder()

	_, err := builder.BuildView()
	if err == nil {
		t.Error("Expected error for invalid format, got nil")
	}
}

// TestUpdateFieldOrder_InitialOrder verifies initial field order setup
func TestUpdateFieldOrder_InitialOrder(t *testing.T) {
	builder := NewViewBuilder("test-view")

	// Select three fields
	builder.AvailableFields[0].Selected = true // status
	builder.AvailableFields[1].Selected = true // summary
	builder.AvailableFields[2].Selected = true // description
	builder.UpdateSelectedFields()

	if len(builder.SelectedFields) != 3 {
		t.Fatalf("Expected 3 selected fields, got %d", len(builder.SelectedFields))
	}

	// Initial update should preserve order
	builder.UpdateFieldOrder()

	if len(builder.FieldOrder) != 3 {
		t.Errorf("Expected 3 fields in order, got %d", len(builder.FieldOrder))
	}

	// Verify order matches selection
	for i, name := range builder.FieldOrder {
		if name != builder.SelectedFields[i] {
			t.Errorf("Order mismatch at index %d: got %q, expected %q", i, name, builder.SelectedFields[i])
		}
	}
}

// TestUpdateFieldOrder_PreservesExistingOrder verifies order is preserved when fields remain selected
func TestUpdateFieldOrder_PreservesExistingOrder(t *testing.T) {
	builder := NewViewBuilder("test-view")

	// Select and order three fields
	builder.AvailableFields[0].Selected = true // status
	builder.AvailableFields[1].Selected = true // summary
	builder.AvailableFields[2].Selected = true // description
	builder.UpdateSelectedFields()
	builder.UpdateFieldOrder()

	originalOrder := append([]string{}, builder.FieldOrder...)

	// Deselect middle field
	builder.AvailableFields[1].Selected = false // summary
	builder.UpdateSelectedFields()
	builder.UpdateFieldOrder()

	// Verify relative order is preserved (status before description)
	if len(builder.FieldOrder) != 2 {
		t.Fatalf("Expected 2 fields in order, got %d", len(builder.FieldOrder))
	}

	if builder.FieldOrder[0] != originalOrder[0] || builder.FieldOrder[1] != originalOrder[2] {
		t.Errorf("Order not preserved: got %v, expected [%s, %s]",
			builder.FieldOrder, originalOrder[0], originalOrder[2])
	}
}

// TestUpdateFieldOrder_AddsNewFields verifies new fields are appended
func TestUpdateFieldOrder_AddsNewFields(t *testing.T) {
	builder := NewViewBuilder("test-view")

	// Start with two fields
	builder.AvailableFields[0].Selected = true // status
	builder.AvailableFields[1].Selected = true // summary
	builder.UpdateSelectedFields()
	builder.UpdateFieldOrder()

	// Add a third field
	builder.AvailableFields[2].Selected = true // description
	builder.UpdateSelectedFields()
	builder.UpdateFieldOrder()

	if len(builder.FieldOrder) != 3 {
		t.Errorf("Expected 3 fields in order, got %d", len(builder.FieldOrder))
	}

	// New field should be last
	if builder.FieldOrder[2] != "description" {
		t.Errorf("Expected 'description' at end, got %q", builder.FieldOrder[2])
	}
}

// TestValidate_Success verifies valid configurations pass
func TestValidate_Success(t *testing.T) {
	builder := NewViewBuilder("test-view")

	// Set up valid configuration
	builder.AvailableFields[0].Selected = true
	builder.UpdateSelectedFields()

	err := builder.Validate()
	if err != nil {
		t.Errorf("Validate failed on valid config: %v", err)
	}
}

// TestValidate_NoFieldsSelected verifies validation catches empty selection
func TestValidate_NoFieldsSelected(t *testing.T) {
	builder := NewViewBuilder("test-view")

	// Deselect all
	for i := range builder.AvailableFields {
		builder.AvailableFields[i].Selected = false
	}
	builder.UpdateSelectedFields()

	err := builder.Validate()
	if err == nil {
		t.Error("Expected validation error for no selected fields")
	}
}

// TestValidate_InvalidFormat verifies format validation
func TestValidate_InvalidFormat(t *testing.T) {
	builder := NewViewBuilder("test-view")

	// Set invalid format
	builder.AvailableFields[0].Selected = true
	builder.AvailableFields[0].Format = "totally_invalid"
	builder.UpdateSelectedFields()

	err := builder.Validate()
	if err == nil {
		t.Error("Expected validation error for invalid format")
	}
}

// TestGetFieldItem_Found verifies helper finds existing fields
func TestGetFieldItem_Found(t *testing.T) {
	builder := NewViewBuilder("test-view")

	item := builder.getFieldItem("status")
	if item == nil {
		t.Fatal("getFieldItem returned nil for 'status'")
	}

	if item.Name != "status" {
		t.Errorf("Expected field name 'status', got %q", item.Name)
	}
}

// TestGetFieldItem_NotFound verifies helper returns nil for missing fields
func TestGetFieldItem_NotFound(t *testing.T) {
	builder := NewViewBuilder("test-view")

	item := builder.getFieldItem("nonexistent_field")
	if item != nil {
		t.Errorf("Expected nil for nonexistent field, got %+v", item)
	}
}

// TestGetFieldItem_Modification verifies returned pointer can modify original
func TestGetFieldItem_Modification(t *testing.T) {
	builder := NewViewBuilder("test-view")

	item := builder.getFieldItem("status")
	if item == nil {
		t.Fatal("getFieldItem returned nil")
	}

	// Modify through pointer
	item.Color = true

	// Verify original is modified
	for _, field := range builder.AvailableFields {
		if field.Name == "status" {
			if !field.Color {
				t.Error("Modification through getFieldItem pointer didn't affect original")
			}
			return
		}
	}
	t.Error("status field not found in AvailableFields")
}

// TestBuilderState_String verifies state string representations
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
		got := tt.state.String()
		if got != tt.expected {
			t.Errorf("State %d: expected %q, got %q", tt.state, tt.expected, got)
		}
	}
}
