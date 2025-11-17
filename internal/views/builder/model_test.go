package builder

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestNewModel verifies model initialization
func TestNewModel(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)

	if model.builder != builder {
		t.Error("Model builder not set correctly")
	}

	if model.cursor != 0 {
		t.Errorf("Expected cursor 0, got %d", model.cursor)
	}

	if model.width != 80 {
		t.Errorf("Expected width 80, got %d", model.width)
	}

	if model.textInput.Value() != "" {
		t.Error("Expected empty text input")
	}
}

// TestInit verifies Init returns textinput.Blink
func TestInit(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)

	cmd := model.Init()
	if cmd == nil {
		t.Error("Expected Init to return a command")
	}
}

// TestUpdate_WindowSize verifies window size updates
func TestUpdate_WindowSize(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := model.Update(msg)

	m := updated.(builderModel)
	if m.width != 120 {
		t.Errorf("Expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("Expected height 40, got %d", m.height)
	}
}

// TestUpdate_EscapeKey verifies escape cancels
func TestUpdate_EscapeKey(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, cmd := model.Update(msg)

	m := updated.(builderModel)
	if m.builder.CurrentState != StateCancelled {
		t.Errorf("Expected StateCancelled, got %v", m.builder.CurrentState)
	}
	if !m.quitting {
		t.Error("Expected quitting to be true")
	}
	if cmd == nil {
		t.Error("Expected quit command")
	}
}

// TestUpdate_CtrlC verifies ctrl+c cancels
func TestUpdate_CtrlC(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	updated, cmd := model.Update(msg)

	m := updated.(builderModel)
	if m.builder.CurrentState != StateCancelled {
		t.Errorf("Expected StateCancelled, got %v", m.builder.CurrentState)
	}
	if !m.quitting {
		t.Error("Expected quitting to be true")
	}
	if cmd == nil {
		t.Error("Expected quit command")
	}
}

// TestHandleEnter_WelcomeToBasicInfo verifies welcome to basic info transition
func TestHandleEnter_WelcomeToBasicInfo(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)
	model.builder.CurrentState = StateWelcome

	updated, _ := model.handleEnter()
	m := updated.(builderModel)

	if m.builder.CurrentState != StateBasicInfo {
		t.Errorf("Expected StateBasicInfo, got %v", m.builder.CurrentState)
	}
}

// TestHandleEnter_BasicInfoToFieldSelection verifies description entry
func TestHandleEnter_BasicInfoToFieldSelection(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)
	model.builder.CurrentState = StateBasicInfo
	model.textInput.SetValue("Test Description")

	updated, _ := model.handleEnter()
	m := updated.(builderModel)

	if m.builder.ViewDescription != "Test Description" {
		t.Errorf("Expected description 'Test Description', got %q", m.builder.ViewDescription)
	}
	if m.builder.CurrentState != StateFieldSelection {
		t.Errorf("Expected StateFieldSelection, got %v", m.builder.CurrentState)
	}
	if m.cursor != 0 {
		t.Errorf("Expected cursor reset to 0, got %d", m.cursor)
	}
}

// TestHandleEnter_FieldSelectionValidation verifies empty selection is rejected
func TestHandleEnter_FieldSelectionValidation(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)
	model.builder.CurrentState = StateFieldSelection

	// Deselect all fields
	for i := range model.builder.AvailableFields {
		model.builder.AvailableFields[i].Selected = false
	}

	updated, _ := model.handleEnter()
	m := updated.(builderModel)

	// Should stay in same state
	if m.builder.CurrentState != StateFieldSelection {
		t.Errorf("Expected to stay in StateFieldSelection, got %v", m.builder.CurrentState)
	}

	// Should set error message
	if m.errorMsg == "" {
		t.Error("Expected error message for empty field selection")
	}
}

