package todoist

import (
	"github.com/joho/godotenv"
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

func TestTodoistBackendIntegration_API(t *testing.T) {
	// Try to load .env file from project root (best effort, ignore errors)
	_ = godotenv.Load("../../.env")

	apiToken := os.Getenv("TODOIST_API_TOKEN")
	if apiToken == "" {
		t.Skip("Skipping integration test: TODOIST_API_TOKEN not set")
	}

	config := backend.BackendConfig{
		Type:     "todoist",
		Name:     "todoist",
		Enabled:  true,
		APIToken: apiToken,
	}

	tb, err := NewTodoistBackend(config)
	if err != nil {
		t.Fatalf("NewTodoistBackend() error = %v", err)
	}

	// Test GetTaskLists
	t.Run("GetTaskLists", func(t *testing.T) {
		lists, err := tb.GetTaskLists()
		if err != nil {
			t.Fatalf("GetTaskLists() error = %v", err)
		}
		if len(lists) == 0 {
			t.Log("No projects found (this is OK for a new account)")
		} else {
			t.Logf("Found %d projects", len(lists))
			t.Logf("First project: %s (ID: %s)", lists[0].Name, lists[0].ID)
		}
	})

	// Test creating, updating, and deleting a test project
	t.Run("ProjectLifecycle", func(t *testing.T) {
		// Create test project
		testProjectName := "gosynctasks-test-" + time.Now().Format("20060102-150405")
		projectID, err := tb.CreateTaskList(testProjectName, "Test project", "grey")
		if err != nil {
			t.Fatalf("CreateTaskList() error = %v", err)
		}
		t.Logf("Created test project: %s", projectID)

		// Cleanup at the end
		defer func() {
			if err := tb.DeleteTaskList(projectID); err != nil {
				t.Errorf("Cleanup: DeleteTaskList() error = %v", err)
			} else {
				t.Log("Cleaned up test project")
			}
		}()

		// Rename project
		newName := testProjectName + "-renamed"
		if err := tb.RenameTaskList(projectID, newName); err != nil {
			t.Fatalf("RenameTaskList() error = %v", err)
		}

		// Verify rename
		lists, err := tb.GetTaskLists()
		if err != nil {
			t.Fatalf("GetTaskLists() after rename error = %v", err)
		}
		found := false
		for _, list := range lists {
			if list.ID == projectID && list.Name == newName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Renamed project not found or name not updated")
		}
	})

	// Test task operations
	t.Run("TaskOperations", func(t *testing.T) {
		// Create a temporary project for testing
		testProjectName := "gosynctasks-task-test-" + time.Now().Format("20060102-150405")
		projectID, err := tb.CreateTaskList(testProjectName, "Test tasks", "grey")
		if err != nil {
			t.Fatalf("CreateTaskList() error = %v", err)
		}

		defer func() {
			tb.DeleteTaskList(projectID)
		}()

		// Add a task
		dueDate := time.Now().Add(24 * time.Hour)
		newTask := backend.Task{
			Summary:     "Integration Test Task",
			Description: "This is a test task created by integration tests",
			Priority:    1,
			Categories:  []string{"test", "integration"},
			DueDate:     &dueDate,
		}

		if _, err := tb.AddTask(projectID, newTask); err != nil {
			t.Fatalf("AddTask() error = %v", err)
		}
		t.Log("Created test task")

		// Get tasks
		tasks, err := tb.GetTasks(projectID, nil)
		if err != nil {
			t.Fatalf("GetTasks() error = %v", err)
		}

		if len(tasks) == 0 {
			t.Fatal("Expected at least one task")
		}

		testTask := tasks[0]
		t.Logf("Retrieved task: %s (ID: %s)", testTask.Summary, testTask.UID)

		// Update task
		testTask.Summary = "Updated Integration Test Task"
		testTask.Priority = 5
		if err := tb.UpdateTask(projectID, testTask); err != nil {
			t.Fatalf("UpdateTask() error = %v", err)
		}
		t.Log("Updated task")

		// Mark as done
		testTask.Status = "DONE"
		if err := tb.UpdateTask(projectID, testTask); err != nil {
			t.Fatalf("UpdateTask(DONE) error = %v", err)
		}
		t.Log("Marked task as done")

		// Delete task
		if err := tb.DeleteTask(projectID, testTask.UID); err != nil {
			t.Fatalf("DeleteTask() error = %v", err)
		}
		t.Log("Deleted task")
	})
}

// TestTodoistDirectOperations tests Todoist backend operations without sync
func TestTodoistDirectOperationsIntegration(t *testing.T) {
	apiToken := os.Getenv("TODOIST_API_TOKEN")
	if apiToken == "" {
		t.Skip("TODOIST_API_TOKEN not set, skipping Todoist integration tests")
	}

	tb, err := NewTodoistBackend(backend.BackendConfig{
		Type:     "todoist",
		Name:     "todoist",
		Enabled:  true,
		APIToken: apiToken,
	})
	if err != nil {
		t.Fatalf("Failed to create Todoist backend: %v", err)
	}

	// ============================================================
	// TEST 0: Create test list (tests CreateTaskList like CLI does)
	// ============================================================
	testListName := "GoSyncTasks Direct Test " + time.Now().Format("20060102150405")
	t.Logf("Test 0: Creating test list via CreateTaskList: %s", testListName)

	testListID, err := tb.CreateTaskList(testListName, "Direct operations test list", "")
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}
	t.Logf("✓ List created with ID: %s", testListID)

	// Verify the list was actually created by fetching all lists
	lists, err := tb.GetTaskLists()
	if err != nil {
		t.Fatalf("Failed to get task lists: %v", err)
	}

	listFound := false
	for _, list := range lists {
		if list.ID == testListID && list.Name == testListName {
			listFound = true
			break
		}
	}
	if !listFound {
		t.Fatalf("Created list not found in task lists")
	}
	t.Logf("✓ List verified in task lists")

	// Ensure cleanup: delete the test list when done
	// Track if list was created to ensure we always try to delete it
	listCreated := true
	defer func() {
		if !listCreated {
			return
		}

		// ============================================================
		// CLEANUP: Delete test list (tests DeleteTaskList like CLI does)
		// ============================================================
		t.Log("Cleanup: Deleting test list via DeleteTaskList...")

		// This is exactly how the CLI deletes lists (cmd/gosynctasks/list.go:163)
		if err := tb.DeleteTaskList(testListID); err != nil {
			t.Errorf("Failed to delete test list: %v", err)
			return
		}
		t.Logf("✓ List deleted")

		// Verify the list was actually deleted
		time.Sleep(100 * time.Millisecond)
		lists, err := tb.GetTaskLists()
		if err != nil {
			t.Logf("Warning: failed to verify list deletion: %v", err)
			return
		}

		for _, list := range lists {
			if list.ID == testListID {
				t.Errorf("List still exists after deletion!")
				return
			}
		}
		t.Logf("✓ List deletion verified")
	}()

	// Test 1: Add a task
	testTaskName := "Test Task " + time.Now().Format("20060102150405")
	task := backend.Task{
		Summary: testTaskName,
		Status:  "TODO",
		Created: time.Now(),
	}

	taskUID, err := tb.AddTask(testListID, task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}
	t.Logf("Created task with UID: %s", taskUID)

	// Verify task was created
	tasks, err := tb.GetTasks(testListID, nil)
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
	err = tb.UpdateTask(testListID, createdTask)
	if err != nil {
		t.Fatalf("Failed to complete task %s: %v", taskUID, err)
	}
	t.Logf("Successfully completed task %s", taskUID)

	// Verify task is completed
	time.Sleep(100 * time.Millisecond) // Give Todoist API time to process
	tasks, err = tb.GetTasks(testListID, nil)
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
	err = tb.DeleteTask(testListID, taskUID)
	if err != nil {
		t.Fatalf("Failed to delete task %s: %v", taskUID, err)
	}
	t.Logf("Successfully deleted task %s", taskUID)

	// Verify task is deleted
	time.Sleep(100 * time.Millisecond)
	tasks, err = tb.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get tasks after delete: %v", err)
	}

	for _, tsk := range tasks {
		if tsk.UID == taskUID {
			t.Errorf("Task %s still exists after deletion", taskUID)
		}
	}
}

