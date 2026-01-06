package nextcloud

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gosynctasks/backend"
	"gosynctasks/backend/sqlite"
	"gosynctasks/backend/sync"
	"gosynctasks/internal/config"
	"gosynctasks/internal/operations"
)

// createNextcloudBackend creates a Nextcloud backend from environment variables
func createNextcloudBackend(t *testing.T) (*NextcloudBackend, error) {
	ncURL := os.Getenv("GOSYNCTASKS_NEXTCLOUD_HOST")
	ncUser := os.Getenv("GOSYNCTASKS_NEXTCLOUD_USERNAME")
	ncPass := os.Getenv("GOSYNCTASKS_NEXTCLOUD_PASSWORD")

	// Construct URL with credentials
	u, err := url.Parse(fmt.Sprintf("nextcloud://%s:%s@%s", ncUser, ncPass, ncURL))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Nextcloud URL: %w", err)
	}

	// Create connector config
	connConfig := backend.ConnectorConfig{
		URL:                 u,
		InsecureSkipVerify:  true, // For test servers with self-signed certs
		SuppressSSLWarning:  true,
		AllowHTTP:           true, // For test servers on HTTP
		SuppressHTTPWarning: true,
	}

	// NewNextcloudBackend returns backend.TaskManager interface
	tm, err := NewNextcloudBackend(connConfig)
	if err != nil {
		return nil, err
	}

	// Type assert to concrete NextcloudBackend type
	nb, ok := tm.(*NextcloudBackend)
	if !ok {
		return nil, fmt.Errorf("failed to create NextcloudBackend: wrong type")
	}

	return nb, nil
}

