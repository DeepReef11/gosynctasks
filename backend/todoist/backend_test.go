package todoist

import (
	"gosynctasks/backend"
	"testing"
)

func TestTodoistBackend_GetBackendType(t *testing.T) {
	// Test that backend type returns "todoist"
	// We create a minimal config without validation to test the type method
	tb := &TodoistBackend{}

	backendType := tb.GetBackendType()
	expected := "todoist"
	if backendType != expected {
		t.Errorf("GetBackendType() = %q, want %q", backendType, expected)
	}
}

func TestTodoistBackend_GetBackendDisplayName(t *testing.T) {
	tb := &TodoistBackend{}

	displayName := tb.GetBackendDisplayName()
	expected := "[todoist]"
	if displayName != expected {
		t.Errorf("GetBackendDisplayName() = %q, want %q", displayName, expected)
	}
}

func TestTodoistBackend_ParseStatusFlag(t *testing.T) {
	tb := &TodoistBackend{}

	tests := []struct {
		name       string
		statusFlag string
		want       string
		wantErr    bool
	}{
		{"abbreviation T", "T", "TODO", false},
		{"abbreviation D", "D", "DONE", false},
		{"full TODO", "TODO", "TODO", false},
		{"full DONE", "DONE", "DONE", false},
		{"lowercase todo", "todo", "TODO", false},
		{"invalid", "INVALID", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tb.ParseStatusFlag(tt.statusFlag)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseStatusFlag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseStatusFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTodoistBackend_GetPriorityColor(t *testing.T) {
	tb := &TodoistBackend{}

	tests := []struct {
		priority int
		hasColor bool
	}{
		{0, false}, // No color for undefined
		{1, true},  // Urgent (red)
		{3, true},  // High (yellow)
		{5, true},  // Medium (cyan)
		{7, true},  // Low (blue)
	}

	for _, tt := range tests {
		t.Run("priority", func(t *testing.T) {
			color := tb.GetPriorityColor(tt.priority)
			hasColor := len(color) > 0
			if hasColor != tt.hasColor {
				t.Errorf("GetPriorityColor(%d) hasColor = %v, want %v", tt.priority, hasColor, tt.hasColor)
			}
		})
	}
}

func TestNewTodoistBackend_MissingToken(t *testing.T) {
	config := backend.BackendConfig{
		Type:    "todoist",
		Enabled: true,
		// APIToken is missing
	}

	_, err := NewTodoistBackend(config)
	if err == nil {
		t.Error("NewTodoistBackend() expected error for missing API token, got nil")
	}
}

// Note: Testing with a real API token requires integration tests
// These unit tests only cover methods that don't require API access
