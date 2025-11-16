package cache

import (
	"encoding/json"
	"gosynctasks/backend"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// mockTaskManager implements backend.TaskManager for testing
type mockTaskManager struct {
	lists []backend.TaskList
	err   error
}

func (m *mockTaskManager) GetTaskLists() ([]backend.TaskList, error) {
	return m.lists, m.err
}

func (m *mockTaskManager) GetTasks(listID string, filter *backend.TaskFilter) ([]backend.Task, error) {
	return nil, nil
}

func (m *mockTaskManager) FindTasksBySummary(listID string, summary string) ([]backend.Task, error) {
	return nil, nil
}

func (m *mockTaskManager) AddTask(listID string, task backend.Task) error {
	return nil
}

func (m *mockTaskManager) UpdateTask(listID string, task backend.Task) error {
	return nil
}

func (m *mockTaskManager) DeleteTask(listID string, taskUID string) error {
	return nil
}

func (m *mockTaskManager) SortTasks(tasks []backend.Task) {
}

func (m *mockTaskManager) GetPriorityColor(priority int) string {
	return ""
}

func (m *mockTaskManager) ParseStatusFlag(status string) (string, error) {
	return status, nil
}

func (m *mockTaskManager) CreateTaskList(name, description, color string) (string, error) {
	return "new-list-id", nil
}

func (m *mockTaskManager) DeleteTaskList(listID string) error {
	return nil
}

func (m *mockTaskManager) RenameTaskList(listID, newName string) error {
	return nil
}

func (m *mockTaskManager) GetDeletedTaskLists() ([]backend.TaskList, error) {
	return nil, nil
}

func (m *mockTaskManager) RestoreTaskList(listID string) error {
	return nil
}

func (m *mockTaskManager) PermanentlyDeleteTaskList(listID string) error {
	return nil
}

func (m *mockTaskManager) StatusToDisplayName(backendStatus string) string {
	return backendStatus
}

func (m *mockTaskManager) GetBackendType() string {
	return "mock"
}

func (m *mockTaskManager) GetBackendDisplayName() string {
	return "[mock]"
}

func (m *mockTaskManager) GetBackendContext() string {
	return "mock-backend"
}

func (m *mockTaskManager) CanDetect() (bool, error) {
	return false, nil
}

func (m *mockTaskManager) GetDetectionInfo() string {
	return ""
}

// setupTestCache creates a temporary cache directory for testing
func setupTestCache(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "gosynctasks-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Override XDG_CACHE_HOME for testing
	oldCacheHome := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpDir)

	cleanup := func() {
		os.Setenv("XDG_CACHE_HOME", oldCacheHome)
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestGetCacheDir(t *testing.T) {
	tmpDir, cleanup := setupTestCache(t)
	defer cleanup()

	dir, err := GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir() failed: %v", err)
	}

	expected := filepath.Join(tmpDir, "gosynctasks")
	if dir != expected {
		t.Errorf("GetCacheDir() = %q, want %q", dir, expected)
	}

	// Verify directory was created
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Cache directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("Cache path is not a directory")
	}
}

func TestGetCacheFile(t *testing.T) {
	tmpDir, cleanup := setupTestCache(t)
	defer cleanup()

	file, err := GetCacheFile()
	if err != nil {
		t.Fatalf("GetCacheFile() failed: %v", err)
	}

	expectedDir := filepath.Join(tmpDir, "gosynctasks")
	expected := filepath.Join(expectedDir, "lists.json")
	if file != expected {
		t.Errorf("GetCacheFile() = %q, want %q", file, expected)
	}
}

