package config

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
	}{
		{
			name:     "Empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "Tilde only",
			input:    "~",
			expected: homeDir,
		},
		{
			name:     "Tilde with path",
			input:    "~/.local/share/gosynctasks",
			expected: filepath.Join(homeDir, ".local/share/gosynctasks"),
		},
		{
			name:     "$HOME variable",
			input:    "$HOME/.config/gosynctasks",
			expected: filepath.Join(homeDir, ".config/gosynctasks"),
		},
		{
			name:     "$HOME in middle of path",
			input:    "/prefix/$HOME/suffix",
			expected: "/prefix/" + homeDir + "/suffix",
		},
		{
			name:     "Multiple $HOME",
			input:    "$HOME/test/$HOME/data",
			expected: homeDir + "/test/" + homeDir + "/data",
		},
		{
			name:     "Escaped tilde",
			input:    `\~/literal`,
			expected: "~/literal",
		},
		{
			name:     "Escaped dollar",
			input:    `\$HOME/literal`,
			expected: "$HOME/literal",
		},
		{
			name:     "Mixed escaped and unescaped",
			input:    `$HOME/test/\$HOME/data`,
			expected: filepath.Join(homeDir, "test/$HOME/data"),
		},
		{
			name:     "Tilde not at start",
			input:    "/path/~/not-expanded",
			expected: "/path/~/not-expanded",
		},
		{
			name:     "Escaped tilde at start",
			input:    `\~/not-expanded`,
			expected: "~/not-expanded",
		},
		{
			name:     "Complex mixed case",
			input:    `$HOME/.local/\$HOME/~/test`,
			expected: filepath.Join(homeDir, ".local/$HOME/~/test"),
		},
		{
			name:     "Double escaped",
			input:    `\$HOME/\~/test`,
			expected: "$HOME/~/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			if result != tt.expected {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExpandPath_FileURLScheme(t *testing.T) {
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
			name:     "file:// with $HOME",
			input:    "file://$HOME/.local/share/gosynctasks/tasks.json",
			expected: "file://" + filepath.Join(homeDir, ".local/share/gosynctasks/tasks.json"),
		},
		{
			name:     "file:// with tilde",
			input:    "file://~/tasks.json",
			expected: "file://" + filepath.Join(homeDir, "tasks.json"),
		},
		{
			name:     "http:// URL not affected",
			input:    "http://example.com/$HOME/test",
			expected: "http://example.com/$HOME/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			// Simulate the file:// URL expansion logic
			if strings.HasPrefix(tt.input, "file://") {
				pathPart := strings.TrimPrefix(tt.input, "file://")
				expandedPath := expandPath(pathPart)
				result = "file://" + expandedPath
			} else {
				result = tt.input
			}

			if result != tt.expected {
				t.Errorf("expand file URL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConfigPathExpansion_EndToEnd(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	// Create a temporary YAML config with path variables
	yamlConfig := `
canWriteConfig: false
backends:
  sqlite:
    type: sqlite
    enabled: true
    db_path: "$HOME/.local/share/gosynctasks/tasks.db"
  git:
    type: git
    enabled: true
    file: "~/TODO.md"
    fallback_files:
      - "~/.gosynctasks.md"
      - "$HOME/tasks.md"
  file:
    type: file
    enabled: true
    url: "file://$HOME/data/tasks.json"
  escaped:
    type: sqlite
    enabled: true
    db_path: '\$HOME/literal/path'
ui: cli
`

	// Parse the config
	config, err := parseConfig([]byte(yamlConfig), "test.yaml")
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify SQLite db_path expansion
	sqliteBackend := config.Backends["sqlite"]
	expectedSQLitePath := filepath.Join(homeDir, ".local/share/gosynctasks/tasks.db")
	if sqliteBackend.DBPath != expectedSQLitePath {
		t.Errorf("SQLite db_path: got %q, want %q", sqliteBackend.DBPath, expectedSQLitePath)
	}

	// Verify Git file expansion
	gitBackend := config.Backends["git"]
	expectedGitFile := filepath.Join(homeDir, "TODO.md")
	if gitBackend.File != expectedGitFile {
		t.Errorf("Git file: got %q, want %q", gitBackend.File, expectedGitFile)
	}

	// Verify Git fallback files expansion
	expectedFallbacks := []string{
		filepath.Join(homeDir, ".gosynctasks.md"),
		filepath.Join(homeDir, "tasks.md"),
	}
	if len(gitBackend.FallbackFiles) != len(expectedFallbacks) {
		t.Fatalf("Git fallback_files length: got %d, want %d", len(gitBackend.FallbackFiles), len(expectedFallbacks))
	}
	for i, expected := range expectedFallbacks {
		if gitBackend.FallbackFiles[i] != expected {
			t.Errorf("Git fallback_files[%d]: got %q, want %q", i, gitBackend.FallbackFiles[i], expected)
		}
	}

	// Verify File backend URL expansion
	fileBackend := config.Backends["file"]
	expectedFileURL := "file://" + filepath.Join(homeDir, "data/tasks.json")
	if fileBackend.URL != expectedFileURL {
		t.Errorf("File URL: got %q, want %q", fileBackend.URL, expectedFileURL)
	}

	// Verify escaped path doesn't expand
	escapedBackend := config.Backends["escaped"]
	expectedEscapedPath := "$HOME/literal/path"
	if escapedBackend.DBPath != expectedEscapedPath {
		t.Errorf("Escaped db_path: got %q, want %q (should not expand)", escapedBackend.DBPath, expectedEscapedPath)
	}
}