// TestTodoistWithSyncOperations tests Todoist operations WITH sync enabled
func TestTodoistWithSyncOperationsIntegration(t *testing.T) {
	apiToken := os.Getenv("TODOIST_API_TOKEN")
	if apiToken == "" {
		t.Skip("TODOIST_API_TOKEN not set, skipping Todoist sync integration tests")
	}

	// Setup: Create SQLite cache backend
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "todoist_cache.db")

	cacheBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Name:    "todoist",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create cache backend: %v", err)
	}
	defer cacheBackend.Close()

	// Create Todoist remote backend
	remoteBackend, err := NewTodoistBackend(backend.BackendConfig{
		Type:     "todoist",
		Name:     "todoist",
		Enabled:  true,
		APIToken: apiToken,
	})
	if err != nil {
		t.Fatalf("Failed to create Todoist backend: %v", err)
	}

	// Create sync manager
	syncManager := sync.NewSyncManager(cacheBackend, remoteBackend, sync.ServerWins)

	// ============================================================
	// TEST 0: Create test list on remote and cache (tests CreateTaskList like CLI does)
	// ============================================================
	testListName := "GoSyncTasks Test " + time.Now().Format("20060102150405")
	t.Logf("Test 0a: Creating test list on remote via CreateTaskList: %s", testListName)

	// Create list on remote first (this is how the CLI creates lists - cmd/gosynctasks/list.go:93)
	testListID, err := remoteBackend.CreateTaskList(testListName, "Integration test list", "")
	if err != nil {
		t.Fatalf("Failed to create test list on remote: %v", err)
	}
	t.Logf("✓ Remote list created with ID: %s", testListID)

	// Verify the list was created on remote
	remoteLists, err := remoteBackend.GetTaskLists()
	if err != nil {
		t.Fatalf("Failed to get remote task lists: %v", err)
	}

	remoteListFound := false
	for _, list := range remoteLists {
		if list.ID == testListID && list.Name == testListName {
			remoteListFound = true
			break
		}
	}
	if !remoteListFound {
		t.Fatalf("Created list not found in remote task lists")
	}
	t.Logf("✓ Remote list verified")

	// Track if list was created to ensure we always try to delete it
	listCreated := true
	defer func() {
		if !listCreated {
			return
		}

		// ============================================================
		// CLEANUP: Delete test list from remote (tests DeleteTaskList like CLI does)
		// ============================================================
		t.Log("Cleanup: Deleting test list from remote via DeleteTaskList...")

		// This is exactly how the CLI deletes lists (cmd/gosynctasks/list.go:163)
		if err := remoteBackend.DeleteTaskList(testListID); err != nil {
			t.Errorf("Failed to delete test list from remote: %v", err)
			return
		}
		t.Logf("✓ Remote list deleted")

		// Verify the list was actually deleted
		time.Sleep(100 * time.Millisecond)
		remoteLists, err := remoteBackend.GetTaskLists()
		if err != nil {
			t.Logf("Warning: failed to verify remote list deletion: %v", err)
			return
		}

		for _, list := range remoteLists {
			if list.ID == testListID {
				t.Errorf("Remote list still exists after deletion!")
				return
			}
		}
		t.Logf("✓ Remote list deletion verified")
	}()

	// Create the same list in cache with the SAME ID as remote (for sync operations)
	t.Logf("Test 0b: Creating test list in cache with remote ID: %s", testListID)

	// Insert list directly into cache database with the remote list ID
	// This is necessary because sync expects cache and remote to use the same list IDs
	db, err := cacheBackend.GetDB()
	if err != nil {
		t.Fatalf("Failed to get cache DB: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO list_sync_metadata (list_id, backend_name, list_name, list_color, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, testListID, "todoist", testListName, "", time.Now().Unix(), time.Now().Unix())
	if err != nil {
		t.Fatalf("Failed to insert list into cache: %v", err)
	}

	cacheListID := testListID
	t.Logf("✓ Cache list created with remote ID: %s", cacheListID)

	// Use cache list ID for operations
	testList := backend.TaskList{
		ID:          cacheListID,
		Name:        testListName,
		Description: "Integration test list",
	}

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
			"todoist": {
				Name:     "todoist",
				Type:     "todoist",
				Enabled:  true,
				APIToken: apiToken,
			},
		},
	}
	// Set this as the global config so triggerPushSync sees it
	config.SetConfigForTest(cfg)

	// Create mock sync provider (doesn't spawn background processes)
	mockSync := newMockSyncProvider()

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
		&testList,
		testTaskName, // task summary
		mockSync,
	)

	_ = capture.stop() // Ignore output

	if err != nil {
		t.Fatalf("HandleAddAction failed: %v", err)
	}

	// Verify task was added to cache with pending UID
	cachedTasks, err := cacheBackend.GetTasks(cacheListID, nil)
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
	// TEST 2: Sync to push task to Todoist
	// ============================================================
	t.Log("Test 2: Syncing to Todoist...")

	// Check pending operations before sync (for debugging)
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
	cachedTasks, err = cacheBackend.GetTasks(cacheListID, nil)
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

	// CRITICAL TEST: The UID should now be a Todoist UID, not pending
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
	// This mimics: gosynctasks TestList complete "Sync Test..."
	err = operations.HandleCompleteAction(
		completeCmd.Command,
		cacheBackend,
		cfg,
		&testList,
		testTaskName, // search by summary
		mockSync,
	)

	_ = capture.stop()

	if err != nil {
		t.Fatalf("HandleCompleteAction failed: %v", err)
	}

	// Verify task is marked DONE in cache
	cachedTasks, err = cacheBackend.GetTasks(cacheListID, nil)
	if err != nil {
		t.Fatalf("Failed to get cached tasks: %v", err)
	}

	completed := false
	for _, tsk := range cachedTasks {
		if tsk.Summary == testTaskName {
			if tsk.Status != "COMPLETED" {
				t.Errorf("Task status is %s, expected COMPLETED", tsk.Status)
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
	// TEST 4: Sync to push completion to Todoist
	// ============================================================
	t.Log("Test 4: Syncing completion to Todoist...")

	result, err = syncManager.Sync()
	if err != nil {
		t.Fatalf("Sync after complete failed: %v", err)
	}
	t.Logf("Sync after complete: pushed %d tasks", result.PushedTasks)

	if result.PushedTasks != 1 {
		t.Errorf("Expected 1 task to be pushed after complete, got %d", result.PushedTasks)
	}

	// Verify task is completed on Todoist
	time.Sleep(2 * time.Second) // Give Todoist time to process
	remoteTasks, err := remoteBackend.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get remote tasks: %v", err)
	}

	// Task should not appear in active tasks (it's completed)
	for _, tsk := range remoteTasks {
		if tsk.Summary == testTaskName && tsk.Status != "COMPLETED" {
			t.Errorf("Task is still active on Todoist after completion")
		}
	}

	time.Sleep(100 * time.Millisecond)
	// ============================================================
	// TEST 5: Delete task using HandleDeleteAction (operations layer)
	// ============================================================
	t.Log("Test 5: Deleting task via operations layer...")

	// HandleDeleteAction prompts for confirmation, so we need to mock stdin
	// For now, we'll call the backend directly for delete
	// TODO: Add non-interactive mode to HandleDeleteAction for tests

	// Get the current task UID
	cachedTasks, err = cacheBackend.GetTasks(cacheListID, nil)
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
	err = cacheBackend.DeleteTask(cacheListID, taskUID)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}
	t.Logf("Task deleted from cache")

	time.Sleep(100 * time.Millisecond)
	// ============================================================
	// TEST 6: Sync to push deletion to Todoist
	// ============================================================
	t.Log("Test 6: Syncing deletion to Todoist...")

	result, err = syncManager.Sync()
	if err != nil {
		t.Fatalf("Sync after delete failed: %v", err)
	}
	t.Logf("Sync after delete: pushed %d tasks", result.PushedTasks)

	// Verify task is deleted from Todoist
	time.Sleep(2 * time.Second)
	remoteTasks, err = remoteBackend.GetTasks(testListID, nil)
	if err != nil {
		t.Fatalf("Failed to get remote tasks after delete: %v", err)
	}

	for _, tsk := range remoteTasks {
		if tsk.Summary == testTaskName {
			t.Errorf("Task still exists on Todoist after deletion")
		}
	}

}

