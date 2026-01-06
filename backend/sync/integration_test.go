package sync

import (
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/backend/sqlite"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Integration tests that require a real Nextcloud server

// skipIfNoNextcloud skips the test if Nextcloud server is not available
func skipIfNoNextcloud(t *testing.T) backend.TaskManager {
	t.Helper()

	if os.Getenv("SKIP_INTEGRATION") == "1" {
		t.Skip("Skipping integration test (SKIP_INTEGRATION=1)")
	}

	url := os.Getenv("GOSYNCTASKS_NEXTCLOUD_HOST")
	config := backend.BackendConfig{
		Type:                "nextcloud",
		Enabled:             true,
		URL:                 url,
		AllowHTTP:           true,
		SuppressHTTPWarning: true,
		InsecureSkipVerify:  true,
		SuppressSSLWarning:  true,
	}

	backend, err := config.TaskManager()
	if err != nil {
		t.Skipf("Nextcloud server not available: %v", err)
	}

	// Try to list task lists to verify connection
	_, err = backend.GetTaskLists()
	if err != nil {
		t.Skipf("Cannot connect to Nextcloud: %v", err)
	}

	return backend
}

// cleanupNextcloudTasks removes all tasks from test lists
func cleanupNextcloudTasks(t *testing.T, backend backend.TaskManager, listID string) {
	t.Helper()

	tasks, err := backend.GetTasks(listID, nil)
	if err != nil {
		t.Logf("Warning: Failed to get tasks for cleanup: %v", err)
		return
	}

	for _, task := range tasks {
		if err := backend.DeleteTask(listID, task.UID); err != nil {
			t.Logf("Warning: Failed to delete task %s: %v", task.UID, err)
		}
	}
}

// TestSyncPushToNextcloud tests pushing local changes to Nextcloud
func TestSyncPushToNextcloud(t *testing.T) {
	remote := skipIfNoNextcloud(t)

	// Setup local SQLite backend
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	local, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create local backend: %v", err)
	}
	defer local.Close()

	// Get or create a test list
	lists, err := remote.GetTaskLists()
	if err != nil {
		t.Fatalf("Failed to get task lists: %v", err)
	}

	var testListID string
	for _, list := range lists {
		if list.Name == "IntegrationTest" {
			testListID = list.ID
			break
		}
	}

	if testListID == "" {
		testListID, err = remote.CreateTaskList("IntegrationTest", "Test list for integration tests", "#00ff00")
		if err != nil {
			t.Fatalf("Failed to create test list: %v", err)
		}
	}

	// Cleanup before and after test
	cleanupNextcloudTasks(t, remote, testListID)
	defer cleanupNextcloudTasks(t, remote, testListID)

	// Create sync manager
	sm := NewSyncManager(local, remote, ServerWins)

	// Create local task
	now := time.Now()
	localTask := backend.Task{
		UID:      "test-push-task-1",
		Summary:  "Test Push backend.Task",
		Status:   "TODO",
		Priority: 5,
		Created:  now,
		Modified: now,
	}

	_, err = local.AddTask(testListID, localTask)
	if err != nil {
		t.Fatalf("Failed to add local task: %v", err)
	}

	// Mark as locally modified
	err = local.MarkLocallyModified(localTask.UID)
	if err != nil {
		t.Fatalf("Failed to mark task as modified: %v", err)
	}

	// Perform sync (should push to remote)
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	t.Logf("Sync result: pushed=%d, pulled=%d, conflicts=%d",
		result.PushedTasks, result.PulledTasks, result.ConflictsResolved)

	if result.PushedTasks != 1 {
		t.Errorf("Expected 1 pushed task, got %d", result.PushedTasks)
	}

	// Verify task exists on remote
	remoteTasks, err := remote.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get remote tasks: %v", err)
	}

	found := false
	for _, task := range remoteTasks {
		if task.Summary == "Test Push backend.Task" {
			found = true
			if task.Priority != 5 {
				t.Errorf("Expected priority 5, got %d", task.Priority)
			}
			break
		}
	}

	if !found {
		t.Error("backend.Task not found on remote after push")
	}
}

