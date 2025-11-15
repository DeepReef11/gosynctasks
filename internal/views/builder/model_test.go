package builder

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModelStateTransitions(t *testing.T) {
	builder := NewViewBuilder("test")
	m := newModel(builder)

	// Welcome → BasicInfo
	if m.builder.CurrentState != StateWelcome {
		t.Fatal("Should start at Welcome")
	}

	updated, _ := m.handleEnter()
	m = updated.(builderModel)
	if m.builder.CurrentState != StateBasicInfo {
		t.Error("Enter at Welcome should go to BasicInfo")
	}

	// BasicInfo → FieldSelection
	m.textInput.SetValue("Test description")
	updated, _ = m.handleEnter()
	m = updated.(builderModel)
	if m.builder.CurrentState != StateFieldSelection {
		t.Error("Enter at BasicInfo should go to FieldSelection")
	}
	if m.builder.ViewDescription != "Test description" {
		t.Error("Description not saved")
	}

	// FieldSelection → FieldOrdering
	updated, _ = m.handleEnter()
	m = updated.(builderModel)
	if m.builder.CurrentState != StateFieldOrdering {
		t.Error("Enter at FieldSelection should go to FieldOrdering")
	}

	// FieldOrdering → FieldConfig
	updated, _ = m.handleEnter()
	m = updated.(builderModel)
	if m.builder.CurrentState != StateFieldConfig {
		t.Error("Enter at FieldOrdering should go to FieldConfig")
	}

	// FieldConfig → DisplayOptions
	updated, _ = m.handleEnter()
	m = updated.(builderModel)
	if m.builder.CurrentState != StateDisplayOptions {
		t.Error("Enter at FieldConfig should go to DisplayOptions")
	}

	// DisplayOptions → Confirm
	updated, _ = m.handleEnter()
	m = updated.(builderModel)
	if m.builder.CurrentState != StateConfirm {
		t.Error("Enter at DisplayOptions should go to Confirm")
	}
}

func TestModelCursorMovement(t *testing.T) {
	builder := NewViewBuilder("test")
	m := newModel(builder)
	m.builder.CurrentState = StateFieldSelection
	m.cursor = 0

	// Down arrow should increase cursor
	updated, _ := m.handleDown()
	m = updated.(builderModel)
	if m.cursor != 1 {
		t.Errorf("Expected cursor 1, got %d", m.cursor)
	}

	// Multiple downs
	updated, _ = m.handleDown()
	m = updated.(builderModel)
	updated, _ = m.handleDown()
	m = updated.(builderModel)
	if m.cursor != 3 {
		t.Errorf("Expected cursor 3, got %d", m.cursor)
	}

	// Up arrow should decrease cursor
	updated, _ = m.handleUp()
	m = updated.(builderModel)
	if m.cursor != 2 {
		t.Errorf("Expected cursor 2, got %d", m.cursor)
	}

	// Up at 0 should stay at 0
	m.cursor = 0
	updated, _ = m.handleUp()
	m = updated.(builderModel)
	if m.cursor != 0 {
		t.Error("Cursor should not go below 0")
	}
}

func TestModelCursorBounds(t *testing.T) {
	builder := NewViewBuilder("test")
	m := newModel(builder)
	m.builder.CurrentState = StateFieldSelection

	maxCursor := m.getMaxCursor()
	if maxCursor != len(m.builder.AvailableFields)-1 {
		t.Errorf("Expected max cursor %d, got %d",
			len(m.builder.AvailableFields)-1, maxCursor)
	}

	// Try to move beyond max
	m.cursor = maxCursor
	updated, _ := m.handleDown()
	m = updated.(builderModel)
	if m.cursor != maxCursor {
		t.Error("Cursor should not exceed max")
	}
}

func TestModelFieldSelection(t *testing.T) {
	builder := NewViewBuilder("test")
	m := newModel(builder)
	m.builder.CurrentState = StateFieldSelection
	m.cursor = 2

	// Toggle selection with space
	initialState := m.builder.AvailableFields[2].Selected
	updated, _ := m.handleSpace()
	m = updated.(builderModel)

	if m.builder.AvailableFields[2].Selected == initialState {
		t.Error("Space should toggle field selection")
	}

	// Toggle again
	updated, _ = m.handleSpace()
	m = updated.(builderModel)
	if m.builder.AvailableFields[2].Selected != initialState {
		t.Error("Double toggle should return to initial state")
	}
}

