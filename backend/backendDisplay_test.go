package backend_test

import (
	"net/url"
	"strings"
	"testing"

	"gosynctasks/backend"
	"gosynctasks/backend/file"
	"gosynctasks/backend/git"
	"gosynctasks/backend/nextcloud"
	"gosynctasks/backend/sqlite"
)

// Helper function to parse URLs in tests
func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic("Failed to parse URL: " + err.Error())
	}
	return u
}

// Testnextcloud.NextcloudBackend_GetBackendDisplayName tests Nextcloud backend display name
func TestNextcloudBackend_GetBackendDisplayName(t *testing.T) {
	ncBackend := &nextcloud.NextcloudBackend{
		Connector: backend.ConnectorConfig{},
	}
	ncBackend.Connector.URL = mustParseURL("nextcloud://testuser:pass@localhost:8080")

	displayName := ncBackend.GetBackendDisplayName()
	expected := "[nextcloud:testuser@localhost:8080]"
	if displayName != expected {
		t.Errorf("GetBackendDisplayName() = %q, want %q", displayName, expected)
	}
}

// Testnextcloud.NextcloudBackend_GetBackendType tests Nextcloud backend type
func TestNextcloudBackend_GetBackendType(t *testing.T) {
	ncBackend := &nextcloud.NextcloudBackend{}
	backendType := ncBackend.GetBackendType()
	expected := "nextcloud"
	if backendType != expected {
		t.Errorf("GetBackendType() = %q, want %q", backendType, expected)
	}
}

// Testnextcloud.NextcloudBackend_GetBackendContext tests Nextcloud backend context
func TestNextcloudBackend_GetBackendContext(t *testing.T) {
	ncBackend := &nextcloud.NextcloudBackend{
		Connector: backend.ConnectorConfig{},
	}
	ncBackend.Connector.URL = mustParseURL("nextcloud://admin:secret@example.com")

	context := ncBackend.GetBackendContext()
	expected := "admin@example.com"
	if context != expected {
		t.Errorf("GetBackendContext() = %q, want %q", context, expected)
	}
}

// Testsqlite.SQLiteBackend_GetBackendDisplayName tests SQLite backend display name
func TestSQLiteBackend_GetBackendDisplayName(t *testing.T) {
	sbBackend := &sqlite.SQLiteBackend{
		Config: backend.BackendConfig{
			DBPath: "/home/user/.local/share/gosynctasks/tasks.db",
		},
	}

	displayName := sbBackend.GetBackendDisplayName()
	if !strings.Contains(displayName, "[sqlite:") {
		t.Errorf("GetBackendDisplayName() = %q, should contain '[sqlite:'", displayName)
	}
	if !strings.Contains(displayName, "tasks.db]") {
		t.Errorf("GetBackendDisplayName() = %q, should contain 'tasks.db]'", displayName)
	}
}

// Testsqlite.SQLiteBackend_GetBackendType tests SQLite backend type
func TestSQLiteBackend_GetBackendType(t *testing.T) {
	sbBackend := &sqlite.SQLiteBackend{}
	backendType := sbBackend.GetBackendType()
	expected := "sqlite"
	if backendType != expected {
		t.Errorf("GetBackendType() = %q, want %q", backendType, expected)
	}
}

// Testsqlite.SQLiteBackend_GetBackendContext tests SQLite backend context
func TestSQLiteBackend_GetBackendContext(t *testing.T) {
	sbBackend := &sqlite.SQLiteBackend{
		Config: backend.BackendConfig{
			DBPath: "/tmp/test.db",
		},
	}

	context := sbBackend.GetBackendContext()
	if !strings.Contains(context, "test.db") {
		t.Errorf("GetBackendContext() = %q, should contain 'test.db'", context)
	}
}