// TestSyncPullFromNextcloud tests pulling changes from Nextcloud to local
func TestSyncPullFromNextcloud(t *testing.T) {
	remote := skipIfNoNextcloud(t)

	// Setup local SQLite backend
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	local, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create local backend: %v", err)
	}
	defer local.Close()

	// Get test list
	lists, err := remote.GetTaskLists()
	if err != nil {
		t.Fatalf("Failed to get task lists: %v", err)
	}

	var testListID string
	for _, list := range lists {
		if list.Name == "IntegrationTest" {
			testListID = list.ID
			break
		}
	}

	if testListID == "" {
		testListID, err = remote.CreateTaskList("IntegrationTest", "Test list", "#00ff00")
		if err != nil {
			t.Fatalf("Failed to create test list: %v", err)
		}
	}

	cleanupNextcloudTasks(t, remote, testListID)
	defer cleanupNextcloudTasks(t, remote, testListID)

	// Create task on remote
	now := time.Now()
	remoteTask := backend.Task{
		UID:      "test-pull-task-1",
		Summary:  "Test Pull backend.Task",
		Status:   "TODO",
		Priority: 3,
		Created:  now,
		Modified: now,
	}

	_, err = remote.AddTask(testListID, remoteTask)
	if err != nil {
		t.Fatalf("Failed to add remote task: %v", err)
	}

	// Wait for Nextcloud to propagate CTag changes
	time.Sleep(500 * time.Millisecond)

	// Create sync manager
	sm := NewSyncManager(local, remote, ServerWins)

	// Perform sync (should pull from remote)
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	t.Logf("Sync result: pushed=%d, pulled=%d, conflicts=%d",
		result.PushedTasks, result.PulledTasks, result.ConflictsResolved)

	if result.PulledTasks == 0 {
		t.Error("Expected at least 1 pulled task, got 0")
	}

	// Verify task exists in local
	localTasks, err := local.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get local tasks: %v", err)
	}

	found := false
	for _, task := range localTasks {
		if task.Summary == "Test Pull backend.Task" {
			found = true
			if task.Priority != 3 {
				t.Errorf("Expected priority 3, got %d", task.Priority)
			}
			break
		}
	}

	if !found {
		t.Error("backend.Task not found locally after pull")
	}
}

// TestSyncBidirectional tests bidirectional sync with both local and remote changes
func TestSyncBidirectional(t *testing.T) {
	remote := skipIfNoNextcloud(t)

	// Setup local SQLite backend
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	local, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create local backend: %v", err)
	}
	defer local.Close()

	// Get test list
	lists, err := remote.GetTaskLists()
	if err != nil {
		t.Fatalf("Failed to get task lists: %v", err)
	}

	var testListID string
	for _, list := range lists {
		if list.Name == "IntegrationTest" {
			testListID = list.ID
			break
		}
	}

	if testListID == "" {
		testListID, err = remote.CreateTaskList("IntegrationTest", "Test list", "#00ff00")
		if err != nil {
			t.Fatalf("Failed to create test list: %v", err)
		}
	}

	cleanupNextcloudTasks(t, remote, testListID)
	defer cleanupNextcloudTasks(t, remote, testListID)

	sm := NewSyncManager(local, remote, ServerWins)
	now := time.Now()

	// Create local task
	localTask := backend.Task{
		UID:      "bidirectional-local-1",
		Summary:  "Local backend.Task",
		Status:   "TODO",
		Priority: 1,
		Created:  now,
		Modified: now,
	}
	_, err = local.AddTask(testListID, localTask)
	if err != nil {
		t.Fatalf("Failed to add local task: %v", err)
	}
	err = local.MarkLocallyModified(localTask.UID)
	if err != nil {
		t.Fatalf("Failed to mark local task as modified: %v", err)
	}

	// Create remote task
	remoteTask := backend.Task{
		UID:      "bidirectional-remote-1",
		Summary:  "Remote backend.Task",
		Status:   "TODO",
		Priority: 2,
		Created:  now,
		Modified: now,
	}
	_, err = remote.AddTask(testListID, remoteTask)
	if err != nil {
		t.Fatalf("Failed to add remote task: %v", err)
	}

	// Wait for Nextcloud to propagate CTag changes
	time.Sleep(500 * time.Millisecond)

	// Perform bidirectional sync
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	t.Logf("Sync result: pushed=%d, pulled=%d, conflicts=%d",
		result.PushedTasks, result.PulledTasks, result.ConflictsResolved)

	// Verify both tasks exist locally
	localTasks, err := local.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get local tasks: %v", err)
	}

	if len(localTasks) != 2 {
		t.Errorf("Expected 2 local tasks, got %d", len(localTasks))
	}

	foundLocal := false
	foundRemote := false
	for _, task := range localTasks {
		if task.Summary == "Local backend.Task" {
			foundLocal = true
		}
		if task.Summary == "Remote backend.Task" {
			foundRemote = true
		}
	}

	if !foundLocal {
		t.Error("Local task not found after sync")
	}
	if !foundRemote {
		t.Error("Remote task not found locally after sync")
	}

	// Verify both tasks exist remotely
	remoteTasks, err := remote.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get remote tasks: %v", err)
	}

	if len(remoteTasks) != 2 {
		t.Errorf("Expected 2 remote tasks, got %d", len(remoteTasks))
	}
}

