package backend

import (
	"net/url"
	"strings"
	"testing"
)

// Helper function to parse URLs in tests
func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic("Failed to parse URL: " + err.Error())
	}
	return u
}

// TestNextcloudBackend_GetBackendDisplayName tests Nextcloud backend display name
func TestNextcloudBackend_GetBackendDisplayName(t *testing.T) {
	backend := &NextcloudBackend{
		Connector: ConnectorConfig{},
		username:  "testuser",
	}
	backend.Connector.URL = mustParseURL("nextcloud://testuser:pass@localhost:8080")

	displayName := backend.GetBackendDisplayName()
	expected := "[nextcloud:testuser@localhost:8080]"
	if displayName != expected {
		t.Errorf("GetBackendDisplayName() = %q, want %q", displayName, expected)
	}
}

// TestNextcloudBackend_GetBackendType tests Nextcloud backend type
func TestNextcloudBackend_GetBackendType(t *testing.T) {
	backend := &NextcloudBackend{}
	backendType := backend.GetBackendType()
	expected := "nextcloud"
	if backendType != expected {
		t.Errorf("GetBackendType() = %q, want %q", backendType, expected)
	}
}

// TestNextcloudBackend_GetBackendContext tests Nextcloud backend context
func TestNextcloudBackend_GetBackendContext(t *testing.T) {
	backend := &NextcloudBackend{
		Connector: ConnectorConfig{},
		username:  "admin",
	}
	backend.Connector.URL = mustParseURL("nextcloud://admin:secret@example.com")

	context := backend.GetBackendContext()
	expected := "admin@example.com"
	if context != expected {
		t.Errorf("GetBackendContext() = %q, want %q", context, expected)
	}
}

// TestSQLiteBackend_GetBackendDisplayName tests SQLite backend display name
func TestSQLiteBackend_GetBackendDisplayName(t *testing.T) {
	backend := &SQLiteBackend{
		Config: BackendConfig{
			DBPath: "/home/user/.local/share/gosynctasks/tasks.db",
		},
	}

	displayName := backend.GetBackendDisplayName()
	if !strings.Contains(displayName, "[sqlite:") {
		t.Errorf("GetBackendDisplayName() = %q, should contain '[sqlite:'", displayName)
	}
	if !strings.Contains(displayName, "tasks.db]") {
		t.Errorf("GetBackendDisplayName() = %q, should contain 'tasks.db]'", displayName)
	}
}

// TestSQLiteBackend_GetBackendType tests SQLite backend type
func TestSQLiteBackend_GetBackendType(t *testing.T) {
	backend := &SQLiteBackend{}
	backendType := backend.GetBackendType()
	expected := "sqlite"
	if backendType != expected {
		t.Errorf("GetBackendType() = %q, want %q", backendType, expected)
	}
}

// TestSQLiteBackend_GetBackendContext tests SQLite backend context
func TestSQLiteBackend_GetBackendContext(t *testing.T) {
	backend := &SQLiteBackend{
		Config: BackendConfig{
			DBPath: "/tmp/test.db",
		},
	}

	context := backend.GetBackendContext()
	if !strings.Contains(context, "test.db") {
		t.Errorf("GetBackendContext() = %q, should contain 'test.db'", context)
	}
}

// TestFileBackend_GetBackendDisplayName tests File backend display name
func TestFileBackend_GetBackendDisplayName(t *testing.T) {
	backend := &FileBackend{
		Connector: ConnectorConfig{},
	}
	backend.Connector.URL = mustParseURL("file:///home/user/tasks.txt")

	displayName := backend.GetBackendDisplayName()
	expected := "[file:/home/user/tasks.txt]"
	if displayName != expected {
		t.Errorf("GetBackendDisplayName() = %q, want %q", displayName, expected)
	}
}

// TestFileBackend_GetBackendType tests File backend type
func TestFileBackend_GetBackendType(t *testing.T) {
	backend := &FileBackend{}
	backendType := backend.GetBackendType()
	expected := "file"
	if backendType != expected {
		t.Errorf("GetBackendType() = %q, want %q", backendType, expected)
	}
}