// TestNextcloudDirectOperations tests Nextcloud backend operations without sync
func TestNextcloudDirectOperationsIntegration(t *testing.T) {
	// Check for required environment variables
	ncURL := os.Getenv("GOSYNCTASKS_NEXTCLOUD_HOST")
	ncUser := os.Getenv("GOSYNCTASKS_NEXTCLOUD_USERNAME")
	ncPass := os.Getenv("GOSYNCTASKS_NEXTCLOUD_PASSWORD")

	if ncURL == "" || ncUser == "" || ncPass == "" {
		t.Skip("GOSYNCTASKS_NEXTCLOUD_HOST, GOSYNCTASKS_NEXTCLOUD_USERNAME, and GOSYNCTASKS_NEXTCLOUD_PASSWORD must be set for Nextcloud integration tests")
	}

	// Create Nextcloud backend
	nb, err := createNextcloudBackend(t)
	if err != nil {
		t.Fatalf("Failed to create Nextcloud backend: %v", err)
	}

	// ============================================================
	// TEST 0: Create test calendar (tests CreateTaskList like CLI does)
	// ============================================================
	testCalendarName := "GoSyncTasks Direct Test " + time.Now().Format("20060102150405")
	t.Logf("Test 0: Creating test calendar via CreateTaskList: %s", testCalendarName)

	// This is exactly how the CLI creates calendars (cmd/gosynctasks/list.go:93)
	testCalendarID, err := nb.CreateTaskList(testCalendarName, "Direct operations test calendar", "")
	if err != nil {
		t.Fatalf("Failed to create test calendar: %v", err)
	}
	t.Logf("✓ Calendar created with ID: %s", testCalendarID)

	// Verify the calendar was actually created by fetching all calendars
	calendars, err := nb.GetTaskLists()
	if err != nil {
		t.Fatalf("Failed to get task lists: %v", err)
	}

	calendarFound := false
	for _, cal := range calendars {
		if cal.ID == testCalendarID && cal.Name == testCalendarName {
			calendarFound = true
			break
		}
	}
	if !calendarFound {
		t.Fatalf("Created calendar not found in task lists")
	}
	t.Logf("✓ Calendar verified in task lists")

	// Ensure cleanup: delete the test calendar when done
	// Track if calendar was created to ensure we always try to delete it
	calendarCreated := true
	defer func() {
		if !calendarCreated {
			return
		}

		// ============================================================
		// CLEANUP: Delete test calendar (tests DeleteTaskList like CLI does)
		// ============================================================
		t.Log("Cleanup: Deleting test calendar via DeleteTaskList...")

		// This is exactly how the CLI deletes calendars (cmd/gosynctasks/list.go:163)
		if err := nb.DeleteTaskList(testCalendarID); err != nil {
			t.Errorf("Failed to delete test calendar: %v", err)
			return
		}
		t.Logf("✓ Calendar deleted")

		// Verify the calendar was actually deleted
		time.Sleep(100 * time.Millisecond)
		calendars, err := nb.GetTaskLists()
		if err != nil {
			t.Logf("Warning: failed to verify calendar deletion: %v", err)
			return
		}

		for _, cal := range calendars {
			if cal.ID == testCalendarID {
				t.Errorf("Calendar still exists after deletion!")
				return
			}
		}
		t.Logf("✓ Calendar deletion verified")
	}()

	// Test 1: Add a task
	testTaskName := "Test Task " + time.Now().Format("20060102150405")
	task := backend.Task{
		Summary: testTaskName,
		Status:  "NEEDS-ACTION",
		Created: time.Now(),
	}

	taskUID, err := nb.AddTask(testCalendarID, task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}
	t.Logf("Created task with UID: %s", taskUID)

	// Verify task was created
	tasks, err := nb.GetTasks(testCalendarID, nil)
	if err != nil {
		t.Fatalf("Failed to get tasks: %v", err)
	}

	found := false
	var createdTask backend.Task
	for _, tsk := range tasks {
		if tsk.UID == taskUID {
			found = true
			createdTask = tsk
			break
		}
	}
	if !found {
		t.Fatalf("Task %s not found after creation", taskUID)
	}

	time.Sleep(100 * time.Millisecond)

	// Test 2: Complete the task (THIS IS THE CRITICAL TEST)
	createdTask.Status = "COMPLETED"
	err = nb.UpdateTask(testCalendarID, createdTask)
	if err != nil {
		t.Fatalf("Failed to complete task %s: %v", taskUID, err)
	}
	t.Logf("Successfully completed task %s", taskUID)

	// Verify task is completed
	time.Sleep(100 * time.Millisecond) // Give Nextcloud API time to process
	tasks, err = nb.GetTasks(testCalendarID, nil)
	if err != nil {
		t.Fatalf("Failed to get tasks after complete: %v", err)
	}

	// The task might not appear in active tasks after completion
	// Try to find it or verify it's gone from active list
	stillActive := false
	for _, tsk := range tasks {
		if tsk.UID == taskUID && tsk.Status != "COMPLETED" {
			stillActive = true
			break
		}
	}
	if stillActive {
		t.Errorf("Task %s is still active after completion", taskUID)
	}

	time.Sleep(100 * time.Millisecond)
	// Test 3: Delete the task (THIS IS ANOTHER CRITICAL TEST)
	err = nb.DeleteTask(testCalendarID, taskUID)
	if err != nil {
		t.Fatalf("Failed to delete task %s: %v", taskUID, err)
	}
	t.Logf("Successfully deleted task %s", taskUID)

	// Verify task is deleted
	time.Sleep(100 * time.Millisecond)
	tasks, err = nb.GetTasks(testCalendarID, nil)
	if err != nil {
		t.Fatalf("Failed to get tasks after delete: %v", err)
	}

	for _, tsk := range tasks {
		if tsk.UID == taskUID {
			t.Errorf("Task %s still exists after deletion", taskUID)
		}
	}
}