func TestModelFieldReordering(t *testing.T) {
	builder := NewViewBuilder("test")
	m := newModel(builder)
	m.builder.CurrentState = StateFieldOrdering
	m.builder.FieldOrder = []string{"a", "b", "c"}
	m.cursor = 1

	// Move down: b↓ → a, c, b
	updated, _ := m.handleMoveDown()
	m = updated.(builderModel)
	if m.builder.FieldOrder[1] != "c" || m.builder.FieldOrder[2] != "b" {
		t.Errorf("Move down failed: got %v", m.builder.FieldOrder)
	}
	if m.cursor != 2 {
		t.Errorf("Cursor should follow moved field, got %d", m.cursor)
	}

	// Move up: b↑ → a, b, c
	updated, _ = m.handleMoveUp()
	m = updated.(builderModel)
	if m.builder.FieldOrder[1] != "b" || m.builder.FieldOrder[2] != "c" {
		t.Errorf("Move up failed: got %v", m.builder.FieldOrder)
	}
	if m.cursor != 1 {
		t.Errorf("Cursor should follow moved field, got %d", m.cursor)
	}
}

func TestModelFieldReorderingBounds(t *testing.T) {
	builder := NewViewBuilder("test")
	m := newModel(builder)
	m.builder.CurrentState = StateFieldOrdering
	m.builder.FieldOrder = []string{"a", "b", "c"}

	// Can't move first item up
	m.cursor = 0
	initialOrder := append([]string{}, m.builder.FieldOrder...)
	updated, _ := m.handleMoveUp()
	m = updated.(builderModel)

	for i := range m.builder.FieldOrder {
		if m.builder.FieldOrder[i] != initialOrder[i] {
			t.Error("First item should not move up")
		}
	}

	// Can't move last item down
	m.cursor = 2
	initialOrder = append([]string{}, m.builder.FieldOrder...)
	updated, _ = m.handleMoveDown()
	m = updated.(builderModel)

	for i := range m.builder.FieldOrder {
		if m.builder.FieldOrder[i] != initialOrder[i] {
			t.Error("Last item should not move down")
		}
	}
}

func TestModelFieldConfigNavigation(t *testing.T) {
	builder := NewViewBuilder("test")
	m := newModel(builder)
	m.builder.CurrentState = StateFieldConfig
	m.builder.FieldOrder = []string{"status", "summary", "priority"}
	m.cursor = 0

	// Tab should move to next field
	updated, _ := m.handleNext()
	m = updated.(builderModel)
	if m.cursor != 1 {
		t.Errorf("Expected cursor 1, got %d", m.cursor)
	}

	// Shift+Tab should move to previous field
	updated, _ = m.handlePrevious()
	m = updated.(builderModel)
	if m.cursor != 0 {
		t.Errorf("Expected cursor 0, got %d", m.cursor)
	}

	// Tab at last field should stay
	m.cursor = 2
	updated, _ = m.handleNext()
	m = updated.(builderModel)
	if m.cursor != 2 {
		t.Error("Tab at last field should stay at last")
	}

	// Shift+Tab at first field should stay
	m.cursor = 0
	updated, _ = m.handlePrevious()
	m = updated.(builderModel)
	if m.cursor != 0 {
		t.Error("Shift+Tab at first field should stay at first")
	}
}

func TestModelColorToggle(t *testing.T) {
	builder := NewViewBuilder("test")
	m := newModel(builder)
	m.builder.CurrentState = StateFieldConfig
	m.builder.FieldOrder = []string{"status", "summary"}
	m.cursor = 0

	// Toggle color for first field
	initialColor := m.builder.AvailableFields[0].Color
	updated, _ := m.handleSpace()
	m = updated.(builderModel)

	if m.builder.AvailableFields[0].Color == initialColor {
		t.Error("Space should toggle color in FieldConfig state")
	}
}

