package sync_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gosynctasks/backend"
	_ "gosynctasks/backend/nextcloud" // Import for side-effects (registers backend)
	"gosynctasks/backend/sqlite"
	"gosynctasks/backend/sync"
	"gosynctasks/internal/config"
	"gosynctasks/internal/operations"

	"github.com/spf13/cobra"
)

// createTestNextcloudBackend creates a Nextcloud backend from environment variables
func createTestNextcloudBackend(t *testing.T) backend.TaskManager {
	t.Helper()

	if os.Getenv("SKIP_INTEGRATION") == "1" {
		t.Skip("Skipping integration test (SKIP_INTEGRATION=1)")
	}

	ncURL := os.Getenv("GOSYNCTASKS_NEXTCLOUD_HOST")
	ncUser := os.Getenv("GOSYNCTASKS_NEXTCLOUD_USERNAME")
	ncPass := os.Getenv("GOSYNCTASKS_NEXTCLOUD_PASSWORD")

	if ncURL == "" || ncUser == "" || ncPass == "" {
		t.Skip("GOSYNCTASKS_NEXTCLOUD_HOST, GOSYNCTASKS_NEXTCLOUD_USERNAME, and GOSYNCTASKS_NEXTCLOUD_PASSWORD must be set for integration tests")
	}

	config := backend.BackendConfig{
		Type:                "nextcloud",
		Enabled:             true,
		URL:                 fmt.Sprintf("nextcloud://%s:%s@%s", ncUser, ncPass, ncURL),
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

// cleanupTestList removes all tasks from a test list
func cleanupTestList(t *testing.T, taskManager backend.TaskManager, listID string) {
	t.Helper()

	tasks, err := taskManager.GetTasks(listID, nil)
	if err != nil {
		t.Logf("Warning: Failed to get tasks for cleanup: %v", err)
		return
	}

	for _, task := range tasks {
		if err := taskManager.DeleteTask(listID, task.UID); err != nil {
			t.Logf("Warning: Failed to delete task %s: %v", task.UID, err)
		}
	}
}

// TestSyncWorkflowWithCLIOperations tests end-to-end sync using CLI operations layer
func TestSyncWorkflowWithCLIOperations(t *testing.T) {
	// Setup remote backend
	remote := createTestNextcloudBackend(t)

	// Setup local SQLite backend (cache)
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

	// Create sync manager
	sm := sync.NewSyncManager(local, remote, sync.ServerWins)

	// Create test list on remote
	testListName := "CLI Integration Test " + time.Now().Format("20060102150405")
	testListID, err := remote.CreateTaskList(testListName, "CLI integration test list", "#00ff00")
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}

	// Cleanup list at the end
	defer func() {
		cleanupTestList(t, remote, testListID)
		if err := remote.DeleteTaskList(testListID); err != nil {
			t.Logf("Warning: Failed to delete test list: %v", err)
		}
	}()

	// Setup local list metadata for sync
	db, err := local.GetDB()
	if err != nil {
		t.Fatalf("Failed to get local DB: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO list_sync_metadata (list_id, backend_name, list_name, list_color, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, testListID, "nextcloud", testListName, "#00ff00", time.Now().Unix(), time.Now().Unix())
	if err != nil {
		t.Fatalf("Failed to insert list into local metadata: %v", err)
	}

	testList := backend.TaskList{
		ID:          testListID,
		Name:        testListName,
		Description: "CLI integration test list",
	}

	// Create mock sync provider
	mockSync := newMockSyncProvider()

	// Create test config
	cfg := &config.Config{
		DateFormat: "2006-01-02",
		Sync: &config.SyncConfig{
			Enabled:            true,
			AutoSync:           true,
			LocalBackend:       "sqlite",
			ConflictResolution: "server_wins",
		},
	}
	config.SetConfigForTest(cfg)

	// ===================================================================
	// TEST 1: Add task using CLI operations
	// ===================================================================
	t.Log("Test 1: Adding task via CLI operations...")

	taskName1 := "CLI Test Task 1 " + time.Now().Format("150405")

	capture := newOutputCapture()
	capture.start()

	cmd := newMockCommand()
	err = operations.HandleAddAction(
		cmd.Command,
		local,
		&testList,
		taskName1,
		mockSync,
	)

	_ = capture.stop()

	if err != nil {
		t.Fatalf("HandleAddAction failed: %v", err)
	}

	// Verify task was added to local
	localTasks, err := local.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get local tasks: %v", err)
	}

	if len(localTasks) != 1 {
		t.Errorf("Expected 1 local task, got %d", len(localTasks))
	}

	var addedTask *backend.Task
	for i := range localTasks {
		if localTasks[i].Summary == taskName1 {
			addedTask = &localTasks[i]
			break
		}
	}

	if addedTask == nil {
		t.Fatalf("Task not found in local after HandleAddAction")
	}

	t.Logf("Task added with UID: %s", addedTask.UID)

	// Check pending operations
	pendingOps, err := local.GetPendingSyncOperations()
	if err != nil {
		t.Fatalf("Failed to get pending ops: %v", err)
	}
	t.Logf("Test 1: Pending operations: %d", len(pendingOps))
	for i, op := range pendingOps {
		t.Logf("  Op %d: %s on task %s", i+1, op.Operation, op.TaskUID)
	}

	// ===================================================================
	// TEST 2: Sync to push task to remote
	// ===================================================================
	t.Log("Test 2: Syncing to push task to remote...")

	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	t.Logf("Sync result: pushed=%d, pulled=%d", result.PushedTasks, result.PulledTasks)

	if result.PushedTasks < 1 {
		t.Errorf("Expected at least 1 pushed task, got %d", result.PushedTasks)
	}

	// Verify task exists on remote
	time.Sleep(500 * time.Millisecond) // Give Nextcloud time to propagate
	remoteTasks, err := remote.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get remote tasks: %v", err)
	}

	foundOnRemote := false
	for _, task := range remoteTasks {
		if task.Summary == taskName1 {
			foundOnRemote = true
			break
		}
	}

	if !foundOnRemote {
		t.Errorf("Test task not found on remote after sync (total remote tasks: %d)", len(remoteTasks))
	}

	// ===================================================================
	// TEST 3: Add another task on remote directly
	// ===================================================================
	t.Log("Test 3: Adding task directly on remote...")

	taskName2 := "CLI Test Task 2 " + time.Now().Format("150405")
	remoteTask := backend.Task{
		Summary:  taskName2,
		Status:   "NEEDS-ACTION",
		Priority: 3,
		Created:  time.Now(),
		Modified: time.Now(),
	}

	_, err = remote.AddTask(testListID, remoteTask)
	if err != nil {
		t.Fatalf("Failed to add remote task: %v", err)
	}

	// ===================================================================
	// TEST 4: Sync to pull task from remote
	// ===================================================================
	t.Log("Test 4: Syncing to pull task from remote...")

	time.Sleep(500 * time.Millisecond) // Give Nextcloud time to propagate CTag
	result, err = sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	t.Logf("Sync result: pushed=%d, pulled=%d", result.PushedTasks, result.PulledTasks)

	if result.PulledTasks == 0 {
		t.Error("Expected at least 1 pulled task, got 0")
	}

	// Verify both test tasks exist locally
	localTasks, err = local.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get local tasks: %v", err)
	}

	localTask1Found := false
	localTask2Found := false
	for _, task := range localTasks {
		if task.Summary == taskName1 {
			localTask1Found = true
		}
		if task.Summary == taskName2 {
			localTask2Found = true
		}
	}

	if !localTask1Found || !localTask2Found {
		t.Errorf("Not all test tasks found locally (task1=%v, task2=%v, total=%d)",
			localTask1Found, localTask2Found, len(localTasks))
	}

	// ===================================================================
	// TEST 5: Complete task using CLI operations
	// ===================================================================
	t.Log("Test 5: Completing task via CLI operations...")

	capture = newOutputCapture()
	capture.start()

	completeCmd := newMockCommand()
	err = operations.HandleCompleteAction(
		completeCmd.Command,
		local,
		cfg,
		&testList,
		taskName1,
		mockSync,
	)

	_ = capture.stop()

	if err != nil {
		t.Fatalf("HandleCompleteAction failed: %v", err)
	}

	// Verify task is marked as done locally
	localTasks, err = local.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get local tasks: %v", err)
	}

	completed := false
	for _, task := range localTasks {
		if task.Summary == taskName1 {
			if task.Status == "DONE" || task.Status == "COMPLETED" {
				completed = true
			}
			break
		}
	}

	if !completed {
		t.Error("Task not marked as completed locally")
	}

	// ===================================================================
	// TEST 6: Sync to push completion to remote
	// ===================================================================
	t.Log("Test 6: Syncing completion to remote...")

	result, err = sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	t.Logf("Sync result: pushed=%d, pulled=%d", result.PushedTasks, result.PulledTasks)

	if result.PushedTasks < 1 {
		t.Errorf("Expected at least 1 pushed task (completion), got %d", result.PushedTasks)
	}

	// ===================================================================
	// TEST 7: Update task using CLI operations
	// ===================================================================
	t.Log("Test 7: Updating task via CLI operations...")

	capture = newOutputCapture()
	capture.start()

	updateCmd := newMockCommand()
	updateCmd.withFlag("priority", 9)
	// Manually mark the priority flag as changed since Set() might not do it
	updateCmd.Command.Flags().Lookup("priority").Changed = true
	err = operations.HandleUpdateAction(
		updateCmd.Command,
		local,
		cfg,
		&testList,
		taskName2,
		mockSync,
	)

	_ = capture.stop()

	if err != nil {
		t.Fatalf("HandleUpdateAction failed: %v", err)
	}

	// Verify task was updated locally
	localTasks, err = local.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get local tasks: %v", err)
	}

	t.Logf("Test 7: Local tasks after update: %d", len(localTasks))
	for i, task := range localTasks {
		if task.Summary == taskName2 || task.Summary == taskName1 {
			t.Logf("  Task %d: %s (Priority: %d, Status: %s)", i+1, task.Summary, task.Priority, task.Status)
		}
	}

	updated := false
	var actualPriority int
	for _, task := range localTasks {
		if task.Summary == taskName2 {
			actualPriority = task.Priority
			if task.Priority == 9 {
				updated = true
			}
			break
		}
	}

	if !updated {
		t.Errorf("Task not updated locally (expected priority 9, got %d)", actualPriority)
		// Check pending operations to see if update was queued
		pendingOps, _ := local.GetPendingSyncOperations()
		t.Logf("Test 7: Pending operations after update: %d", len(pendingOps))
		for i, op := range pendingOps {
			t.Logf("  Op %d: %s on task %s", i+1, op.Operation, op.TaskUID)
		}
	}

	// ===================================================================
	// TEST 8: Final sync to verify everything
	// ===================================================================
	t.Log("Test 8: Final sync to verify all changes...")

	result, err = sm.Sync()
	if err != nil {
		t.Fatalf("Final sync failed: %v", err)
	}

	t.Logf("Final sync result: pushed=%d, pulled=%d", result.PushedTasks, result.PulledTasks)

	t.Log("✓ All CLI operation tests passed")
}

