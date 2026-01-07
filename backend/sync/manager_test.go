package sync

import (
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/backend/sqlite"
	"path/filepath"
	"testing"
	"time"
)

// Helper to create test sync manager
func createTestSyncManager(t *testing.T, strategy ConflictResolutionStrategy) (*SyncManager, *sqlite.SQLiteBackend, *backend.MockBackend, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := backend.BackendConfig{
		Type:    "sqlite",
		Enabled: true,
		DBPath:  dbPath,
	}

	local, err := sqlite.NewSQLiteBackend(config)
	if err != nil {
		t.Fatalf("Failed to create local backend: %v", err)
	}

	remote := backend.NewMockBackend()
	sm := NewSyncManager(local, remote, strategy)

	cleanup := func() {
		local.Close()
	}

	return sm, local, remote, cleanup
}

// TestSyncManagerCreation tests creating a sync manager
func TestSyncManagerCreation(t *testing.T) {
	sm, _, _, cleanup := createTestSyncManager(t, ServerWins)
	defer cleanup()

	if sm == nil {
		t.Fatal("Expected sync manager to be created")
	}

	if sm.strategy != ServerWins {
		t.Errorf("Expected strategy ServerWins, got %s", sm.strategy)
	}
}

// TestPullNewTasks tests pulling new tasks from remote
func TestPullNewTasks(t *testing.T) {
	sm, local, remote, cleanup := createTestSyncManager(t, ServerWins)
	defer cleanup()

	// Create list on remote
	listID, _ := remote.CreateTaskList("Test List", "", "")
	remote.Lists[0].CTags = "ctag-123"

	// Add tasks to remote
	now := time.Now()
	remote.AddTask(listID, backend.Task{
		UID:      "task-1",
		Summary:  "Remote backend.Task 1",
		Status:   "NEEDS-ACTION",
		Priority: 5,
		Created:  now,
		Modified: now,
	})

	remote.AddTask(listID, backend.Task{
		UID:      "task-2",
		Summary:  "Remote backend.Task 2",
		Status:   "NEEDS-ACTION",
		Priority: 3,
		Created:  now,
		Modified: now,
	})

	// Sync
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.PulledTasks != 2 {
		t.Errorf("Expected 2 pulled tasks, got %d", result.PulledTasks)
	}

	// Verify tasks are in local database
	tasks, err := local.GetTasks(listID, nil)
	if err != nil {
		t.Fatalf("Failed to get local tasks: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 local tasks, got %d", len(tasks))
	}
}

// TestPullUpdatedTasks tests pulling updated tasks from remote
func TestPullUpdatedTasks(t *testing.T) {
	sm, local, remote, cleanup := createTestSyncManager(t, ServerWins)
	defer cleanup()

	// Create list on both local and remote
	listID, _ := local.CreateTaskList("Test List", "", "")
	remote.Lists = append(remote.Lists, backend.TaskList{
		ID:    listID,
		Name:  "Test List",
		CTags: "ctag-123",
	})
	remote.Tasks[listID] = []backend.Task{}

	// Add task to local (not modified)
	now := time.Now()
	task := backend.Task{
		UID:      "task-1",
		Summary:  "Original Summary",
		Status:   "NEEDS-ACTION",
		Priority: 5,
		Created:  now,
		Modified: now,
	}
	local.AddTask(listID, task)
	local.ClearSyncFlagsAndQueue("task-1") // Clear modification flag and queue entry

	// Update task on remote
	updated := now.Add(time.Hour)
	task.Summary = "Updated Summary"
	task.Priority = 1
	task.Modified = updated
	remote.AddTask(listID, task)

	// Change CTag to trigger sync
	remote.Lists[0].CTags = "ctag-456"

	// Sync
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.PulledTasks != 1 {
		t.Errorf("Expected 1 pulled task, got %d", result.PulledTasks)
	}

	// Verify local task is updated
	tasks, _ := local.GetTasks(listID, nil)
	if tasks[0].Summary != "Updated Summary" {
		t.Errorf("Expected summary 'Updated Summary', got '%s'", tasks[0].Summary)
	}
}