// Testfile.FileBackend_GetBackendDisplayName tests File backend display name
func TestFileBackend_GetBackendDisplayName(t *testing.T) {
	fileBackend := &file.FileBackend{
		Connector: backend.ConnectorConfig{},
	}
	fileBackend.Connector.URL = mustParseURL("file:///home/user/tasks.txt")

	displayName := fileBackend.GetBackendDisplayName()
	expected := "[file:/home/user/tasks.txt]"
	if displayName != expected {
		t.Errorf("GetBackendDisplayName() = %q, want %q", displayName, expected)
	}
}

// Testfile.FileBackend_GetBackendType tests File backend type
func TestFileBackend_GetBackendType(t *testing.T) {
	fileBackend := &file.FileBackend{}
	backendType := fileBackend.GetBackendType()
	expected := "file"
	if backendType != expected {
		t.Errorf("GetBackendType() = %q, want %q", backendType, expected)
	}
}

// Testfile.FileBackend_GetBackendContext tests File backend context
func TestFileBackend_GetBackendContext(t *testing.T) {
	fileBackend := &file.FileBackend{
		Connector: backend.ConnectorConfig{},
	}
	fileBackend.Connector.URL = mustParseURL("file:///var/data/todos.json")

	context := fileBackend.GetBackendContext()
	expected := "/var/data/todos.json"
	if context != expected {
		t.Errorf("GetBackendContext() = %q, want %q", context, expected)
	}
}

// Testgit.GitBackend_GetBackendDisplayName tests Git backend display name
func TestGitBackend_GetBackendDisplayName(t *testing.T) {
	gitBackend := &git.GitBackend{
	}

	displayName := gitBackend.GetBackendDisplayName()
	expected := "[git:myproject/TODO.md]"
	if displayName != expected {
		t.Errorf("GetBackendDisplayName() = %q, want %q", displayName, expected)
	}
}

// Testgit.GitBackend_GetBackendType tests Git backend type
func TestGitBackend_GetBackendType(t *testing.T) {
	gitBackend := &git.GitBackend{}
	backendType := gitBackend.GetBackendType()
	expected := "git"
	if backendType != expected {
		t.Errorf("GetBackendType() = %q, want %q", backendType, expected)
	}
}

// Testgit.GitBackend_GetBackendContext tests Git backend context
func TestGitBackend_GetBackendContext(t *testing.T) {
	gitBackend := &git.GitBackend{
	}

	context := gitBackend.GetBackendContext()
	expected := "gosynctasks/TASKS.md"
	if context != expected {
		t.Errorf("GetBackendContext() = %q, want %q", context, expected)
	}
}

// TestTaskList_StringWithBackend tests TaskList backend display integration
func TestTaskList_StringWithBackend(t *testing.T) {
	ncBackend := &nextcloud.NextcloudBackend{
		Connector: backend.ConnectorConfig{},
	}
	ncBackend.Connector.URL = mustParseURL("nextcloud://admin:pass@localhost")

	taskList := backend.TaskList{
		ID:          "list-1",
		Name:        "Test List",
		Description: "A test task list",
	}

	output := taskList.StringWithBackend(ncBackend)

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
	taskList := backend.TaskList{
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
	sbBackend := &sqlite.SQLiteBackend{
		Config: backend.BackendConfig{
			DBPath: "/very/long/path/to/database/that/exceeds/normal/width.db",
		},
	}

	taskList := backend.TaskList{
		ID:          "list-1",
		Name:        "A Very Long Task List Name That Should Be Truncated",
		Description: "With an even longer description that definitely won't fit",
	}

	// Use a narrow terminal width to force truncation
	output := taskList.StringWithWidthAndBackend(50, sbBackend)

	// Should still contain box drawing characters
	if !strings.Contains(output, "┌") || !strings.Contains(output, "┐") {
		t.Errorf("StringWithWidthAndBackend() should contain box drawing characters even when truncated, got: %q", output)
	}

	// Output should not be empty
	if len(output) == 0 {
		t.Errorf("StringWithWidthAndBackend() should not be empty")
	}
}