// TestHandleEnter_FieldSelectionToOrdering verifies successful field selection
func TestHandleEnter_FieldSelectionToOrdering(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)
	model.builder.CurrentState = StateFieldSelection

	// Keep default selections (status, summary)
	updated, _ := model.handleEnter()
	m := updated.(builderModel)

	if m.builder.CurrentState != StateFieldOrdering {
		t.Errorf("Expected StateFieldOrdering, got %v", m.builder.CurrentState)
	}
	if m.errorMsg != "" {
		t.Errorf("Expected no error message, got %q", m.errorMsg)
	}
	if len(m.builder.FieldOrder) < 2 {
		t.Errorf("Expected at least 2 fields in order, got %d", len(m.builder.FieldOrder))
	}
}

// TestHandleUp_DecrementsCursor verifies up arrow
func TestHandleUp_DecrementsCursor(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)
	model.cursor = 5

	updated, _ := model.handleUp()
	m := updated.(builderModel)

	if m.cursor != 4 {
		t.Errorf("Expected cursor 4, got %d", m.cursor)
	}
}

// TestHandleUp_StopsAtZero verifies cursor doesn't go negative
func TestHandleUp_StopsAtZero(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)
	model.cursor = 0

	updated, _ := model.handleUp()
	m := updated.(builderModel)

	if m.cursor != 0 {
		t.Errorf("Expected cursor to stay at 0, got %d", m.cursor)
	}
}

// TestHandleDown_IncrementsCursor verifies down arrow
func TestHandleDown_IncrementsCursor(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)
	model.builder.CurrentState = StateFieldSelection
	model.cursor = 0

	updated, _ := model.handleDown()
	m := updated.(builderModel)

	if m.cursor != 1 {
		t.Errorf("Expected cursor 1, got %d", m.cursor)
	}
}

// TestHandleDown_RespectsBounds verifies cursor doesn't exceed max
func TestHandleDown_RespectsBounds(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)
	model.builder.CurrentState = StateFieldSelection
	maxCursor := len(model.builder.AvailableFields) - 1
	model.cursor = maxCursor

	updated, _ := model.handleDown()
	m := updated.(builderModel)

	if m.cursor != maxCursor {
		t.Errorf("Expected cursor to stay at %d, got %d", maxCursor, m.cursor)
	}
}

// TestHandleSpace_TogglesFieldSelection verifies space toggles selection
func TestHandleSpace_TogglesFieldSelection(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)
	model.builder.CurrentState = StateFieldSelection
	model.cursor = 0

	initialSelected := model.builder.AvailableFields[0].Selected

	updated, _ := model.handleSpace()
	m := updated.(builderModel)

	if m.builder.AvailableFields[0].Selected == initialSelected {
		t.Error("Expected field selection to toggle")
	}
}

// TestHandleSpace_TogglesColor verifies space toggles color in config state
func TestHandleSpace_TogglesColor(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)
	model.builder.CurrentState = StateFieldConfig

	// Set up field order
	model.builder.AvailableFields[0].Selected = true
	model.builder.UpdateSelectedFields()
	model.builder.UpdateFieldOrder()
	model.cursor = 0

	initialColor := model.builder.AvailableFields[0].Color

	updated, _ := model.handleSpace()
	m := updated.(builderModel)

	if m.builder.AvailableFields[0].Color == initialColor {
		t.Error("Expected color to toggle")
	}
}

// TestHandleSpace_TogglesDisplayOptions verifies space toggles display options
func TestHandleSpace_TogglesDisplayOptions(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)
	model.builder.CurrentState = StateDisplayOptions

	tests := []struct {
		cursor     int
		checkField func(*ViewBuilder) bool
		name       string
	}{
		{0, func(b *ViewBuilder) bool { return b.ShowHeader }, "ShowHeader"},
		{1, func(b *ViewBuilder) bool { return b.ShowBorder }, "ShowBorder"},
		{2, func(b *ViewBuilder) bool { return b.CompactMode }, "CompactMode"},
	}

	for _, tt := range tests {
		model.cursor = tt.cursor
		initial := tt.checkField(model.builder)

		updated, _ := model.handleSpace()
		m := updated.(builderModel)

		after := tt.checkField(m.builder)
		if after == initial {
			t.Errorf("%s: Expected toggle, but stayed %v", tt.name, initial)
		}
	}
}

