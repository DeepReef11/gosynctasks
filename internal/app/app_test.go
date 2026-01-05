package app

import (
	"encoding/json"
	"gosynctasks/backend"
	"gosynctasks/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockTaskManagerForApp implements backend.TaskManager for app testing
type mockTaskManagerForApp struct {
	lists         []backend.TaskList
	err           error
	backendType   string
	canDetect     bool
	detectionInfo string
}

func (m *mockTaskManagerForApp) GetTaskLists() ([]backend.TaskList, error) {
	return m.lists, m.err
}

func (m *mockTaskManagerForApp) GetTasks(listID string, filter *backend.TaskFilter) ([]backend.Task, error) {
	return nil, nil
}

func (m *mockTaskManagerForApp) FindTasksBySummary(listID string, summary string) ([]backend.Task, error) {
	return nil, nil
}

func (m *mockTaskManagerForApp) AddTask(listID string, task backend.Task) (string, error) {
	return "mock-task-id", nil
}

func (m *mockTaskManagerForApp) UpdateTask(listID string, task backend.Task) error {
	return nil
}

func (m *mockTaskManagerForApp) DeleteTask(listID string, taskUID string) error {
	return nil
}

func (m *mockTaskManagerForApp) SortTasks(tasks []backend.Task) {
}

func (m *mockTaskManagerForApp) GetPriorityColor(priority int) string {
	return ""
}

func (m *mockTaskManagerForApp) ParseStatusFlag(status string) (string, error) {
	return status, nil
}

func (m *mockTaskManagerForApp) CreateTaskList(name, description, color string) (string, error) {
	return "new-list-id", nil
}

func (m *mockTaskManagerForApp) DeleteTaskList(listID string) error {
	return nil
}

func (m *mockTaskManagerForApp) RenameTaskList(listID, newName string) error {
	return nil
}

func (m *mockTaskManagerForApp) GetDeletedTaskLists() ([]backend.TaskList, error) {
	return nil, nil
}

func (m *mockTaskManagerForApp) RestoreTaskList(listID string) error {
	return nil
}

func (m *mockTaskManagerForApp) PermanentlyDeleteTaskList(listID string) error {
	return nil
}

func (m *mockTaskManagerForApp) StatusToDisplayName(backendStatus string) string {
	return backendStatus
}

func (m *mockTaskManagerForApp) GetBackendType() string {
	return m.backendType
}

func (m *mockTaskManagerForApp) GetBackendDisplayName() string {
	return "[mock]"
}

func (m *mockTaskManagerForApp) GetBackendContext() string {
	return "mock-backend"
}

func (m *mockTaskManagerForApp) CanDetect() (bool, error) {
	return m.canDetect, nil
}

func (m *mockTaskManagerForApp) GetDetectionInfo() string {
	return m.detectionInfo
}

// setupTestConfig creates a temporary config for testing
func setupTestConfig(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "gosynctasks-app-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Override XDG paths for testing
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	oldCacheHome := os.Getenv("XDG_CACHE_HOME")

	configDir := filepath.Join(tmpDir, "config")
	cacheDir := filepath.Join(tmpDir, "cache")

	os.Setenv("XDG_CONFIG_HOME", configDir)
	os.Setenv("XDG_CACHE_HOME", cacheDir)

	cleanup := func() {
		os.Setenv("XDG_CONFIG_HOME", oldConfigHome)
		os.Setenv("XDG_CACHE_HOME", oldCacheHome)
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// createTestConfigFile creates a test config file with the given backends
func createTestConfigFile(t *testing.T, backends map[string]backend.BackendConfig) {
	cfg := &config.Config{
		Backends:          backends,
		DefaultBackend:    "test",
		AutoDetectBackend: false,
		BackendPriority:   []string{"test"},
		UI:                "cli",
		CanWriteConfig:    true,
	}

	// Get config dir
	configHome := os.Getenv("XDG_CONFIG_HOME")
	configDir := filepath.Join(configHome, "gosynctasks")
	os.MkdirAll(configDir, 0755)

	configPath := filepath.Join(configDir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
}

func TestGetTaskLists(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	expectedLists := []backend.TaskList{
		{ID: "list1", Name: "Test List 1"},
		{ID: "list2", Name: "Test List 2"},
	}

	app := &App{
		taskLists: expectedLists,
	}

	lists := app.GetTaskLists()
	if len(lists) != len(expectedLists) {
		t.Fatalf("GetTaskLists() returned %d lists, want %d", len(lists), len(expectedLists))
	}

	for i, list := range lists {
		if list.ID != expectedLists[i].ID {
			t.Errorf("List[%d].ID = %q, want %q", i, list.ID, expectedLists[i].ID)
		}
	}
}

func TestGetTaskManager(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	mockManager := &mockTaskManagerForApp{backendType: "test"}
	app := &App{
		taskManager: mockManager,
	}

	manager := app.GetTaskManager()
	if manager == nil {
		t.Error("GetTaskManager() returned nil")
	}
}

func TestRefreshTaskLists(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	initialLists := []backend.TaskList{
		{ID: "old1", Name: "Old List"},
	}

	newLists := []backend.TaskList{
		{ID: "new1", Name: "New List"},
	}

	mockManager := &mockTaskManagerForApp{
		lists: newLists,
	}

	app := &App{
		taskLists:   initialLists,
		taskManager: mockManager,
	}

	// Before refresh
	if len(app.taskLists) != 1 || app.taskLists[0].ID != "old1" {
		t.Fatal("Initial lists not set correctly")
	}

	// Refresh
	err := app.RefreshTaskLists()
	if err != nil {
		t.Fatalf("RefreshTaskLists() failed: %v", err)
	}

	// After refresh
	if len(app.taskLists) != 1 || app.taskLists[0].ID != "new1" {
		t.Errorf("Lists not refreshed correctly, got: %+v", app.taskLists)
	}
}

func TestRefreshTaskLists_Error(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	mockManager := &mockTaskManagerForApp{
		err: os.ErrPermission,
	}

	app := &App{
		taskManager: mockManager,
	}

	err := app.RefreshTaskLists()
	if err == nil {
		t.Error("RefreshTaskLists() should return error when backend fails")
	}
}

func TestListBackends_NoBackends(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Create registry with empty backends map
	registry, err := backend.NewBackendRegistry(map[string]backend.BackendConfig{})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	app := &App{
		registry: registry,
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = app.ListBackends()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("ListBackends() failed: %v", err)
	}

	// Read output
	var output strings.Builder
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			output.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "No backends configured") {
		t.Errorf("Expected 'No backends configured', got: %q", outputStr)
	}
}

func TestDetectBackends_NoDetected(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Create registry with empty backends map
	registry, err := backend.NewBackendRegistry(map[string]backend.BackendConfig{})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}
	selector := backend.NewBackendSelector(registry)

	app := &App{
		registry: registry,
		selector: selector,
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = app.DetectBackends()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("DetectBackends() failed: %v", err)
	}

	// Read output
	var output strings.Builder
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			output.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "No backends detected") {
		t.Errorf("Expected 'No backends detected', got: %q", outputStr)
	}
}

func TestApp_BackendSelection(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	app := &App{
		selectedBackend: "testbackend",
	}

	if app.selectedBackend != "testbackend" {
		t.Errorf("selectedBackend = %q, want %q", app.selectedBackend, "testbackend")
	}
}

// Note: TestRun_* tests are omitted because they require complex mocking of cobra.Command.
// The Run() method's error handling logic is better tested through integration tests.
