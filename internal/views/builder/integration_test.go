package builder

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestIntegration_FullWizardFlow simulates a complete wizard flow
func TestIntegration_FullWizardFlow(t *testing.T) {
	builder := NewViewBuilder("integration-test")
	m := newModel(builder)

	// Welcome -> Basic Info
	m = assertTransition(t, m, StateWelcome, StateBasicInfo)

	// Basic Info -> Field Selection
	m.textInput.SetValue("Integration test view")
	m = assertTransition(t, m, StateBasicInfo, StateFieldSelection)

	if m.builder.ViewDescription != "Integration test view" {
		t.Errorf("Expected description set, got %q", m.builder.ViewDescription)
	}

	// Field Selection -> Field Ordering (default selections)
	m = assertTransition(t, m, StateFieldSelection, StateFieldOrdering)

	if len(m.builder.FieldOrder) < 2 {
		t.Errorf("Expected at least 2 fields in order, got %d", len(m.builder.FieldOrder))
	}

	// Field Ordering -> Field Config
	m = assertTransition(t, m, StateFieldOrdering, StateFieldConfig)

	// Field Config -> Display Options
	m = assertTransition(t, m, StateFieldConfig, StateDisplayOptions)

	// Display Options -> Confirm
	m = assertTransition(t, m, StateDisplayOptions, StateConfirm)

	// Confirm -> Done (via 'y' key)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	m = updateModel(t, m, msg)

	if m.builder.CurrentState != StateDone {
		t.Errorf("Expected StateDone, got %v", m.builder.CurrentState)
	}

	if m.builder.View == nil {
		t.Fatal("Expected view to be built")
	}
}

// TestIntegration_CancelOperation simulates cancellation
func TestIntegration_CancelOperation(t *testing.T) {
	builder := NewViewBuilder("cancel-test")
	m := newModel(builder)
	m.builder.CurrentState = StateConfirm

	// Cancel with 'n'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	m = updateModel(t, m, msg)

	if m.builder.CurrentState != StateCancelled {
		t.Errorf("Expected StateCancelled, got %v", m.builder.CurrentState)
	}
}

// TestIntegration_ValidationError simulates and recovers from validation error
func TestIntegration_ValidationError(t *testing.T) {
	builder := NewViewBuilder("error-test")
	m := newModel(builder)
	m.builder.CurrentState = StateFieldSelection

	// Deselect all fields
	for i := range m.builder.AvailableFields {
		m.builder.AvailableFields[i].Selected = false
	}

	// Try to continue (should fail)
	updated, _ := m.handleEnter()
	m = updated.(builderModel)

	if m.builder.CurrentState != StateFieldSelection {
		t.Error("Should stay in StateFieldSelection")
	}
	if m.errorMsg == "" {
		t.Error("Expected error message")
	}

	// Fix and retry
	m.builder.AvailableFields[0].Selected = true
	m = assertTransition(t, m, StateFieldSelection, StateFieldOrdering)

	if m.errorMsg != "" {
		t.Errorf("Error should be cleared, got %q", m.errorMsg)
	}
}

// TestIntegration_FieldReordering tests field reordering
func TestIntegration_FieldReordering(t *testing.T) {
	builder := NewViewBuilder("reorder-test")
	m := newModel(builder)
	m.builder.CurrentState = StateFieldOrdering
	m.builder.FieldOrder = []string{"a", "b", "c"}
	m.cursor = 1

	// Move up
	msg := tea.KeyMsg{Type: tea.KeyCtrlUp}
	m = updateModel(t, m, msg)

	if m.builder.FieldOrder[0] != "b" || m.builder.FieldOrder[1] != "a" {
		t.Error("Field order not updated correctly after move up")
	}

	// Move down
	msg = tea.KeyMsg{Type: tea.KeyCtrlDown}
	m = updateModel(t, m, msg)

	if m.builder.FieldOrder[0] != "a" || m.builder.FieldOrder[1] != "b" {
		t.Error("Field order not updated correctly after move down")
	}
}

// Helper: assert state transition via handleEnter
func assertTransition(t *testing.T, m builderModel, from, to BuilderState) builderModel {
	t.Helper()
	if m.builder.CurrentState != from {
		t.Fatalf("Expected to be in %v, got %v", from, m.builder.CurrentState)
	}

	updated, _ := m.handleEnter()
	m = updated.(builderModel)

	if m.builder.CurrentState != to {
		t.Errorf("Expected transition to %v, got %v", to, m.builder.CurrentState)
	}

	return m
}

// Helper: update model with message
func updateModel(t *testing.T, m builderModel, msg tea.Msg) builderModel {
	t.Helper()
	updated, _ := m.Update(msg)
	return updated.(builderModel)
}
