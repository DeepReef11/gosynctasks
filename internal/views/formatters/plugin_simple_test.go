package formatters

import (
	"gosynctasks/backend"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestPluginFormatter_Simple tests plugin formatters without external dependencies
func TestPluginFormatter_Simple(t *testing.T) {
	ctx := NewFormatContext(nil, "2006-01-02")

	// Create a temporary test script
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_formatter.sh")

	// Test case 1: Simple echo script (no dependencies)
	t.Run("simple echo formatter", func(t *testing.T) {
		script := `#!/bin/sh
echo "FORMATTED OUTPUT"
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
		if result != "FORMATTED OUTPUT" {
			t.Errorf("Expected 'FORMATTED OUTPUT', got '%s'", result)
		}
	})

	// Test case 2: Script with arguments
	t.Run("formatter with arguments", func(t *testing.T) {
		script := `#!/bin/sh
echo "PREFIX: $1"
`
		if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		formatter := NewPluginFormatter(ctx, scriptPath, []string{"ARG1"}, 1000, nil)
		task := backend.Task{Summary: "my task"}

		result := formatter.Format(task, "", 0, false)
		if result != "PREFIX: ARG1" {
			t.Errorf("Expected 'PREFIX: ARG1', got '%s'", result)
		}
	})

	// Test case 3: Script with environment variables
	t.Run("formatter with environment", func(t *testing.T) {
		script := `#!/bin/sh
echo "ENV: $MY_VAR"
`
		if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		env := map[string]string{"MY_VAR": "test_value"}
		formatter := NewPluginFormatter(ctx, scriptPath, nil, 1000, env)
		task := backend.Task{Summary: "my task"}

		result := formatter.Format(task, "", 0, false)
		if result != "ENV: test_value" {
			t.Errorf("Expected 'ENV: test_value', got '%s'", result)
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
		// Should contain timeout error message
		if result == "" || result == "done" {
			t.Errorf("Expected timeout error, got '%s'", result)
		}
	})

	// Test case 5: Script error handling
	t.Run("error handling", func(t *testing.T) {
		script := `#!/bin/sh
exit 1
`
		if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		formatter := NewPluginFormatter(ctx, scriptPath, nil, 1000, nil)
		task := backend.Task{Summary: "test"}

		result := formatter.Format(task, "", 0, false)
		// Should contain error message
		if result == "" || !contains(result, "error") {
			t.Errorf("Expected error message, got '%s'", result)
		}
	})
}

// TestPluginFormatter_WithJq tests plugin formatters that require jq
func TestPluginFormatter_WithJq(t *testing.T) {
	// Check if jq is available
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not found - skipping tests that require jq")
	}

	ctx := NewFormatContext(nil, "2006-01-02")

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_formatter.sh")

	// Test case: JSON parsing with jq
	t.Run("json parsing with jq", func(t *testing.T) {
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

	// Test case: Custom status formatter with color
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
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