// TestSyncNoAccidentalDeletionOnDBReset tests that re-initializing the local DB
// doesn't cause tasks to be deleted from remote
func TestSyncNoAccidentalDeletionOnDBReset(t *testing.T) {
	// Setup remote backend
	remote := createTestNextcloudBackend(t)

	// ===================================================================
	// PHASE 1: Initial setup - create list and tasks
	// ===================================================================
	t.Log("Phase 1: Creating test list and tasks...")

	testListName := "DB Reset Test " + time.Now().Format("20060102150405")
	testListID, err := remote.CreateTaskList(testListName, "DB reset test list", "#ff0000")
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}

	// Cleanup list at the end
	defer func() {
		cleanupTestList(t, remote, testListID)
		if err := remote.DeleteTaskList(testListID); err != nil {
			t.Logf("Warning: Failed to delete test list: %v", err)
		}
	}()

	// Create first local backend
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	local1, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create first local backend: %v", err)
	}

	// Create sync manager for first local
	sm1 := sync.NewSyncManager(local1, remote, sync.ServerWins)

	// Setup local list metadata
	db, err := local1.GetDB()
	if err != nil {
		t.Fatalf("Failed to get local DB: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO list_sync_metadata (list_id, backend_name, list_name, list_color, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, testListID, "nextcloud", testListName, "#ff0000", time.Now().Unix(), time.Now().Unix())
	if err != nil {
		t.Fatalf("Failed to insert list into local metadata: %v", err)
	}

	testList := backend.TaskList{
		ID:          testListID,
		Name:        testListName,
		Description: "DB reset test list",
	}

	// Create mock sync provider and config
	mockSync := newMockSyncProvider()
	cfg := &config.Config{
		DateFormat: "2006-01-02",
		Sync: &config.SyncConfig{
			Enabled:            true,
			AutoSync:           true,
			LocalBackend:       "sqlite",
			ConflictResolution: "server_wins",
		},
	}
	config.SetConfigForTest(cfg)

	// Add tasks using CLI operations
	taskName1 := "Persistent Task 1 " + time.Now().Format("150405")
	taskName2 := "Persistent Task 2 " + time.Now().Format("150405")

	capture := newOutputCapture()
	capture.start()

	cmd := newMockCommand()
	err = operations.HandleAddAction(cmd.Command, local1, &testList, taskName1, mockSync)
	if err != nil {
		capture.stop()
		t.Fatalf("HandleAddAction failed for task 1: %v", err)
	}

	cmd = newMockCommand()
	err = operations.HandleAddAction(cmd.Command, local1, &testList, taskName2, mockSync)
	if err != nil {
		capture.stop()
		t.Fatalf("HandleAddAction failed for task 2: %v", err)
	}

	_ = capture.stop()

	t.Log("Phase 1: Added 2 tasks locally")

	// Verify both tasks are in local DB before sync
	localTasksBeforeSync, err := local1.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get local tasks before sync: %v", err)
	}
	t.Logf("Phase 1: Local tasks before sync: %d", len(localTasksBeforeSync))
	for i, task := range localTasksBeforeSync {
		t.Logf("  Task %d: %s (UID: %s, Status: %s)", i+1, task.Summary, task.UID, task.Status)
	}

	// Check pending operations before sync
	pendingOps, err := local1.GetPendingSyncOperations()
	if err != nil {
		t.Fatalf("Failed to get pending operations: %v", err)
	}
	t.Logf("Phase 1: Pending operations: %d", len(pendingOps))
	for i, op := range pendingOps {
		t.Logf("  Op %d: %s on task %s (list %s, retry: %d)", i+1, op.Operation, op.TaskUID, op.ListID, op.RetryCount)
	}

	// ===================================================================
	// PHASE 2: First sync - push tasks to remote
	// ===================================================================
	t.Log("Phase 2: First sync to push tasks to remote...")

	result, err := sm1.Sync()
	if err != nil {
		t.Fatalf("First sync failed: %v", err)
	}

	t.Logf("First sync result: pushed=%d, pulled=%d", result.PushedTasks, result.PulledTasks)

	// Check pending operations after sync to see if any failed
	pendingOpsAfterSync, err := local1.GetPendingSyncOperations()
	if err != nil {
		t.Fatalf("Failed to get pending operations after sync: %v", err)
	}
	t.Logf("Phase 2: Pending operations after sync: %d", len(pendingOpsAfterSync))
	for i, op := range pendingOpsAfterSync {
		t.Logf("  Op %d: %s on task %s (retry: %d, error: %s)",
			i+1, op.Operation, op.TaskUID, op.RetryCount, op.LastError)
	}

	if result.PushedTasks < 2 {
		t.Errorf("Expected at least 2 pushed tasks, got %d", result.PushedTasks)
	}

	// Verify tasks exist on remote
	time.Sleep(500 * time.Millisecond)
	remoteTasks, err := remote.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get remote tasks: %v", err)
	}

	// Verify our 2 test tasks are present (there may be other tasks too)
	foundTask1 := false
	foundTask2 := false
	for _, task := range remoteTasks {
		if task.Summary == taskName1 {
			foundTask1 = true
		}
		if task.Summary == taskName2 {
			foundTask2 = true
		}
	}

	if !foundTask1 || !foundTask2 {
		t.Fatalf("Not all test tasks found on remote after first sync (found task1=%v, task2=%v, total tasks=%d)",
			foundTask1, foundTask2, len(remoteTasks))
	}

	t.Logf("Phase 2: Verified both test tasks on remote (total tasks: %d)", len(remoteTasks))

	// Close first local backend
	local1.Close()

	// ===================================================================
	// PHASE 3: Delete local DB and recreate (simulating fresh start)
	// ===================================================================
	t.Log("Phase 3: Deleting local DB and recreating...")

	if err := os.Remove(dbPath); err != nil {
		t.Fatalf("Failed to delete local DB: %v", err)
	}

	t.Log("Phase 3: Local DB deleted")

	// Create second local backend (fresh database)
	local2, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create second local backend: %v", err)
	}
	defer local2.Close()

	// Create sync manager for second local
	sm2 := sync.NewSyncManager(local2, remote, sync.ServerWins)

	// Setup local list metadata again
	db, err = local2.GetDB()
	if err != nil {
		t.Fatalf("Failed to get local DB: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO list_sync_metadata (list_id, backend_name, list_name, list_color, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, testListID, "nextcloud", testListName, "#ff0000", time.Now().Unix(), time.Now().Unix())
	if err != nil {
		t.Fatalf("Failed to insert list into second local metadata: %v", err)
	}

	t.Log("Phase 3: Fresh local DB created")

	// ===================================================================
	// PHASE 4: Second sync - should PULL tasks, not DELETE them
	// ===================================================================
	t.Log("Phase 4: Second sync after DB reset - should pull tasks from remote...")

	result, err = sm2.Sync()
	if err != nil {
		t.Fatalf("Second sync failed: %v", err)
	}

	t.Logf("Second sync result: pushed=%d, pulled=%d", result.PushedTasks, result.PulledTasks)

	if result.PulledTasks < 2 {
		t.Errorf("Expected at least 2 pulled tasks after DB reset, got %d", result.PulledTasks)
	}

	// CRITICAL: Verify tasks still exist on remote
	time.Sleep(500 * time.Millisecond)
	remoteTasks, err = remote.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get remote tasks: %v", err)
	}

	// Check that our test tasks are still on remote (the critical check)
	remoteTask1Found := false
	remoteTask2Found := false
	for _, task := range remoteTasks {
		if task.Summary == taskName1 {
			remoteTask1Found = true
		}
		if task.Summary == taskName2 {
			remoteTask2Found = true
		}
	}

	if !remoteTask1Found || !remoteTask2Found {
		t.Errorf("CRITICAL: Test tasks were accidentally deleted from remote! (found task1=%v, task2=%v, total=%d)",
			remoteTask1Found, remoteTask2Found, len(remoteTasks))
	}

	t.Logf("Phase 4: ✓ Both test tasks still on remote (total: %d)", len(remoteTasks))

	// Verify both tasks exist locally
	localTasks, err := local2.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get local tasks: %v", err)
	}

	found1 := false
	found2 := false
	for _, task := range localTasks {
		if task.Summary == taskName1 {
			found1 = true
		}
		if task.Summary == taskName2 {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Errorf("Not all test tasks were pulled from remote after DB reset (found task1=%v, task2=%v, total=%d)",
			found1, found2, len(localTasks))
	}

	t.Logf("Phase 4: ✓ Both test tasks pulled to local cache (total: %d)", len(localTasks))

	t.Log("Phase 4: ✓ Tasks correctly pulled from remote, no accidental deletion")

	// ===================================================================
	// PHASE 5: Add a new task and verify sync still works
	// ===================================================================
	t.Log("Phase 5: Adding new task after DB reset...")

	taskName3 := "Post-Reset Task " + time.Now().Format("150405")

	capture = newOutputCapture()
	capture.start()

	cmd = newMockCommand()
	err = operations.HandleAddAction(cmd.Command, local2, &testList, taskName3, mockSync)

	_ = capture.stop()

	if err != nil {
		t.Fatalf("HandleAddAction failed for post-reset task: %v", err)
	}

	// Sync new task
	result, err = sm2.Sync()
	if err != nil {
		t.Fatalf("Sync after adding post-reset task failed: %v", err)
	}

	t.Logf("Post-reset sync result: pushed=%d, pulled=%d", result.PushedTasks, result.PulledTasks)

	if result.PushedTasks != 1 {
		t.Errorf("Expected 1 pushed task (post-reset), got %d", result.PushedTasks)
	}

	// Verify all 3 test tasks exist on remote
	time.Sleep(500 * time.Millisecond)
	remoteTasks, err = remote.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get remote tasks: %v", err)
	}

	// Check that all 3 of our test tasks are on remote
	remoteTask1Found = false
	remoteTask2Found = false
	remoteTask3Found := false
	for _, task := range remoteTasks {
		if task.Summary == taskName1 {
			remoteTask1Found = true
		}
		if task.Summary == taskName2 {
			remoteTask2Found = true
		}
		if task.Summary == taskName3 {
			remoteTask3Found = true
		}
	}

	if !remoteTask1Found || !remoteTask2Found || !remoteTask3Found {
		t.Errorf("Not all test tasks on remote after post-reset add (task1=%v, task2=%v, task3=%v, total=%d)",
			remoteTask1Found, remoteTask2Found, remoteTask3Found, len(remoteTasks))
	}

	t.Logf("Phase 5: ✓ All 3 test tasks on remote (total: %d)", len(remoteTasks))
	t.Log("✓ All DB reset tests passed - no accidental deletions detected")
}

