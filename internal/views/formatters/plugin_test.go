package formatters

import (
	"gosynctasks/backend"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPluginFormatter_Format(t *testing.T) {
	ctx := NewFormatContext(nil, "2006-01-02")

	// Create a temporary test script
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_formatter.sh")

	// Test case 1: Simple echo script
	t.Run("simple echo formatter", func(t *testing.T) {
		script := `#!/bin/sh
cat | jq -r '.summary' | tr '[:lower:]' '[:upper:]'
`
		if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		formatter := NewPluginFormatter(ctx, scriptPath, nil, 1000, nil)
		task := backend.Task{
			Summary: "test task",
			Status:  "TODO",
		}

		result := formatter.Format(task, "", 0, false)
		if result != "TEST TASK" {
			t.Errorf("Expected 'TEST TASK', got '%s'", result)
		}
	})

	// Test case 2: Custom status formatter with color
	t.Run("custom status formatter", func(t *testing.T) {
		script := `#!/bin/sh
read input
status=$(echo "$input" | jq -r '.status')
case "$status" in
    "TODO") echo "⏳ TODO" ;;
    "DONE") echo "✅ DONE" ;;
    *) echo "📝 $status" ;;
esac
`
		if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		formatter := NewPluginFormatter(ctx, scriptPath, nil, 1000, nil)

		task := backend.Task{Status: "TODO"}
		result := formatter.Format(task, "", 0, false)
		if result != "⏳ TODO" {
			t.Errorf("Expected '⏳ TODO', got '%s'", result)
		}

		task.Status = "DONE"
		result = formatter.Format(task, "", 0, false)
		if result != "✅ DONE" {
			t.Errorf("Expected '✅ DONE', got '%s'", result)
		}
	})

	// Test case 3: Script with arguments
	t.Run("formatter with arguments", func(t *testing.T) {
		script := `#!/bin/sh
prefix="$1"
cat | jq -r '.summary' | sed "s/^/$prefix /"
`
		if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		formatter := NewPluginFormatter(ctx, scriptPath, []string{"[TASK]"}, 1000, nil)
		task := backend.Task{Summary: "my task"}

		result := formatter.Format(task, "", 0, false)
		if result != "[TASK] my task" {
			t.Errorf("Expected '[TASK] my task', got '%s'", result)
		}
	})

	// Test case 4: Timeout handling
	t.Run("timeout handling", func(t *testing.T) {
		script := `#!/bin/sh
sleep 5
echo "done"
`
		if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		// Use a short timeout (100ms)
		formatter := NewPluginFormatter(ctx, scriptPath, nil, 100, nil)
		task := backend.Task{Summary: "test"}

		result := formatter.Format(task, "", 0, false)
		if !strings.Contains(result, "timed out") {
			t.Errorf("Expected timeout error, got '%s'", result)
		}
	})

	// Test case 5: Error handling
	t.Run("script error handling", func(t *testing.T) {
		script := `#!/bin/sh
echo "error message" >&2
exit 1
`
		if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		formatter := NewPluginFormatter(ctx, scriptPath, nil, 1000, nil)
		task := backend.Task{Summary: "test"}

		result := formatter.Format(task, "", 0, false)
		if !strings.Contains(result, "plugin error") {
			t.Errorf("Expected error message, got '%s'", result)
		}
	})

	// Test case 6: Date formatting
	t.Run("date formatting", func(t *testing.T) {
		script := `#!/bin/sh
cat | jq -r 'if .due_date then "Due: " + .due_date else "No due date" end'
`
		if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		formatter := NewPluginFormatter(ctx, scriptPath, nil, 1000, nil)

		dueDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		task := backend.Task{
			Summary: "test",
			DueDate: &dueDate,
		}

		result := formatter.Format(task, "", 0, false)
		if !strings.Contains(result, "2025-01-15") {
			t.Errorf("Expected date in result, got '%s'", result)
		}
	})

	// Test case 7: Priority formatting with custom logic
	t.Run("priority with custom logic", func(t *testing.T) {
		script := `#!/bin/sh
read input
priority=$(echo "$input" | jq -r '.priority')
if [ "$priority" = "1" ]; then
    echo "🔥 CRITICAL"
elif [ "$priority" -le "3" ]; then
    echo "⚡ HIGH"
elif [ "$priority" -le "6" ]; then
    echo "📌 MEDIUM"
else
    echo "💤 LOW"
fi
`
		if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		formatter := NewPluginFormatter(ctx, scriptPath, nil, 1000, nil)

		tests := []struct {
			priority int
			expected string
		}{
			{1, "🔥 CRITICAL"},
			{2, "⚡ HIGH"},
			{5, "📌 MEDIUM"},
			{9, "💤 LOW"},
		}

		for _, tc := range tests {
			task := backend.Task{Priority: tc.priority}
			result := formatter.Format(task, "", 0, false)
			if result != tc.expected {
				t.Errorf("Priority %d: expected '%s', got '%s'", tc.priority, tc.expected, result)
			}
		}
	})
}

func TestPluginFormatter_DefaultTimeout(t *testing.T) {
	ctx := NewFormatContext(nil, "2006-01-02")

	// Create formatter with 0 timeout (should use default)
	formatter := NewPluginFormatter(ctx, "echo", nil, 0, nil)

	if formatter.timeout != 1000*time.Millisecond {
		t.Errorf("Expected default timeout of 1000ms, got %v", formatter.timeout)
	}
}

func TestPluginFormatter_MaxTimeout(t *testing.T) {
	ctx := NewFormatContext(nil, "2006-01-02")

	// Create formatter with excessive timeout (should be capped at 5000ms)
	formatter := NewPluginFormatter(ctx, "echo", nil, 10000, nil)

	if formatter.timeout != 5000*time.Millisecond {
		t.Errorf("Expected max timeout of 5000ms, got %v", formatter.timeout)
	}
}

func TestPluginFormatter_EnvironmentVariables(t *testing.T) {
	ctx := NewFormatContext(nil, "2006-01-02")

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_env.sh")

	script := `#!/bin/sh
echo "$CUSTOM_VAR"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	env := map[string]string{
		"CUSTOM_VAR": "test_value",
	}

	formatter := NewPluginFormatter(ctx, scriptPath, nil, 1000, env)
	task := backend.Task{Summary: "test"}

	result := formatter.Format(task, "", 0, false)
	if result != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", result)
	}
}