func TestSaveAndLoadTaskListsCache(t *testing.T) {
	_, cleanup := setupTestCache(t)
	defer cleanup()

	// Create test data
	testLists := []backend.TaskList{
		{
			ID:          "list1",
			Name:        "Test List 1",
			Description: "First test list",
			Color:       "#ff0000",
		},
		{
			ID:          "list2",
			Name:        "Test List 2",
			Description: "Second test list",
			Color:       "#00ff00",
		},
	}

	// Save to cache
	err := SaveTaskListsToCache(testLists)
	if err != nil {
		t.Fatalf("SaveTaskListsToCache() failed: %v", err)
	}

	// Load from cache
	loaded, err := LoadTaskListsFromCache()
	if err != nil {
		t.Fatalf("LoadTaskListsFromCache() failed: %v", err)
	}

	// Verify loaded data matches
	if len(loaded) != len(testLists) {
		t.Fatalf("Loaded %d lists, want %d", len(loaded), len(testLists))
	}

	for i, list := range loaded {
		expected := testLists[i]
		if list.ID != expected.ID {
			t.Errorf("List[%d].ID = %q, want %q", i, list.ID, expected.ID)
		}
		if list.Name != expected.Name {
			t.Errorf("List[%d].Name = %q, want %q", i, list.Name, expected.Name)
		}
		if list.Description != expected.Description {
			t.Errorf("List[%d].Description = %q, want %q", i, list.Description, expected.Description)
		}
		if list.Color != expected.Color {
			t.Errorf("List[%d].Color = %q, want %q", i, list.Color, expected.Color)
		}
	}
}

func TestLoadTaskListsFromCache_NoFile(t *testing.T) {
	_, cleanup := setupTestCache(t)
	defer cleanup()

	// Try to load from non-existent cache
	_, err := LoadTaskListsFromCache()
	if err == nil {
		t.Error("LoadTaskListsFromCache() should fail when cache doesn't exist")
	}
}

func TestLoadTaskListsFromCache_InvalidJSON(t *testing.T) {
	_, cleanup := setupTestCache(t)
	defer cleanup()

	// Write invalid JSON to cache file
	cacheFile, err := GetCacheFile()
	if err != nil {
		t.Fatalf("GetCacheFile() failed: %v", err)
	}

	err = os.WriteFile(cacheFile, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Try to load invalid cache
	_, err = LoadTaskListsFromCache()
	if err == nil {
		t.Error("LoadTaskListsFromCache() should fail with invalid JSON")
	}
}

func TestCacheTimestamp(t *testing.T) {
	_, cleanup := setupTestCache(t)
	defer cleanup()

	testLists := []backend.TaskList{
		{ID: "list1", Name: "Test List"},
	}

	beforeSave := time.Now().Unix()
	err := SaveTaskListsToCache(testLists)
	if err != nil {
		t.Fatalf("SaveTaskListsToCache() failed: %v", err)
	}
	afterSave := time.Now().Unix()

	// Read cache file and check timestamp
	cacheFile, _ := GetCacheFile()
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}

	var cached CachedData
	if err := json.Unmarshal(data, &cached); err != nil {
		t.Fatalf("Failed to unmarshal cache: %v", err)
	}

	// Timestamp should be between beforeSave and afterSave
	if cached.Timestamp < beforeSave || cached.Timestamp > afterSave {
		t.Errorf("Timestamp %d not in range [%d, %d]", cached.Timestamp, beforeSave, afterSave)
	}
}

func TestLoadTaskListsWithFallback_UseCache(t *testing.T) {
	_, cleanup := setupTestCache(t)
	defer cleanup()

	// Create cached data
	cachedLists := []backend.TaskList{
		{ID: "cached1", Name: "Cached List"},
	}
	err := SaveTaskListsToCache(cachedLists)
	if err != nil {
		t.Fatalf("SaveTaskListsToCache() failed: %v", err)
	}

	// Create mock that should not be called (cache hit)
	remoteLists := []backend.TaskList{
		{ID: "remote1", Name: "Remote List"},
	}
	mock := &mockTaskManager{lists: remoteLists}

	// Load with fallback
	loaded, err := LoadTaskListsWithFallback(mock)
	if err != nil {
		t.Fatalf("LoadTaskListsWithFallback() failed: %v", err)
	}

	// Should return cached data, not remote
	if len(loaded) != 1 || loaded[0].ID != "cached1" {
		t.Errorf("Expected cached data, got: %+v", loaded)
	}
}