// TestNextcloudWithSyncOperations tests Nextcloud operations WITH sync enabled
// This is the REAL test that should catch the UID mismatch bug
// Uses operations layer (HandleAddAction, HandleCompleteAction, etc.)
// instead of calling backend methods directly
func TestNextcloudWithSyncOperationsIntegration(t *testing.T) {
	// Check for required environment variables
	ncURL := os.Getenv("GOSYNCTASKS_NEXTCLOUD_HOST")
	ncUser := os.Getenv("GOSYNCTASKS_NEXTCLOUD_USERNAME")
	ncPass := os.Getenv("GOSYNCTASKS_NEXTCLOUD_PASSWORD")

	if ncURL == "" || ncUser == "" || ncPass == "" {
		t.Skip("GOSYNCTASKS_NEXTCLOUD_HOST, GOSYNCTASKS_NEXTCLOUD_USERNAME, and GOSYNCTASKS_NEXTCLOUD_PASSWORD must be set for Nextcloud sync integration tests")
	}

	// Setup: Create SQLite cache backend
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nextcloud_cache.db")

	cacheBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Name:    "nextcloud",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create cache backend: %v", err)
	}
	defer cacheBackend.Close()

	// Create Nextcloud remote backend
	remoteBackend, err := createNextcloudBackend(t)
	if err != nil {
		t.Fatalf("Failed to create Nextcloud backend: %v", err)
	}

	// Create sync manager
	syncManager := sync.NewSyncManager(cacheBackend, remoteBackend, sync.ServerWins)

	// ============================================================
	// TEST 0: Create test calendar on remote and cache (tests CreateTaskList like CLI does)
	// ============================================================
	testCalendarName := "GoSyncTasks Test " + time.Now().Format("20060102150405")
	t.Logf("Test 0a: Creating test calendar on remote via CreateTaskList: %s", testCalendarName)

	// Create calendar on remote first (this is how the CLI creates calendars - cmd/gosynctasks/list.go:93)
	testCalendarID, err := remoteBackend.CreateTaskList(testCalendarName, "Integration test calendar", "")
	if err != nil {
		t.Fatalf("Failed to create test calendar on remote: %v", err)
	}
	t.Logf("✓ Remote calendar created with ID: %s", testCalendarID)

	// Verify the calendar was created on remote
	remoteCalendars, err := remoteBackend.GetTaskLists()
	if err != nil {
		t.Fatalf("Failed to get remote task lists: %v", err)
	}

	remoteCalendarFound := false
	for _, cal := range remoteCalendars {
		if cal.ID == testCalendarID && cal.Name == testCalendarName {
			remoteCalendarFound = true
			break
		}
	}
	if !remoteCalendarFound {
		t.Fatalf("Created calendar not found in remote task lists")
	}
	t.Logf("✓ Remote calendar verified")

	// Track if calendar was created to ensure we always try to delete it
	calendarCreated := true
	defer func() {
		if !calendarCreated {
			return
		}

		// ============================================================
		// CLEANUP: Delete test calendar from remote (tests DeleteTaskList like CLI does)
		// ============================================================
		t.Log("Cleanup: Deleting test calendar from remote via DeleteTaskList...")

		// This is exactly how the CLI deletes calendars (cmd/gosynctasks/list.go:163)
		if err := remoteBackend.DeleteTaskList(testCalendarID); err != nil {
			t.Errorf("Failed to delete test calendar from remote: %v", err)
			return
		}
		t.Logf("✓ Remote calendar deleted")

		// Verify the calendar was actually deleted
		time.Sleep(100 * time.Millisecond)
		remoteCalendars, err := remoteBackend.GetTaskLists()
		if err != nil {
			t.Logf("Warning: failed to verify remote calendar deletion: %v", err)
			return
		}

		for _, cal := range remoteCalendars {
			if cal.ID == testCalendarID {
				t.Errorf("Remote calendar still exists after deletion!")
				return
			}
		}
		t.Logf("✓ Remote calendar deletion verified")
	}()

	// Create the same calendar in cache with the SAME ID as remote (for sync operations)
	t.Logf("Test 0b: Creating test calendar in cache with remote ID: %s", testCalendarID)

	// Insert list directly into cache database with the remote calendar ID
	// This is necessary because sync expects cache and remote to use the same list IDs
	db, err := cacheBackend.GetDB()
	if err != nil {
		t.Fatalf("Failed to get cache DB: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO list_sync_metadata (list_id, backend_name, list_name, list_color, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, testCalendarID, "nextcloud", testCalendarName, "", time.Now().Unix(), time.Now().Unix())
	if err != nil {
		t.Fatalf("Failed to insert list into cache: %v", err)
	}

	cacheCalendarID := testCalendarID
	t.Logf("✓ Cache calendar created with remote ID: %s", cacheCalendarID)

	// Use cache calendar ID for operations
	testCalendar := backend.TaskList{
		ID:          cacheCalendarID,
		Name:        testCalendarName,
		Description: "Integration test calendar",
	}

	// Create mock sync provider (doesn't spawn background processes)
	mockSync := newMockSyncProvider()

	// Create test config with sync enabled and backend configuration
	cfg := &config.Config{
		DateFormat: "2006-01-02",
		Sync: &config.SyncConfig{
			Enabled:            true,
			AutoSync:           true,
			LocalBackend:       "sqlite",
			ConflictResolution: "server_wins",
		},
		Backends: map[string]backend.BackendConfig{
			"nextcloud": {
				Name:    "nextcloud",
				Type:    "nextcloud",
				Enabled: true,
				// Sync will use cache_nextcloud as the cache backend
			},
		},
	}
	// Set this as the global config so triggerPushSync sees it
	config.SetConfigForTest(cfg)

	testTaskName := "Sync Test " + time.Now().Format("20060102150405")

	// ============================================================
	// TEST 1: Add task using HandleAddAction (operations layer)
	// ============================================================
	t.Log("Test 1: Adding task via operations layer...")

	// Capture output to avoid test clutter
	capture := newOutputCapture()
	capture.start()

	// Create mock command with task summary as argument
	cmd := newMockCommand()

	// Call HandleAddAction - this is what the CLI actually calls!
	err = operations.HandleAddAction(
		cmd.Command,
		cacheBackend,
		&testCalendar,
		testTaskName, // task summary
		mockSync,
	)

	_ = capture.stop() // Ignore output

	if err != nil {
		t.Fatalf("HandleAddAction failed: %v", err)
	}

	// Verify task was added to cache with pending UID
	cachedTasks, err := cacheBackend.GetTasks(cacheCalendarID, nil)
	if err != nil {
		t.Fatalf("Failed to get cached tasks: %v", err)
	}

	var addedTask *backend.Task
	for i := range cachedTasks {
		if cachedTasks[i].Summary == testTaskName {
			addedTask = &cachedTasks[i]
			break
		}
	}
	if addedTask == nil {
		t.Fatalf("Task not found in cache after HandleAddAction")
	}

	t.Logf("Task added with UID: %s", addedTask.UID)

	// Verify it has a pending UID
	if addedTask.UID[:8] != "pending-" {
		t.Errorf("Expected pending UID, got: %s", addedTask.UID)
	}

	time.Sleep(100 * time.Millisecond)

	// ============================================================
	// TEST 2: Sync to push task to Nextcloud
	// ============================================================
	t.Log("Test 2: Syncing to Nextcloud...")

	// Check pending operations before sync
	pendingOps, err := cacheBackend.GetPendingSyncOperations()
	if err != nil {
		t.Fatalf("Failed to get pending ops: %v", err)
	}
	t.Logf("Pending operations before sync: %d", len(pendingOps))
	for _, op := range pendingOps {
		t.Logf("  - Operation: %s, TaskUID: %s, ListID: %s", op.Operation, op.TaskUID, op.ListID)
	}

	result, err := syncManager.Sync()
	if err != nil {
		t.Fatalf("Initial sync failed: %v", err)
	}
	t.Logf("Initial sync: pushed %d tasks", result.PushedTasks)

	if result.PushedTasks != 1 {
		t.Errorf("Expected 1 task to be pushed, got %d", result.PushedTasks)
	}

	// Get the task from cache to see the updated UID
	cachedTasks, err = cacheBackend.GetTasks(cacheCalendarID, nil)
	if err != nil {
		t.Fatalf("Failed to get cached tasks: %v", err)
	}

	var syncedTask *backend.Task
	for i := range cachedTasks {
		if cachedTasks[i].Summary == testTaskName {
			syncedTask = &cachedTasks[i]
			break
		}
	}
	if syncedTask == nil {
		t.Fatalf("Task %s not found in cache after sync", testTaskName)
	}

	t.Logf("Task UID after sync: %s", syncedTask.UID)

	// CRITICAL TEST: The UID should now be a Nextcloud UID, not pending
	if syncedTask.UID[:8] == "pending-" {
		t.Errorf("BUG: UID is still pending after sync! UID: %s", syncedTask.UID)
	}

	time.Sleep(100 * time.Millisecond)
	// ============================================================
	// TEST 3: Complete task using HandleCompleteAction (operations layer)
	// ============================================================
	t.Log("Test 3: Completing task via operations layer...")

	capture = newOutputCapture()
	capture.start()

	// Create command with status flag
	completeCmd := newMockCommand()

	// Call HandleCompleteAction with the task summary to find it
	// This mimics: gosynctasks TestCalendar complete "Sync Test..."
	err = operations.HandleCompleteAction(
		completeCmd.Command,
		cacheBackend,
		cfg,
		&testCalendar,
		testTaskName, // search by summary
		mockSync,
	)

	_ = capture.stop()

	if err != nil {
		t.Fatalf("HandleCompleteAction failed: %v", err)
	}

	// Verify task is marked DONE in cache
	cachedTasks, err = cacheBackend.GetTasks(cacheCalendarID, nil)
	if err != nil {
		t.Fatalf("Failed to get cached tasks: %v", err)
	}

	completed := false
	for _, tsk := range cachedTasks {
		if tsk.Summary == testTaskName {
			// Nextcloud uses COMPLETED, SQLite stores as DONE
			if tsk.Status != "DONE" && tsk.Status != "COMPLETED" {
				t.Errorf("Task status is %s, expected DONE or COMPLETED", tsk.Status)
			} else {
				completed = true
			}
			break
		}
	}
	if !completed {
		t.Error("Task not marked as completed in cache")
	}

	time.Sleep(100 * time.Millisecond)
	// ============================================================
	// TEST 4: Sync to push completion to Nextcloud
	// ============================================================
	t.Log("Test 4: Syncing completion to Nextcloud...")

	result, err = syncManager.Sync()
	if err != nil {
		t.Fatalf("Sync after complete failed: %v", err)
	}
	t.Logf("Sync after complete: pushed %d tasks", result.PushedTasks)

	if result.PushedTasks != 1 {
		t.Errorf("Expected 1 task to be pushed after complete, got %d", result.PushedTasks)
	}

	// Verify task is completed on Nextcloud
	time.Sleep(2 * time.Second) // Give Nextcloud time to process
	remoteTasks, err := remoteBackend.GetTasks(testCalendarID, nil)
	if err != nil {
		t.Fatalf("Failed to get remote tasks: %v", err)
	}

	// Task should not appear in active tasks (it's completed)
	for _, tsk := range remoteTasks {
		if tsk.Summary == testTaskName && tsk.Status != "COMPLETED" {
			t.Errorf("Task is still active on Nextcloud after completion")
		}
	}

	time.Sleep(100 * time.Millisecond)
	// ============================================================
	// TEST 5: Delete task using backend (HandleDeleteAction requires confirmation)
	// ============================================================
	t.Log("Test 5: Deleting task...")

	// Get the current task UID
	cachedTasks, err = cacheBackend.GetTasks(cacheCalendarID, nil)
	if err != nil {
		t.Fatalf("Failed to get cached tasks: %v", err)
	}

	var taskUID string
	for _, tsk := range cachedTasks {
		if tsk.Summary == testTaskName {
			taskUID = tsk.UID
			break
		}
	}
	if taskUID == "" {
		t.Fatal("Task not found for deletion")
	}

	// Delete directly since HandleDeleteAction requires user confirmation
	err = cacheBackend.DeleteTask(cacheCalendarID, taskUID)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}
	t.Logf("Task deleted from cache")

	time.Sleep(100 * time.Millisecond)
	// ============================================================
	// TEST 6: Sync to push deletion to Nextcloud
	// ============================================================
	t.Log("Test 6: Syncing deletion to Nextcloud...")

	result, err = syncManager.Sync()
	if err != nil {
		t.Fatalf("Sync after delete failed: %v", err)
	}
	t.Logf("Sync after delete: pushed %d tasks", result.PushedTasks)

	// Verify task is deleted from Nextcloud
	time.Sleep(2 * time.Second)
	remoteTasks, err = remoteBackend.GetTasks(testCalendarID, nil)
	if err != nil {
		t.Fatalf("Failed to get remote tasks after delete: %v", err)
	}

	for _, tsk := range remoteTasks {
		if tsk.Summary == testTaskName {
			t.Errorf("Task still exists on Nextcloud after deletion")
		}
	}

}