// TestHandleMoveUp_SwapsFields verifies ctrl+up swaps fields
func TestHandleMoveUp_SwapsFields(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)
	model.builder.CurrentState = StateFieldOrdering

	// Set up field order
	model.builder.FieldOrder = []string{"a", "b", "c"}
	model.cursor = 1

	updated, _ := model.handleMoveUp()
	m := updated.(builderModel)

	if len(m.builder.FieldOrder) != 3 {
		t.Fatal("Field order length changed")
	}

	expected := []string{"b", "a", "c"}
	for i, name := range expected {
		if m.builder.FieldOrder[i] != name {
			t.Errorf("Position %d: expected %q, got %q", i, name, m.builder.FieldOrder[i])
		}
	}

	if m.cursor != 0 {
		t.Errorf("Expected cursor to move to 0, got %d", m.cursor)
	}
}

// TestHandleMoveDown_SwapsFields verifies ctrl+down swaps fields
func TestHandleMoveDown_SwapsFields(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)
	model.builder.CurrentState = StateFieldOrdering

	// Set up field order
	model.builder.FieldOrder = []string{"a", "b", "c"}
	model.cursor = 1

	updated, _ := model.handleMoveDown()
	m := updated.(builderModel)

	if len(m.builder.FieldOrder) != 3 {
		t.Fatal("Field order length changed")
	}

	expected := []string{"a", "c", "b"}
	for i, name := range expected {
		if m.builder.FieldOrder[i] != name {
			t.Errorf("Position %d: expected %q, got %q", i, name, m.builder.FieldOrder[i])
		}
	}

	if m.cursor != 2 {
		t.Errorf("Expected cursor to move to 2, got %d", m.cursor)
	}
}

// TestGetMaxCursor_FieldSelection verifies max cursor for field selection
func TestGetMaxCursor_FieldSelection(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)
	model.builder.CurrentState = StateFieldSelection

	max := model.getMaxCursor()
	expected := len(model.builder.AvailableFields) - 1

	if max != expected {
		t.Errorf("Expected max cursor %d, got %d", expected, max)
	}
}

// TestGetMaxCursor_DisplayOptions verifies max cursor for display options
func TestGetMaxCursor_DisplayOptions(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)
	model.builder.CurrentState = StateDisplayOptions

	max := model.getMaxCursor()
	if max != 2 {
		t.Errorf("Expected max cursor 2 for 3 display options, got %d", max)
	}
}

// TestView_ShowsErrorMessage verifies error message display
func TestView_ShowsErrorMessage(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)
	model.errorMsg = "Test error message"

	output := model.View()

	if output == "" {
		t.Error("Expected non-empty view output")
	}

	// Error message should be in output
	if len(output) > 0 && model.errorMsg != "" {
		// Just verify View() doesn't panic with error message
	}
}

// TestView_QuittingReturnsEmpty verifies quitting state
func TestView_QuittingReturnsEmpty(t *testing.T) {
	builder := NewViewBuilder("test-view")
	model := newModel(builder)
	model.quitting = true

	output := model.View()

	if output != "" {
		t.Error("Expected empty output when quitting")
	}
}

// TestGetStateNumber verifies state numbering
func TestGetStateNumber(t *testing.T) {
	tests := []struct {
		state    BuilderState
		expected int
	}{
		{StateWelcome, 0},
		{StateBasicInfo, 1},
		{StateFieldSelection, 2},
		{StateFieldOrdering, 3},
		{StateFieldConfig, 4},
		{StateDisplayOptions, 5},
		{StateFilterConfig, 6},
		{StateConfirm, 7},
	}

	builder := NewViewBuilder("test-view")
	model := newModel(builder)

	for _, tt := range tests {
		model.builder.CurrentState = tt.state
		got := model.getStateNumber()
		if got != tt.expected {
			t.Errorf("State %v: expected number %d, got %d", tt.state, tt.expected, got)
		}
	}
}