func TestModelDisplayOptionsToggle(t *testing.T) {
	builder := NewViewBuilder("test")
	m := newModel(builder)
	m.builder.CurrentState = StateDisplayOptions

	// Toggle ShowHeader (cursor 0)
	m.cursor = 0
	initialHeader := m.builder.ShowHeader
	updated, _ := m.handleSpace()
	m = updated.(builderModel)
	if m.builder.ShowHeader == initialHeader {
		t.Error("Space should toggle ShowHeader")
	}

	// Toggle ShowBorder (cursor 1)
	m.cursor = 1
	initialBorder := m.builder.ShowBorder
	updated, _ = m.handleSpace()
	m = updated.(builderModel)
	if m.builder.ShowBorder == initialBorder {
		t.Error("Space should toggle ShowBorder")
	}

	// Toggle CompactMode (cursor 2)
	m.cursor = 2
	initialCompact := m.builder.CompactMode
	updated, _ = m.handleSpace()
	m = updated.(builderModel)
	if m.builder.CompactMode == initialCompact {
		t.Error("Space should toggle CompactMode")
	}
}

func TestModelConfirmAccept(t *testing.T) {
	builder := NewViewBuilder("test")
	m := newModel(builder)

	// Set up valid builder state (status and summary are pre-selected)
	builder.UpdateSelectedFields()
	builder.UpdateFieldOrder()
	m.builder.CurrentState = StateConfirm

	// Press Y
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(builderModel)

	if m.builder.CurrentState != StateDone {
		t.Error("Y at Confirm should go to StateDone")
	}

	if m.builder.View == nil {
		t.Error("View should be built after confirmation")
	}
}

func TestModelConfirmCancel(t *testing.T) {
	builder := NewViewBuilder("test")
	m := newModel(builder)
	m.builder.CurrentState = StateConfirm

	// Press N
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(builderModel)

	if m.builder.CurrentState != StateCancelled {
		t.Error("N at Confirm should go to StateCancelled")
	}
}

func TestModelCancelAnytime(t *testing.T) {
	states := []BuilderState{
		StateWelcome,
		StateBasicInfo,
		StateFieldSelection,
		StateFieldOrdering,
		StateFieldConfig,
		StateDisplayOptions,
	}

	for _, state := range states {
		builder := NewViewBuilder("test")
		m := newModel(builder)
		m.builder.CurrentState = state

		// Press Ctrl+C
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		m = updated.(builderModel)

		if m.builder.CurrentState != StateCancelled {
			t.Errorf("Ctrl+C at %v should cancel", state)
		}

		if !m.quitting {
			t.Errorf("Should be quitting after cancel at %v", state)
		}
	}
}

func TestModelGetMaxCursor(t *testing.T) {
	builder := NewViewBuilder("test")
	m := newModel(builder)

	tests := []struct {
		state       BuilderState
		setupFunc   func()
		expectedMax int
	}{
		{
			state: StateFieldSelection,
			setupFunc: func() {
				// 12 available fields
			},
			expectedMax: 11,
		},
		{
			state: StateFieldOrdering,
			setupFunc: func() {
				m.builder.FieldOrder = []string{"a", "b", "c"}
			},
			expectedMax: 2,
		},
		{
			state: StateFieldConfig,
			setupFunc: func() {
				m.builder.FieldOrder = []string{"a", "b", "c", "d"}
			},
			expectedMax: 3,
		},
		{
			state: StateDisplayOptions,
			setupFunc: func() {
				// Fixed 3 options
			},
			expectedMax: 2,
		},
	}

	for _, tt := range tests {
		m.builder.CurrentState = tt.state
		if tt.setupFunc != nil {
			tt.setupFunc()
		}

		maxCursor := m.getMaxCursor()
		if maxCursor != tt.expectedMax {
			t.Errorf("State %v: expected max cursor %d, got %d",
				tt.state, tt.expectedMax, maxCursor)
		}
	}
}

func TestModelWindowSize(t *testing.T) {
	builder := NewViewBuilder("test")
	m := newModel(builder)

	// Send window size message
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(builderModel)

	if m.width != 120 {
		t.Errorf("Expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("Expected height 40, got %d", m.height)
	}
}