// ============================================================================
// Test Helpers
// ============================================================================

// mockCommand creates a cobra.Command with specified flags for testing
type mockCommand struct {
	*cobra.Command
	flags map[string]interface{}
}

// newMockCommand creates a mock cobra command with common flags used in operations
func newMockCommand() *mockCommand {
	cmd := &cobra.Command{
		Use: "test",
	}

	// Add all flags that operations might use
	cmd.Flags().String("description", "", "Task description")
	cmd.Flags().Int("priority", 0, "Task priority")
	cmd.Flags().String("add-status", "", "Status for add action")
	cmd.Flags().StringArray("status", []string{}, "Status for update/complete")
	cmd.Flags().String("due-date", "", "Due date")
	cmd.Flags().String("start-date", "", "Start date")
	cmd.Flags().String("parent", "", "Parent task")
	cmd.Flags().Bool("literal", false, "Literal mode")
	cmd.Flags().String("summary", "", "Task summary")
	cmd.Flags().String("view", "", "View name")

	return &mockCommand{
		Command: cmd,
		flags:   make(map[string]interface{}),
	}
}

// withFlag sets a flag value on the mock command
func (m *mockCommand) withFlag(name string, value interface{}) *mockCommand {
	switch v := value.(type) {
	case string:
		_ = m.Command.Flags().Set(name, v)
	case int:
		_ = m.Command.Flags().Set(name, fmt.Sprintf("%d", v))
	case bool:
		if v {
			_ = m.Command.Flags().Set(name, "true")
		}
	case []string:
		for _, s := range v {
			_ = m.Command.Flags().Set(name, s)
		}
	}
	return m
}