func TestLoadTaskListsWithFallback_FetchRemote(t *testing.T) {
	_, cleanup := setupTestCache(t)
	defer cleanup()

	// No cache exists - should fetch from remote
	remoteLists := []backend.TaskList{
		{ID: "remote1", Name: "Remote List"},
	}
	mock := &mockTaskManager{lists: remoteLists}

	loaded, err := LoadTaskListsWithFallback(mock)
	if err != nil {
		t.Fatalf("LoadTaskListsWithFallback() failed: %v", err)
	}

	// Should return remote data
	if len(loaded) != 1 || loaded[0].ID != "remote1" {
		t.Errorf("Expected remote data, got: %+v", loaded)
	}

	// Verify it was saved to cache
	cached, err := LoadTaskListsFromCache()
	if err != nil {
		t.Fatalf("Cache should exist after fallback: %v", err)
	}
	if len(cached) != 1 || cached[0].ID != "remote1" {
		t.Errorf("Expected cache to contain remote data, got: %+v", cached)
	}
}

func TestLoadTaskListsWithFallback_RemoteError(t *testing.T) {
	_, cleanup := setupTestCache(t)
	defer cleanup()

	// Mock returns error
	mock := &mockTaskManager{err: os.ErrNotExist}

	_, err := LoadTaskListsWithFallback(mock)
	if err == nil {
		t.Error("LoadTaskListsWithFallback() should return error when remote fails")
	}
}

func TestRefreshAndCacheTaskLists(t *testing.T) {
	_, cleanup := setupTestCache(t)
	defer cleanup()

	// Create cached data (old)
	oldLists := []backend.TaskList{
		{ID: "old1", Name: "Old List"},
	}
	err := SaveTaskListsToCache(oldLists)
	if err != nil {
		t.Fatalf("SaveTaskListsToCache() failed: %v", err)
	}

	// Create mock with new data
	newLists := []backend.TaskList{
		{ID: "new1", Name: "New List"},
	}
	mock := &mockTaskManager{lists: newLists}

	// Refresh cache
	loaded, err := RefreshAndCacheTaskLists(mock)
	if err != nil {
		t.Fatalf("RefreshAndCacheTaskLists() failed: %v", err)
	}

	// Should return new data
	if len(loaded) != 1 || loaded[0].ID != "new1" {
		t.Errorf("Expected new data, got: %+v", loaded)
	}

	// Verify cache was updated
	cached, err := LoadTaskListsFromCache()
	if err != nil {
		t.Fatalf("Failed to load cache: %v", err)
	}
	if len(cached) != 1 || cached[0].ID != "new1" {
		t.Errorf("Cache should contain new data, got: %+v", cached)
	}
}

func TestRefreshAndCacheTaskLists_Error(t *testing.T) {
	_, cleanup := setupTestCache(t)
	defer cleanup()

	// Mock returns error
	mock := &mockTaskManager{err: os.ErrPermission}

	_, err := RefreshAndCacheTaskLists(mock)
	if err == nil {
		t.Error("RefreshAndCacheTaskLists() should return error when remote fails")
	}
}

func TestEmptyListsCache(t *testing.T) {
	_, cleanup := setupTestCache(t)
	defer cleanup()

	// Save empty list
	err := SaveTaskListsToCache([]backend.TaskList{})
	if err != nil {
		t.Fatalf("SaveTaskListsToCache() failed: %v", err)
	}

	// Load empty list
	loaded, err := LoadTaskListsFromCache()
	if err != nil {
		t.Fatalf("LoadTaskListsFromCache() failed: %v", err)
	}

	if len(loaded) != 0 {
		t.Errorf("Expected empty list, got %d items", len(loaded))
	}
}
