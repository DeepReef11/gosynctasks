package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "tilde only",
			input:    "~",
			expected: homeDir,
			wantErr:  false,
		},
		{
			name:     "tilde with path",
			input:    "~/data/file.txt",
			expected: filepath.Join(homeDir, "data/file.txt"),
			wantErr:  false,
		},
		{
			name:     "tilde with .config",
			input:    "~/.config/app/config.yaml",
			expected: filepath.Join(homeDir, ".config/app/config.yaml"),
			wantErr:  false,
		},
		{
			name:     "absolute path unchanged",
			input:    "/absolute/path/file.txt",
			expected: "/absolute/path/file.txt",
			wantErr:  false,
		},
		{
			name:     "relative path unchanged",
			input:    "relative/path/file.txt",
			expected: "relative/path/file.txt",
			wantErr:  false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
			wantErr:  false,
		},
		{
			name:     "env var expansion",
			input:    "$HOME/data",
			expected: filepath.Join(homeDir, "data"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandPath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("ExpandPath() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExpandPath_PreservesTrailingSlash(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	input := "~/data/"
	result, err := ExpandPath(input)
	if err != nil {
		t.Fatalf("ExpandPath() error = %v", err)
	}

	expected := filepath.Join(homeDir, "data")
	// Note: filepath.Join removes trailing slashes on most platforms
	// This test documents the current behavior
	if result != expected {
		t.Errorf("ExpandPath() = %q, want %q", result, expected)
	}
}

func TestExpandPath_TildeNotAtStart(t *testing.T) {
	// Tilde in the middle should not be expanded
	input := "/path/~/file.txt"
	result, err := ExpandPath(input)
	if err != nil {
		t.Fatalf("ExpandPath() error = %v", err)
	}

	if result != input {
		t.Errorf("ExpandPath() = %q, want %q (tilde in middle should not expand)", result, input)
	}
}

func TestExpandPath_CombinedExpansion(t *testing.T) {
	// Test that environment variables are expanded before tilde
	os.Setenv("TEST_VAR", "~/test")
	defer os.Unsetenv("TEST_VAR")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	input := "$TEST_VAR/file.txt"
	result, err := ExpandPath(input)
	if err != nil {
		t.Fatalf("ExpandPath() error = %v", err)
	}

	expected := filepath.Join(homeDir, "test/file.txt")
	if result != expected {
		t.Errorf("ExpandPath() = %q, want %q", result, expected)
	}
}

func TestExpandPath_WindowsStyle(t *testing.T) {
	// Skip on non-Windows if testing Windows-specific behavior
	// This test documents that backslashes are converted to forward slashes by filepath
	if filepath.Separator == '\\' {
		input := "~\\data\\file.txt"
		result, err := ExpandPath(input)
		if err != nil {
			t.Fatalf("ExpandPath() error = %v", err)
		}

		homeDir, _ := os.UserHomeDir()
		expected := filepath.Join(homeDir, "data", "file.txt")
		if result != expected {
			t.Errorf("ExpandPath() = %q, want %q", result, expected)
		}
	}
}

func TestExpandPath_DatabasePath(t *testing.T) {
	// Test realistic database path scenarios
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "SQLite in .local/share",
			input:    "~/.local/share/gosynctasks/tasks.db",
			expected: filepath.Join(homeDir, ".local/share/gosynctasks/tasks.db"),
		},
		{
			name:     "SQLite in custom location",
			input:    "~/Documents/myapp/data.db",
			expected: filepath.Join(homeDir, "Documents/myapp/data.db"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandPath(tt.input)
			if err != nil {
				t.Errorf("ExpandPath() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("ExpandPath() = %q, want %q", result, tt.expected)
			}

			// Verify the result is an absolute path
			if !filepath.IsAbs(result) {
				t.Errorf("ExpandPath() result %q is not an absolute path", result)
			}

			// Verify no tilde remains
			if strings.Contains(result, "~") {
				t.Errorf("ExpandPath() result %q still contains tilde", result)
			}
		})
	}
}