// TestSyncConflictResolution tests conflict resolution strategies
func TestSyncConflictResolution(t *testing.T) {
	remote := skipIfNoNextcloud(t)

	tests := []struct {
		name     string
		strategy ConflictResolutionStrategy
	}{
		{"ServerWins", ServerWins},
		{"LocalWins", LocalWins},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tmpDir := t.TempDir()
			dbPath := filepath.Join(tmpDir, "test.db")

			local, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
				Type:    "sqlite",
				Enabled: true,
				DBPath:  dbPath,
			})
			if err != nil {
				t.Fatalf("Failed to create local backend: %v", err)
			}
			defer local.Close()

			// Get test list
			lists, err := remote.GetTaskLists()
			if err != nil {
				t.Fatalf("Failed to get task lists: %v", err)
			}

			var testListID string
			for _, list := range lists {
				if list.Name == "IntegrationTest" {
					testListID = list.ID
					break
				}
			}

			if testListID == "" {
				testListID, err = remote.CreateTaskList("IntegrationTest", "Test list", "#00ff00")
				if err != nil {
					t.Fatalf("Failed to create test list: %v", err)
				}
			}

			cleanupNextcloudTasks(t, remote, testListID)

			sm := NewSyncManager(local, remote, tt.strategy)
			now := time.Now()

			// Create task on remote first
			taskUID := fmt.Sprintf("conflict-task-%s", tt.name)
			remoteTask := backend.Task{
				UID:         taskUID,
				Summary:     "Original backend.Task",
				Description: "Remote version",
				Status:      "TODO",
				Priority:    1,
				Created:     now,
				Modified:    now,
			}
			_, err = remote.AddTask(testListID, remoteTask)
			if err != nil {
				t.Fatalf("Failed to add remote task: %v", err)
			}

			// Wait for Nextcloud to propagate CTag changes
			time.Sleep(500 * time.Millisecond)

			// Pull it to local
			_, err = sm.Sync()
			if err != nil {
				t.Fatalf("Initial sync failed: %v", err)
			}

			// Modify locally
			localTask := backend.Task{
				UID:         taskUID,
				Summary:     "Modified Locally",
				Description: "Local version",
				Status:      "TODO",
				Priority:    5,
				Created:     now,
				Modified:    now.Add(1 * time.Minute),
			}
			localTask.UID = taskUID
			err = local.UpdateTask(testListID, localTask)
			if err != nil {
				t.Fatalf("Failed to update local task: %v", err)
			}
			err = local.MarkLocallyModified(taskUID)
			if err != nil {
				t.Fatalf("Failed to mark as modified: %v", err)
			}

			// Modify remotely (create conflict)
			remoteTask.Summary = "Modified Remotely"
			remoteTask.Description = "Remote version updated"
			remoteTask.Priority = 3
			remoteTask.Modified = now.Add(2 * time.Minute)
			remoteTask.UID = taskUID
			err = remote.UpdateTask(testListID, remoteTask)
			if err != nil {
				t.Fatalf("Failed to update remote task: %v", err)
			}

			// Wait for Nextcloud to propagate CTag changes
			// This prevents a race condition where the sync happens before
			// the CTag is updated, causing the list to be skipped
			time.Sleep(1 * time.Second)

			// Sync again - should resolve conflict
			result, err := sm.Sync()
			if err != nil {
				t.Fatalf("Conflict sync failed: %v", err)
			}

			t.Logf("Conflict resolution result: conflicts=%d", result.ConflictsResolved)

			// Verify resolution based on strategy
			finalTasks, err := local.GetTasks(testListID, nil)
			if err != nil {
				t.Fatalf("Failed to get final tasks: %v", err)
			}

			var finalTask *backend.Task
			for i := range finalTasks {
				if finalTasks[i].UID == taskUID {
					finalTask = &finalTasks[i]
					break
				}
			}

			if finalTask == nil {
				t.Fatal("backend.Task disappeared after conflict resolution")
			}

			switch tt.strategy {
			case ServerWins:
				if finalTask.Summary != "Modified Remotely" {
					t.Errorf("ServerWins: expected remote version, got %q", finalTask.Summary)
				}
			case LocalWins:
				if finalTask.Summary != "Modified Locally" {
					t.Errorf("LocalWins: expected local version, got %q", finalTask.Summary)
				}
			}
		})
	}
}