// mockSyncProvider is a test sync provider that doesn't spawn background processes
// It works by returning nil from GetSyncCoordinator(), which prevents triggerPushSync
// from spawning background sync processes during tests
type mockSyncProvider struct {
	syncCoordinator interface{}
}

func newMockSyncProvider() *mockSyncProvider {
	return &mockSyncProvider{}
}

func (m *mockSyncProvider) GetSyncCoordinator() interface{} {
	return m.syncCoordinator // Always nil - prevents background sync spawning
}

// outputCapture captures stdout/stderr during test execution
type outputCapture struct {
	oldStdout *os.File
	oldStderr *os.File
	r         *os.File
	w         *os.File
	outC      chan string
}

// newOutputCapture creates a new output capture
func newOutputCapture() *outputCapture {
	return &outputCapture{}
}

// start begins capturing output
func (o *outputCapture) start() {
	o.oldStdout = os.Stdout
	o.oldStderr = os.Stderr
	o.r, o.w, _ = os.Pipe()
	os.Stdout = o.w
	os.Stderr = o.w

	o.outC = make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, o.r)
		o.outC <- buf.String()
	}()
}

// stop stops capturing and returns the captured output
func (o *outputCapture) stop() string {
	_ = o.w.Close()
	os.Stdout = o.oldStdout
	os.Stderr = o.oldStderr
	out := <-o.outC
	return out
}
