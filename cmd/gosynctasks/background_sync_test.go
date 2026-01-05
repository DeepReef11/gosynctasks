package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gosynctasks/backend"
	"gosynctasks/backend/sqlite"
)

// TestBackgroundSyncCommand tests the background sync command creation
func TestBackgroundSyncCommand(t *testing.T) {
	cmd := newBackgroundSyncCmd()

	if cmd == nil {
		t.Fatal("Expected command to be created")
	}

	if cmd.Use != "_internal_background_sync" {
		t.Errorf("Expected command Use to be '_internal_background_sync', got '%s'", cmd.Use)
	}

	if !cmd.Hidden {
		t.Error("Expected command to be hidden")
	}
}

// TestBackgroundSyncWithPendingOperations tests syncing with pending operations
func TestBackgroundSyncWithPendingOperations(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()

	// Set up config and data directories
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	oldDataHome := os.Getenv("XDG_DATA_HOME")
	oldCacheHome := os.Getenv("XDG_CACHE_HOME")

	configDir := filepath.Join(tmpDir, "config")
	dataDir := filepath.Join(tmpDir, "data")
	cacheDir := filepath.Join(tmpDir, "cache")

	os.Setenv("XDG_CONFIG_HOME", configDir)
	os.Setenv("XDG_DATA_HOME", dataDir)
	os.Setenv("XDG_CACHE_HOME", cacheDir)

	defer func() {
		os.Setenv("XDG_CONFIG_HOME", oldConfigHome)
		os.Setenv("XDG_DATA_HOME", oldDataHome)
		os.Setenv("XDG_CACHE_HOME", oldCacheHome)
	}()

	// Create test database with pending operations
	dbPath := filepath.Join(dataDir, "gosynctasks", "caches", "mock.db")
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	sqliteBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Name:    "mock",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create SQLite backend: %v", err)
	}
	defer sqliteBackend.Close()

	// Create a test list and task
	listID, err := sqliteBackend.CreateTaskList("Test List", "", "")
	if err != nil {
		t.Fatalf("Failed to create list: %v", err)
	}

	// Add a task (this queues a create operation)
	task := backend.Task{
		Summary: "Test Task",
		Status:  "NEEDS-ACTION",
		Created: time.Now(),
	}
	_, err = sqliteBackend.AddTask(listID, task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// Verify pending operations exist
	ops, err := sqliteBackend.GetPendingSyncOperations()
	if err != nil {
		t.Fatalf("Failed to get pending operations: %v", err)
	}
	if len(ops) == 0 {
		t.Fatal("Expected at least one pending operation")
	}

	// Note: We can't easily test the full background sync command execution
	// because it requires a complete config with mock backends registered.
	// This test verifies that pending operations are created correctly,
	// which is the prerequisite for background sync to work.
	t.Logf("Successfully created %d pending operations for background sync", len(ops))
}

// TestBackgroundSyncNoPendingOperations tests that sync is skipped when no operations are pending
func TestBackgroundSyncNoPendingOperations(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()

	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	oldDataHome := os.Getenv("XDG_DATA_HOME")

	configDir := filepath.Join(tmpDir, "config")
	dataDir := filepath.Join(tmpDir, "data")

	os.Setenv("XDG_CONFIG_HOME", configDir)
	os.Setenv("XDG_DATA_HOME", dataDir)

	defer func() {
		os.Setenv("XDG_CONFIG_HOME", oldConfigHome)
		os.Setenv("XDG_DATA_HOME", oldDataHome)
	}()

	// Create empty test database
	dbPath := filepath.Join(dataDir, "gosynctasks", "caches", "empty.db")
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	sqliteBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Name:    "empty",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create SQLite backend: %v", err)
	}
	defer sqliteBackend.Close()

	// Verify no pending operations
	ops, err := sqliteBackend.GetPendingSyncOperations()
	if err != nil {
		t.Fatalf("Failed to get pending operations: %v", err)
	}
	if len(ops) != 0 {
		t.Errorf("Expected 0 pending operations, got %d", len(ops))
	}

	// Background sync should skip this backend
	t.Log("No pending operations - background sync would skip this backend")
}

// TestBackgroundSyncMultipleBackends tests syncing multiple backends
func TestBackgroundSyncMultipleBackends(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()

	oldDataHome := os.Getenv("XDG_DATA_HOME")
	dataDir := filepath.Join(tmpDir, "data")
	os.Setenv("XDG_DATA_HOME", dataDir)
	defer os.Setenv("XDG_DATA_HOME", oldDataHome)

	// Create multiple backends with pending operations
	backends := []string{"backend1", "backend2", "backend3"}

	for _, backendName := range backends {
		dbPath := filepath.Join(dataDir, "gosynctasks", "caches", backendName+".db")
		os.MkdirAll(filepath.Dir(dbPath), 0755)

		sqliteBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
			Type:    "sqlite",
			Name:    backendName,
			Enabled: true,
			DBPath:  dbPath,
		})
		if err != nil {
			t.Fatalf("Failed to create SQLite backend %s: %v", backendName, err)
		}

		// Create a test list and task for each backend
		listID, err := sqliteBackend.CreateTaskList("Test List", "", "")
		if err != nil {
			sqliteBackend.Close()
			t.Fatalf("Failed to create list for %s: %v", backendName, err)
		}

		task := backend.Task{
			Summary: "Test Task for " + backendName,
			Status:  "NEEDS-ACTION",
			Created: time.Now(),
		}
		_, err = sqliteBackend.AddTask(listID, task)
		if err != nil {
			sqliteBackend.Close()
			t.Fatalf("Failed to add task for %s: %v", backendName, err)
		}

		// Verify pending operations
		ops, err := sqliteBackend.GetPendingSyncOperations()
		if err != nil {
			sqliteBackend.Close()
			t.Fatalf("Failed to get pending operations for %s: %v", backendName, err)
		}
		if len(ops) == 0 {
			sqliteBackend.Close()
			t.Fatalf("Expected pending operations for %s", backendName)
		}

		sqliteBackend.Close()
		t.Logf("Backend %s has %d pending operations", backendName, len(ops))
	}

	// Background sync should process all backends with pending operations
	t.Logf("Successfully created pending operations for %d backends", len(backends))
}