// TestNextcloudUIDUpdateAfterSync verifies that pending UIDs are updated after sync
func TestNextcloudUIDUpdateAfterSyncIntegration(t *testing.T) {
	// Check for required environment variables
	ncURL := os.Getenv("GOSYNCTASKS_NEXTCLOUD_HOST")
	ncUser := os.Getenv("GOSYNCTASKS_NEXTCLOUD_USERNAME")
	ncPass := os.Getenv("GOSYNCTASKS_NEXTCLOUD_PASSWORD")

	if ncURL == "" || ncUser == "" || ncPass == "" {
		t.Skip("GOSYNCTASKS_NEXTCLOUD_HOST, GOSYNCTASKS_NEXTCLOUD_USERNAME, and GOSYNCTASKS_NEXTCLOUD_PASSWORD must be set")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "uid_test.db")

	cacheBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Name:    "nextcloud",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cacheBackend.Close()

	remoteBackend, err := createNextcloudBackend(t)
	if err != nil {
		t.Fatalf("Failed to create Nextcloud backend: %v", err)
	}

	syncManager := sync.NewSyncManager(cacheBackend, remoteBackend, sync.ServerWins)

	// ============================================================
	// TEST 0: Create test calendar (tests CreateTaskList like CLI does)
	// ============================================================
	testCalendarName := "GoSyncTasks UID Test " + time.Now().Format("20060102150405")
	t.Logf("Test 0: Creating test calendar: %s", testCalendarName)

	// Create on remote (this is how the CLI creates calendars - cmd/gosynctasks/list.go:93)
	testCalendarID, err := remoteBackend.CreateTaskList(testCalendarName, "UID update test calendar", "")
	if err != nil {
		t.Fatalf("Failed to create test calendar on remote: %v", err)
	}
	t.Logf("✓ Remote calendar created with ID: %s", testCalendarID)

	// Track for cleanup
	calendarCreated := true
	defer func() {
		if !calendarCreated {
			return
		}
		t.Log("Cleanup: Deleting test calendar...")
		if err := remoteBackend.DeleteTaskList(testCalendarID); err != nil {
			t.Logf("Warning: failed to delete test calendar: %v", err)
		} else {
			t.Logf("✓ Test calendar deleted")
		}
	}()

	// Create in cache with the SAME ID as remote
	db, err := cacheBackend.GetDB()
	if err != nil {
		t.Fatalf("Failed to get cache DB: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO list_sync_metadata (list_id, backend_name, list_name, list_color, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, testCalendarID, "nextcloud", testCalendarName, "", time.Now().Unix(), time.Now().Unix())
	if err != nil {
		t.Fatalf("Failed to insert calendar into cache: %v", err)
	}

	cacheCalendarID := testCalendarID
	t.Logf("✓ Cache calendar created with remote ID: %s", cacheCalendarID)

	// Add task locally
	task := backend.Task{
		Summary: "UID Test " + time.Now().Format("20060102150405"),
		Status:  "TODO",
		Created: time.Now(),
	}

	pendingUID, err := cacheBackend.AddTask(cacheCalendarID, task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// Verify it's a pending UID
	if pendingUID[:8] != "pending-" {
		t.Fatalf("Expected pending UID, got: %s", pendingUID)
	}
	t.Logf("Task created with pending UID: %s", pendingUID)

	// Check pending operations before sync
	pendingOps, err := cacheBackend.GetPendingSyncOperations()
	if err != nil {
		t.Fatalf("Failed to get pending ops: %v", err)
	}
	t.Logf("Pending operations before sync: %d", len(pendingOps))
	for _, op := range pendingOps {
		t.Logf("  - Operation: %s, TaskUID: %s, ListID: %s", op.Operation, op.TaskUID, op.ListID)
	}

	// Sync
	result, err := syncManager.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	t.Logf("Sync result: pushed %d tasks", result.PushedTasks)

	// Check UID was updated
	tasks, err := cacheBackend.GetTasks(cacheCalendarID, nil)
	if err != nil {
		t.Fatalf("Failed to get tasks: %v", err)
	}

	found := false
	var finalUID string
	for _, tsk := range tasks {
		if tsk.Summary == task.Summary {
			found = true
			finalUID = tsk.UID
			break
		}
	}

	if !found {
		t.Fatal("Task not found after sync")
	}

	// CRITICAL CHECK: UID should be updated to Nextcloud UID
	if finalUID[:8] == "pending-" {
		t.Errorf("BUG DETECTED: UID was NOT updated after sync! Still pending: %s", finalUID)
		t.Error("This is why complete and delete don't work - they're using the wrong UID!")
	} else {
		t.Logf("UID correctly updated from %s to %s", pendingUID, finalUID)
	}

	// Cleanup task before calendar cleanup
	if err := remoteBackend.DeleteTask(testCalendarID, finalUID); err != nil {
		t.Logf("Warning: failed to delete test task: %v", err)
	}
}