// TestConflictResolutionServerWins tests server_wins strategy
func TestConflictResolutionServerWins(t *testing.T) {
	sm, local, remote, cleanup := createTestSyncManager(t, ServerWins)
	defer cleanup()

	// Create list
	listID, _ := local.CreateTaskList("Test List", "", "")
	remote.Lists = append(remote.Lists, backend.TaskList{
		ID:    listID,
		Name:  "Test List",
		CTags: "ctag-123",
	})
	remote.Tasks[listID] = []backend.Task{}

	// Add task to both
	now := time.Now()
	task := backend.Task{
		Summary:  "Original",
		Status:   "NEEDS-ACTION",
		Priority: 5,
		Created:  now,
		Modified: now,
	}

	// Capture the actual UID assigned by SQLite
	taskUID, err := local.AddTask(listID, task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// Modify locally using the actual UID
	task.UID = taskUID
	task.Summary = "Local Modification"
	task.Priority = 1
	local.UpdateTask(listID, task)

	// Modify remotely with the same UID
	remoteTask := task
	remoteTask.Summary = "Remote Modification"
	remoteTask.Priority = 9
	remote.AddTask(listID, remoteTask)

	// Change CTag
	remote.Lists[0].CTags = "ctag-456"

	// Sync
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.ConflictsFound != 1 {
		t.Errorf("Expected 1 conflict, got %d", result.ConflictsFound)
	}

	if result.ConflictsResolved != 1 {
		t.Errorf("Expected 1 resolved conflict, got %d", result.ConflictsResolved)
	}

	// Verify server version won
	tasks, _ := local.GetTasks(listID, nil)
	if tasks[0].Summary != "Remote Modification" {
		t.Errorf("Expected remote summary, got '%s'", tasks[0].Summary)
	}
	if tasks[0].Priority != 9 {
		t.Errorf("Expected remote priority 9, got %d", tasks[0].Priority)
	}
}

// TestConflictResolutionLocalWins tests local_wins strategy
func TestConflictResolutionLocalWins(t *testing.T) {
	sm, local, remote, cleanup := createTestSyncManager(t, LocalWins)
	defer cleanup()

	// Create list
	listID, _ := local.CreateTaskList("Test List", "", "")
	remote.Lists = append(remote.Lists, backend.TaskList{
		ID:    listID,
		Name:  "Test List",
		CTags: "ctag-123",
	})
	remote.Tasks[listID] = []backend.Task{}

	// Add task to both
	now := time.Now()
	task := backend.Task{
		Summary:  "Original",
		Status:   "NEEDS-ACTION",
		Priority: 5,
		Created:  now,
		Modified: now,
	}

	// Capture the actual UID assigned by SQLite
	taskUID, err := local.AddTask(listID, task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// Modify locally using the actual UID
	task.UID = taskUID
	task.Summary = "Local Modification"
	task.Priority = 1
	local.UpdateTask(listID, task)

	// Modify remotely with the same UID
	remoteTask := task
	remoteTask.Summary = "Remote Modification"
	remoteTask.Priority = 9
	remote.AddTask(listID, remoteTask)

	// Change CTag
	remote.Lists[0].CTags = "ctag-456"

	// Sync (pull phase will detect conflict, push phase will send local version)
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.ConflictsFound != 1 {
		t.Errorf("Expected 1 conflict, got %d", result.ConflictsFound)
	}

	// Local version should be preserved and queued for push
	tasks, _ := local.GetTasks(listID, nil)
	if tasks[0].Summary != "Local Modification" {
		t.Errorf("Expected local summary, got '%s'", tasks[0].Summary)
	}
}

// TestConflictResolutionKeepBoth tests keep_both strategy
func TestConflictResolutionKeepBoth(t *testing.T) {
	sm, local, remote, cleanup := createTestSyncManager(t, KeepBoth)
	defer cleanup()

	// Create list
	listID, _ := local.CreateTaskList("Test List", "", "")
	remote.Lists = append(remote.Lists, backend.TaskList{
		ID:    listID,
		Name:  "Test List",
		CTags: "ctag-123",
	})
	remote.Tasks[listID] = []backend.Task{}

	// Add task to both
	now := time.Now()
	task := backend.Task{
		Summary:  "Original",
		Status:   "NEEDS-ACTION",
		Priority: 5,
		Created:  now,
		Modified: now,
	}

	// Capture the actual UID assigned by SQLite
	taskUID, err := local.AddTask(listID, task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// Modify locally using the actual UID
	task.UID = taskUID
	task.Summary = "Local Modification"
	local.UpdateTask(listID, task)

	// Modify remotely with the same UID
	remoteTask := task
	remoteTask.Summary = "Remote Modification"
	remote.AddTask(listID, remoteTask)

	// Change CTag
	remote.Lists[0].CTags = "ctag-456"

	// Sync
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.ConflictsFound != 1 {
		t.Errorf("Expected 1 conflict, got %d", result.ConflictsFound)
	}

	// Should have 2 tasks now (remote version + local copy)
	tasks, _ := local.GetTasks(listID, nil)
	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}

	// One should have "(local copy)" suffix
	foundCopy := false
	for _, t := range tasks {
		if t.Summary == "Local Modification (local copy)" {
			foundCopy = true
		}
	}
	if !foundCopy {
		t.Error("Expected to find task with '(local copy)' suffix")
	}
}

// TestPushCreateOperation tests pushing a create operation
func TestPushCreateOperation(t *testing.T) {
	sm, local, remote, cleanup := createTestSyncManager(t, ServerWins)
	defer cleanup()

	// Create list on both
	listID, _ := local.CreateTaskList("Test List", "", "")
	remote.Lists = append(remote.Lists, backend.TaskList{
		ID:    listID,
		Name:  "Test List",
		CTags: "ctag-123",
	})
	remote.Tasks[listID] = []backend.Task{}

	// Add task locally (this queues a create operation)
	now := time.Now()
	task := backend.Task{
		UID:      "task-1",
		Summary:  "New backend.Task",
		Status:   "NEEDS-ACTION",
		Priority: 5,
		Created:  now,
		Modified: now,
	}
	local.AddTask(listID, task)

	// Sync (should push the create)
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.PushedTasks != 1 {
		t.Errorf("Expected 1 pushed task, got %d", result.PushedTasks)
	}

	// Verify task is on remote
	remoteTasks, _ := remote.GetTasks(listID, nil)
	if len(remoteTasks) != 1 {
		t.Errorf("Expected 1 remote task, got %d", len(remoteTasks))
	}

	if remoteTasks[0].Summary != "New backend.Task" {
		t.Errorf("Expected summary 'New backend.Task', got '%s'", remoteTasks[0].Summary)
	}
}

// TestPushUpdateOperation tests pushing an update operation
func TestPushUpdateOperation(t *testing.T) {
	sm, local, remote, cleanup := createTestSyncManager(t, LocalWins)
	defer cleanup()

	// Create list on both
	listID, _ := local.CreateTaskList("Test List", "", "")
	remote.Lists = append(remote.Lists, backend.TaskList{
		ID:    listID,
		Name:  "Test List",
		CTags: "ctag-123",
	})
	remote.Tasks[listID] = []backend.Task{}

	// Add task to both
	now := time.Now()
	task := backend.Task{
		Summary:  "Original",
		Status:   "NEEDS-ACTION",
		Priority: 5,
		Created:  now,
		Modified: now,
	}

	// Capture the actual UID assigned by SQLite
	taskUID, err := local.AddTask(listID, task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	local.ClearSyncFlagsAndQueue(taskUID) // Clear create flag and queue entry

	// Add to remote with the same UID
	task.UID = taskUID
	remote.AddTask(listID, task)

	// Update locally
	task.Summary = "Updated Locally"
	local.UpdateTask(listID, task)

	// Sync (should push the update)
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.PushedTasks != 1 {
		t.Errorf("Expected 1 pushed task, got %d", result.PushedTasks)
	}

	// Verify remote is updated
	remoteTasks, _ := remote.GetTasks(listID, nil)
	if remoteTasks[0].Summary != "Updated Locally" {
		t.Errorf("Expected updated summary, got '%s'", remoteTasks[0].Summary)
	}
}

// TestPushDeleteOperation tests pushing a delete operation
func TestPushDeleteOperation(t *testing.T) {
	sm, local, remote, cleanup := createTestSyncManager(t, ServerWins)
	defer cleanup()

	// Create list on both
	listID, _ := local.CreateTaskList("Test List", "", "")
	remote.Lists = append(remote.Lists, backend.TaskList{
		ID:    listID,
		Name:  "Test List",
		CTags: "ctag-123",
	})
	remote.Tasks[listID] = []backend.Task{}

	// Add task to both
	now := time.Now()
	task := backend.Task{
		Summary:  "To Delete",
		Status:   "NEEDS-ACTION",
		Created:  now,
		Modified: now,
	}

	// Capture the actual UID assigned by SQLite
	taskUID, err := local.AddTask(listID, task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	local.ClearSyncFlagsAndQueue(taskUID) // Clear create flag and queue entry

	// Add to remote with the same UID
	task.UID = taskUID
	remote.AddTask(listID, task)

	// Delete locally using the actual UID
	local.DeleteTask(listID, taskUID)

	// Sync (should push the delete)
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.PushedTasks != 1 {
		t.Errorf("Expected 1 pushed operation, got %d", result.PushedTasks)
	}

	// Verify task is deleted on remote
	remoteTasks, _ := remote.GetTasks(listID, nil)
	if len(remoteTasks) != 0 {
		t.Errorf("Expected 0 remote tasks, got %d", len(remoteTasks))
	}
}

// TestFullSync tests full synchronization
func TestFullSync(t *testing.T) {
	sm, local, remote, cleanup := createTestSyncManager(t, ServerWins)
	defer cleanup()

	// Create list on remote
	listID, _ := remote.CreateTaskList("Test List", "", "")
	remote.Lists[0].CTags = "ctag-123"

	// Add tasks to remote
	now := time.Now()
	remote.AddTask(listID, backend.Task{
		UID:      "task-1",
		Summary:  "backend.Task 1",
		Status:   "NEEDS-ACTION",
		Created:  now,
		Modified: now,
	})

	// Initial sync
	_, _ = sm.Sync()

	// Add more tasks
	remote.AddTask(listID, backend.Task{
		UID:      "task-2",
		Summary:  "backend.Task 2",
		Status:   "NEEDS-ACTION",
		Created:  now,
		Modified: now,
	})

	// Don't change CTag (simulate CTag not updated)
	// Normal sync would skip, but full sync should pull anyway

	// Full sync
	_, err := sm.FullSync()
	if err != nil {
		t.Fatalf("Full sync failed: %v", err)
	}

	// Should pull the new task
	tasks, _ := local.GetTasks(listID, nil)
	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks after full sync, got %d", len(tasks))
	}
}

// TestRetryLogic tests retry with exponential backoff
func TestRetryLogic(t *testing.T) {
	sm, local, remote, cleanup := createTestSyncManager(t, ServerWins)
	defer cleanup()

	// Create list on both
	listID, _ := local.CreateTaskList("Test List", "", "")
	remote.Lists = append(remote.Lists, backend.TaskList{
		ID:    listID,
		Name:  "Test List",
		CTags: "ctag-123",
	})
	remote.Tasks[listID] = []backend.Task{}

	// Add task locally
	now := time.Now()
	task := backend.Task{
		UID:      "task-1",
		Summary:  "backend.Task",
		Status:   "NEEDS-ACTION",
		Created:  now,
		Modified: now,
	}
	local.AddTask(listID, task)

	// Make remote return error
	remote.AddTaskErr = fmt.Errorf("temporary error")

	// Sync (should fail and increment retry)
	_, _ = sm.Sync()

	// Check retry count
	ops, _ := local.GetPendingSyncOperations()
	if len(ops) != 1 {
		t.Fatalf("Expected 1 pending operation, got %d", len(ops))
	}

	if ops[0].RetryCount != 1 {
		t.Errorf("Expected retry count 1, got %d", ops[0].RetryCount)
	}

	if ops[0].LastError == "" {
		t.Error("Expected last error to be set")
	}

	// Clear error and sync again
	remote.AddTaskErr = nil
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Retry sync failed: %v", err)
	}

	if result.PushedTasks != 1 {
		t.Errorf("Expected 1 pushed task on retry, got %d", result.PushedTasks)
	}

	// Operation should be removed
	ops, _ = local.GetPendingSyncOperations()
	if len(ops) != 0 {
		t.Errorf("Expected 0 pending operations after success, got %d", len(ops))
	}
}

// TestSyncStats tests getting sync statistics
func TestSyncStats(t *testing.T) {
	sm, local, _, cleanup := createTestSyncManager(t, ServerWins)
	defer cleanup()

	// Create list and add tasks
	listID, _ := local.CreateTaskList("Test List", "", "")

	now := time.Now()
	local.AddTask(listID, backend.Task{UID: "task-1", Summary: "backend.Task 1", Status: "NEEDS-ACTION", Created: now, Modified: now})
	local.AddTask(listID, backend.Task{UID: "task-2", Summary: "backend.Task 2", Status: "NEEDS-ACTION", Created: now, Modified: now})

	// Get stats
	stats, err := sm.GetSyncStats()
	if err != nil {
		t.Fatalf("Failed to get sync stats: %v", err)
	}

	if stats.LocalTasks != 2 {
		t.Errorf("Expected 2 local tasks, got %d", stats.LocalTasks)
	}

	if stats.LocalLists != 1 {
		t.Errorf("Expected 1 local list, got %d", stats.LocalLists)
	}

	if stats.PendingOperations != 2 {
		t.Errorf("Expected 2 pending operations, got %d", stats.PendingOperations)
	}

	if stats.LocallyModified != 2 {
		t.Errorf("Expected 2 locally modified tasks, got %d", stats.LocallyModified)
	}
}

// TestSyncWithEmptyRemote tests sync when remote has no data
func TestSyncWithEmptyRemote(t *testing.T) {
	sm, local, _, cleanup := createTestSyncManager(t, ServerWins)
	defer cleanup()

	// Add tasks locally
	listID, _ := local.CreateTaskList("Test List", "", "")
	now := time.Now()
	local.AddTask(listID, backend.Task{UID: "task-1", Summary: "Local backend.Task", Status: "NEEDS-ACTION", Created: now, Modified: now})

	// Sync (should push to empty remote)
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.PushedTasks != 1 {
		t.Errorf("Expected 1 pushed task, got %d", result.PushedTasks)
	}
}

// TestSyncResult tests sync result structure
func TestSyncResult(t *testing.T) {
	sm, _, remote, cleanup := createTestSyncManager(t, ServerWins)
	defer cleanup()

	// Add remote data
	listID, _ := remote.CreateTaskList("Test List", "", "")
	remote.Lists[0].CTags = "ctag-123"

	now := time.Now()
	remote.AddTask(listID, backend.Task{UID: "task-1", Summary: "backend.Task 1", Status: "NEEDS-ACTION", Created: now, Modified: now})

	// Sync
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.Duration == 0 {
		t.Error("Expected duration to be set")
	}

	if result.PulledTasks != 1 {
		t.Errorf("Expected 1 pulled task, got %d", result.PulledTasks)
	}
}