// TestFileBackend_GetBackendContext tests File backend context
func TestFileBackend_GetBackendContext(t *testing.T) {
	backend := &FileBackend{
		Connector: ConnectorConfig{},
	}
	backend.Connector.URL = mustParseURL("file:///var/data/todos.json")

	context := backend.GetBackendContext()
	expected := "/var/data/todos.json"
	if context != expected {
		t.Errorf("GetBackendContext() = %q, want %q", context, expected)
	}
}

// TestGitBackend_GetBackendDisplayName tests Git backend display name
func TestGitBackend_GetBackendDisplayName(t *testing.T) {
	backend := &GitBackend{
		repoPath: "/home/user/myproject",
		fileName: "TODO.md",
	}

	displayName := backend.GetBackendDisplayName()
	expected := "[git:/home/user/myproject/TODO.md]"
	if displayName != expected {
		t.Errorf("GetBackendDisplayName() = %q, want %q", displayName, expected)
	}
}

// TestGitBackend_GetBackendType tests Git backend type
func TestGitBackend_GetBackendType(t *testing.T) {
	backend := &GitBackend{}
	backendType := backend.GetBackendType()
	expected := "git"
	if backendType != expected {
		t.Errorf("GetBackendType() = %q, want %q", backendType, expected)
	}
}

// TestGitBackend_GetBackendContext tests Git backend context
func TestGitBackend_GetBackendContext(t *testing.T) {
	backend := &GitBackend{
		repoPath: "/opt/projects/gosynctasks",
		fileName: "TASKS.md",
	}

	context := backend.GetBackendContext()
	expected := "/opt/projects/gosynctasks/TASKS.md"
	if context != expected {
		t.Errorf("GetBackendContext() = %q, want %q", context, expected)
	}
}

// TestTaskList_StringWithBackend tests TaskList backend display integration
func TestTaskList_StringWithBackend(t *testing.T) {
	backend := &NextcloudBackend{
		Connector: ConnectorConfig{},
		username:  "admin",
	}
	backend.Connector.URL = mustParseURL("nextcloud://admin:pass@localhost")

	taskList := TaskList{
		ID:          "list-1",
		Name:        "Test List",
		Description: "A test task list",
	}

	output := taskList.StringWithBackend(backend)

	// Check that output contains the list name
	if !strings.Contains(output, "Test List") {
		t.Errorf("StringWithBackend() output should contain list name, got: %q", output)
	}

	// Check that output contains backend info
	if !strings.Contains(output, "[nextcloud:admin@localhost]") {
		t.Errorf("StringWithBackend() output should contain backend info, got: %q", output)
	}

	// Check that output contains box drawing characters
	if !strings.Contains(output, "┌") || !strings.Contains(output, "┐") {
		t.Errorf("StringWithBackend() output should contain box drawing characters, got: %q", output)
	}
}

// TestTaskList_StringWithBackend_NilBackend tests fallback when backend is nil
func TestTaskList_StringWithBackend_NilBackend(t *testing.T) {
	taskList := TaskList{
		ID:   "list-1",
		Name: "Test List",
	}

	output := taskList.StringWithBackend(nil)

	// Should fall back to standard display without backend info
	if !strings.Contains(output, "Test List") {
		t.Errorf("StringWithBackend(nil) should contain list name, got: %q", output)
	}

	// Should not contain backend info brackets
	if strings.Contains(output, "[nextcloud:") || strings.Contains(output, "[sqlite:") {
		t.Errorf("StringWithBackend(nil) should not contain backend info, got: %q", output)
	}
}

// TestTaskList_StringWithWidthAndBackend_Truncation tests truncation with long names
func TestTaskList_StringWithWidthAndBackend_Truncation(t *testing.T) {
	backend := &SQLiteBackend{
		Config: BackendConfig{
			DBPath: "/very/long/path/to/database/that/exceeds/normal/width.db",
		},
	}

	taskList := TaskList{
		ID:          "list-1",
		Name:        "A Very Long Task List Name That Should Be Truncated",
		Description: "With an even longer description that definitely won't fit",
	}

	// Use a narrow terminal width to force truncation
	output := taskList.StringWithWidthAndBackend(50, backend)

	// Should still contain box drawing characters
	if !strings.Contains(output, "┌") || !strings.Contains(output, "┐") {
		t.Errorf("StringWithWidthAndBackend() should contain box drawing characters even when truncated, got: %q", output)
	}

	// Output should not be empty
	if len(output) == 0 {
		t.Errorf("StringWithWidthAndBackend() should not be empty")
	}
}