// TestTodoistUIDUpdateAfterSync verifies that pending UIDs are updated after sync
func TestTodoistUIDUpdateAfterSyncIntegration(t *testing.T) {
	apiToken := os.Getenv("TODOIST_API_TOKEN")
	if apiToken == "" {
		t.Skip("TODOIST_API_TOKEN not set")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "uid_test.db")

	cacheBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Name:    "todoist",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cacheBackend.Close()

	remoteBackend, err := NewTodoistBackend(backend.BackendConfig{
		Type:     "todoist",
		Name:     "todoist",
		Enabled:  true,
		APIToken: apiToken,
	})
	if err != nil {
		t.Fatalf("Failed to create Todoist backend: %v", err)
	}

	syncManager := sync.NewSyncManager(cacheBackend, remoteBackend, sync.ServerWins)

	// ============================================================
	// TEST 0: Create test list (tests CreateTaskList like CLI does)
	// ============================================================
	testListName := "GoSyncTasks UID Test " + time.Now().Format("20060102150405")
	t.Logf("Test 0: Creating test list: %s", testListName)

	// Create on remote (this is how the CLI creates lists - cmd/gosynctasks/list.go:93)
	testListID, err := remoteBackend.CreateTaskList(testListName, "UID update test list", "")
	if err != nil {
		t.Fatalf("Failed to create test list on remote: %v", err)
	}
	t.Logf("✓ Remote list created with ID: %s", testListID)

	// Track for cleanup
	listCreated := true
	defer func() {
		if !listCreated {
			return
		}
		t.Log("Cleanup: Deleting test list...")
		if err := remoteBackend.DeleteTaskList(testListID); err != nil {
			t.Logf("Warning: failed to delete test list: %v", err)
		} else {
			t.Logf("✓ Test list deleted")
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
	`, testListID, "todoist", testListName, "", time.Now().Unix(), time.Now().Unix())
	if err != nil {
		t.Fatalf("Failed to insert list into cache: %v", err)
	}

	cacheListID := testListID
	t.Logf("✓ Cache list created with remote ID: %s", cacheListID)

	// Add task locally
	task := backend.Task{
		Summary: "UID Test " + time.Now().Format("20060102150405"),
		Status:  "TODO",
		Created: time.Now(),
	}

	pendingUID, err := cacheBackend.AddTask(cacheListID, task)
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
	tasks, err := cacheBackend.GetTasks(cacheListID, nil)
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

	// CRITICAL CHECK: UID should be updated to Todoist UID
	if finalUID[:8] == "pending-" {
		t.Errorf("BUG DETECTED: UID was NOT updated after sync! Still pending: %s", finalUID)
		t.Error("This is why complete and delete don't work - they're using the wrong UID!")
	} else {
		t.Logf("UID correctly updated from %s to %s", pendingUID, finalUID)
	}

	// Cleanup task before list cleanup
	if err := remoteBackend.DeleteTask(testListID, finalUID); err != nil {
		t.Logf("Warning: failed to delete test task: %v", err)
	}
}