// TestSyncDeleteTask tests syncing task deletions
func TestSyncDeleteTask(t *testing.T) {
	remote := skipIfNoNextcloud(t)

	// Setup
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	local, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create local backend: %v", err)
	}
	defer local.Close()

	// Get test list
	lists, err := remote.GetTaskLists()
	if err != nil {
		t.Fatalf("Failed to get task lists: %v", err)
	}

	var testListID string
	for _, list := range lists {
		if list.Name == "IntegrationTest" {
			testListID = list.ID
			break
		}
	}

	if testListID == "" {
		testListID, err = remote.CreateTaskList("IntegrationTest", "Test list", "#00ff00")
		if err != nil {
			t.Fatalf("Failed to create test list: %v", err)
		}
	}

	cleanupNextcloudTasks(t, remote, testListID)
	defer cleanupNextcloudTasks(t, remote, testListID)

	sm := NewSyncManager(local, remote, ServerWins)
	now := time.Now()

	// Create and sync a task
	task := backend.Task{
		UID:      "delete-test-task",
		Summary:  "backend.Task to Delete",
		Status:   "TODO",
		Priority: 1,
		Created:  now,
		Modified: now,
	}
	_, err = local.AddTask(testListID, task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}
	err = local.MarkLocallyModified(task.UID)
	if err != nil {
		t.Fatalf("Failed to mark as modified: %v", err)
	}

	// Initial sync
	_, err = sm.Sync()
	if err != nil {
		t.Fatalf("Initial sync failed: %v", err)
	}

	// Delete locally
	err = local.DeleteTask(testListID, task.UID)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	// Sync deletion
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Delete sync failed: %v", err)
	}

	t.Logf("Delete sync result: pushed=%d, pulled=%d", result.PushedTasks, result.PulledTasks)

	// Verify task is deleted remotely
	remoteTasks, err := remote.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get remote tasks: %v", err)
	}

	for _, rt := range remoteTasks {
		if rt.UID == task.UID {
			t.Error("backend.Task still exists on remote after deletion sync")
		}
	}
}