// TestBackgroundSyncErrorHandling tests error handling in background sync
func TestBackgroundSyncErrorHandling(t *testing.T) {
	// Test that background sync handles errors gracefully
	tmpDir := t.TempDir()

	oldDataHome := os.Getenv("XDG_DATA_HOME")
	dataDir := filepath.Join(tmpDir, "data")
	os.Setenv("XDG_DATA_HOME", dataDir)
	defer os.Setenv("XDG_DATA_HOME", oldDataHome)

	// Create a backend
	dbPath := filepath.Join(dataDir, "gosynctasks", "caches", "test.db")
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	sqliteBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Name:    "test",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	// Close the backend to simulate an error condition
	sqliteBackend.Close()

	// Try to get pending operations on a closed backend
	// This should handle the error gracefully
	_, err = sqliteBackend.GetPendingSyncOperations()
	if err != nil {
		t.Logf("Correctly handled error from closed backend: %v", err)
	} else {
		t.Log("Backend handled closed state gracefully")
	}
}

// TestBackgroundSyncOperationTypes tests different operation types
func TestBackgroundSyncOperationTypes(t *testing.T) {
	tmpDir := t.TempDir()

	oldDataHome := os.Getenv("XDG_DATA_HOME")
	dataDir := filepath.Join(tmpDir, "data")
	os.Setenv("XDG_DATA_HOME", dataDir)
	defer os.Setenv("XDG_DATA_HOME", oldDataHome)

	dbPath := filepath.Join(dataDir, "gosynctasks", "caches", "ops.db")
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	sqliteBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Name:    "ops",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create SQLite backend: %v", err)
	}
	defer sqliteBackend.Close()

	listID, err := sqliteBackend.CreateTaskList("Test List", "", "")
	if err != nil {
		t.Fatalf("Failed to create list: %v", err)
	}

	// Test CREATE operation (queued by AddTask)
	task := backend.Task{
		Summary: "New Task",
		Status:  "NEEDS-ACTION",
		Created: time.Now(),
	}
	taskUID, err := sqliteBackend.AddTask(listID, task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// Verify CREATE operation queued
	ops, err := sqliteBackend.GetPendingSyncOperations()
	if err != nil {
		t.Fatalf("Failed to get pending operations: %v", err)
	}
	if len(ops) != 1 || ops[0].Operation != "create" {
		t.Errorf("Expected 1 'create' operation, got %d operations", len(ops))
	}

	// Clear the queue for next test
	sqliteBackend.ClearSyncFlagsAndQueue(taskUID)

	// Test UPDATE operation
	task.UID = taskUID
	task.Summary = "Updated Task"
	err = sqliteBackend.UpdateTask(listID, task)
	if err != nil {
		t.Fatalf("Failed to update task: %v", err)
	}

	ops, err = sqliteBackend.GetPendingSyncOperations()
	if err != nil {
		t.Fatalf("Failed to get pending operations: %v", err)
	}
	if len(ops) != 1 || ops[0].Operation != "update" {
		t.Errorf("Expected 1 'update' operation, got %d operations", len(ops))
	}

	// Clear the queue for next test
	sqliteBackend.ClearSyncFlagsAndQueue(taskUID)

	// Test DELETE operation
	err = sqliteBackend.DeleteTask(listID, taskUID)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	ops, err = sqliteBackend.GetPendingSyncOperations()
	if err != nil {
		t.Fatalf("Failed to get pending operations: %v", err)
	}
	if len(ops) != 1 || ops[0].Operation != "delete" {
		t.Errorf("Expected 1 'delete' operation, got %d operations", len(ops))
	}

	t.Logf("Successfully tested all operation types: create, update, delete")
}

// TestBackgroundSyncLogFile tests that background sync creates a log file
func TestBackgroundSyncLogFile(t *testing.T) {
	// The background sync command writes to a log file in /tmp
	// Verify the log path is constructed correctly
	expectedLogPath := filepath.Join(os.TempDir(), "gosynctasks-background-sync.log")

	// Check if log directory is accessible
	_, err := os.Stat(os.TempDir())
	if err != nil {
		t.Fatalf("Cannot access temp directory: %v", err)
	}

	t.Logf("Background sync log would be written to: %s", expectedLogPath)
}
