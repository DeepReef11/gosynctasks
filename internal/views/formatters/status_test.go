package formatters

import (
	"gosynctasks/backend"
	"strings"
	"testing"
)

func TestStatusFormatter_Symbol(t *testing.T) {
	ctx := NewFormatContext(nil, "2006-01-02")
	formatter := NewStatusFormatter(ctx)

	tests := []struct {
		status      string
		expectColor bool
		contains    string
	}{
		{"COMPLETED", false, "✓"},
		{"IN-PROCESS", false, "●"},
		{"CANCELLED", false, "✗"},
		{"NEEDS-ACTION", false, "○"},
		{"COMPLETED", true, "\033[32m"}, // Green
		{"IN-PROCESS", true, "\033[33m"}, // Yellow
		{"CANCELLED", true, "\033[31m"},  // Red
	}

	for _, tt := range tests {
		task := backend.Task{Status: tt.status}
		result := formatter.Format(task, "symbol", 0, tt.expectColor)

		if !strings.Contains(result, tt.contains) {
			t.Errorf("Status %s with color=%v: expected to contain %q, got %q",
				tt.status, tt.expectColor, tt.contains, result)
		}
	}
}

func TestStatusFormatter_Text(t *testing.T) {
	ctx := NewFormatContext(nil, "2006-01-02")
	formatter := NewStatusFormatter(ctx)

	tests := []struct {
		status   string
		expected string
	}{
		{"COMPLETED", "COMPLETED"},
		{"IN-PROCESS", "IN-PROCESS"},
		{"CANCELLED", "CANCELLED"},
		{"NEEDS-ACTION", "TODO"},
	}

	for _, tt := range tests {
		task := backend.Task{Status: tt.status}
		result := formatter.Format(task, "text", 0, false)

		if result != tt.expected {
			t.Errorf("Status %s text format: expected %q, got %q",
				tt.status, tt.expected, result)
		}
	}
}

func TestStatusFormatter_Emoji(t *testing.T) {
	ctx := NewFormatContext(nil, "2006-01-02")
	formatter := NewStatusFormatter(ctx)

	tests := []struct {
		status   string
		expected string
	}{
		{"COMPLETED", "✅"},
		{"IN-PROCESS", "🔄"},
		{"CANCELLED", "❌"},
		{"NEEDS-ACTION", "⭕"},
	}

	for _, tt := range tests {
		task := backend.Task{Status: tt.status}
		result := formatter.Format(task, "emoji", 0, false)

		if result != tt.expected {
			t.Errorf("Status %s emoji format: expected %q, got %q",
				tt.status, tt.expected, result)
		}
	}
}

func TestStatusFormatter_Short(t *testing.T) {
	ctx := NewFormatContext(nil, "2006-01-02")
	formatter := NewStatusFormatter(ctx)

	tests := []struct {
		status   string
		expected string
	}{
		{"COMPLETED", "D"},
		{"IN-PROCESS", "P"},
		{"CANCELLED", "C"},
		{"NEEDS-ACTION", "T"},
	}

	for _, tt := range tests {
		task := backend.Task{Status: tt.status}
		result := formatter.Format(task, "short", 0, false)

		if result != tt.expected {
			t.Errorf("Status %s short format: expected %q, got %q",
				tt.status, tt.expected, result)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "he..."},
		{"test", 0, "test"}, // No truncation
		{"a", 1, "a"},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.width)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q",
				tt.input, tt.width, result, tt.expected)
		}
	}
}

func TestStripAnsi(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"\033[31mred\033[0m", "red"},
		{"\033[32m✓\033[0m", "✓"},
		{"no\033[33mcolor\033[0mhere", "nocolorhere"},
	}

	for _, tt := range tests {
		result := stripAnsi(tt.input)
		if result != tt.expected {
			t.Errorf("stripAnsi(%q) = %q, want %q",
				tt.input, result, tt.expected)
		}
	}
}
