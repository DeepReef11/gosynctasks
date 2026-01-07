package backend_test

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"gosynctasks/backend"
	"gosynctasks/backend/sqlite"
	"gosynctasks/backend/sync"
)

// Integration tests for end-to-end sync workflows
// These tests verify the complete sync system across multiple components

// TestBasicSyncWorkflow tests the basic end-to-end sync flow:
// remote → local → local modification → remote
func TestBasicSyncWorkflow(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create backends
	localBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create local backend: %v", err)
	}
	defer localBackend.Close()

	remoteBackend := backend.NewMockBackend()
	sm := sync.NewSyncManager(localBackend, remoteBackend, sync.ServerWins)

	// Step 1: Create data on remote
	listID, err := remoteBackend.CreateTaskList("Work Tasks", "Work related tasks", "#ff0000")
	if err != nil {
		t.Fatalf("Failed to create remote list: %v", err)
	}
	remoteBackend.Lists[0].CTags = "ctag-001"

	now := time.Now()
	for i := 1; i <= 5; i++ {
		_, err := remoteBackend.AddTask(listID, backend.Task{
			UID:      fmt.Sprintf("task-%d", i),
			Summary:  fmt.Sprintf("Remote backend.Task %d", i),
			Status:   "NEEDS-ACTION",
			Priority: i,
			Created:  now,
			Modified: now,
		})
		if err != nil {
			t.Fatalf("Failed to add remote task %d: %v", i, err)
		}
	}

	// Step 2: Pull from remote → local
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Initial sync failed: %v", err)
	}

	if result.PulledTasks != 5 {
		t.Errorf("Expected 5 pulled tasks, got %d", result.PulledTasks)
	}

	// Verify tasks are in local
	localTasks, err := localBackend.GetTasks(listID, nil)
	if err != nil {
		t.Fatalf("Failed to get local tasks: %v", err)
	}
	if len(localTasks) != 5 {
		t.Errorf("Expected 5 local tasks, got %d", len(localTasks))
	}

	// Step 3: Modify task locally
	task := localTasks[0]
	task.Summary = "Modified Locally"
	task.Priority = 1
	err = localBackend.UpdateTask(listID, task)
	if err != nil {
		t.Fatalf("Failed to update task locally: %v", err)
	}

	// Step 4: Push local modifications → remote
	result, err = sm.Sync()
	if err != nil {
		t.Fatalf("Sync after local modification failed: %v", err)
	}

	if result.PushedTasks != 1 {
		t.Errorf("Expected 1 pushed task, got %d", result.PushedTasks)
	}

	// Verify remote has the modification
	remoteTasks, err := remoteBackend.GetTasks(listID, nil)
	if err != nil {
		t.Fatalf("Failed to get remote tasks: %v", err)
	}

	found := false
	for _, rt := range remoteTasks {
		if rt.UID == task.UID && rt.Summary == "Modified Locally" && rt.Priority == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Remote backend does not have the locally modified task")
	}
}

// TestOfflineModeWorkflow tests offline operations with queue management
func TestOfflineModeWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	localBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create local backend: %v", err)
	}
	defer localBackend.Close()

	// Create list locally
	listID, err := localBackend.CreateTaskList("Offline Work", "", "")
	if err != nil {
		t.Fatalf("Failed to create local list: %v", err)
	}

	// Simulate offline: Create tasks locally
	now := time.Now()
	taskUIDs := []string{}
	for i := 1; i <= 3; i++ {
		task := backend.Task{
			UID:      fmt.Sprintf("offline-task-%d", i),
			Summary:  fmt.Sprintf("Offline backend.Task %d", i),
			Status:   "NEEDS-ACTION",
			Priority: i,
			Created:  now,
			Modified: now,
		}
		_, err := localBackend.AddTask(listID, task)
		if err != nil {
			t.Fatalf("Failed to add offline task %d: %v", i, err)
		}
		taskUIDs = append(taskUIDs, task.UID)
	}

	// Verify operations are queued
	ops, err := localBackend.GetPendingSyncOperations()
	if err != nil {
		t.Fatalf("Failed to get pending operations: %v", err)
	}

	if len(ops) != 3 {
		t.Errorf("Expected 3 pending operations, got %d", len(ops))
	}

	for _, op := range ops {
		if op.Operation != "create" {
			t.Errorf("Expected operation 'create', got '%s'", op.Operation)
		}
	}

	// Simulate going online: Create remote backend and sync
	remoteBackend := backend.NewMockBackend()
	remoteBackend.Lists = append(remoteBackend.Lists, backend.TaskList{
		ID:    listID,
		Name:  "Offline Work",
		CTags: "ctag-100",
	})
	remoteBackend.Tasks[listID] = []backend.Task{}

	sm := sync.NewSyncManager(localBackend, remoteBackend, sync.ServerWins)

	// Sync should push all queued operations
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Sync after coming online failed: %v", err)
	}

	if result.PushedTasks != 3 {
		t.Errorf("Expected 3 pushed tasks, got %d", result.PushedTasks)
	}

	// Verify queue is cleared
	ops, err = localBackend.GetPendingSyncOperations()
	if err != nil {
		t.Fatalf("Failed to get pending operations after sync: %v", err)
	}

	if len(ops) != 0 {
		t.Errorf("Expected 0 pending operations after sync, got %d", len(ops))
	}

	// Verify remote has all tasks
	remoteTasks, err := remoteBackend.GetTasks(listID, nil)
	if err != nil {
		t.Fatalf("Failed to get remote tasks: %v", err)
	}

	if len(remoteTasks) != 3 {
		t.Errorf("Expected 3 remote tasks after sync, got %d", len(remoteTasks))
	}

}

// TestConflictResolutionScenarios tests all four conflict resolution strategies
func TestConflictResolutionScenarios(t *testing.T) {
	strategies := []struct {
		name              sync.ConflictResolutionStrategy
		expectedLocal     string // Expected local task summary after conflict
		expectedRemote    string // Expected remote task summary (for local_wins)
		expectedTaskCount int    // Expected task count (for keep_both)
	}{
		{sync.ServerWins, "Remote Modification", "", 1},
		{sync.LocalWins, "Local Modification", "", 1},
		{sync.Merge, "Remote Modification", "", 1},    // sync.Merge uses remote summary but may merge other fields
		{sync.KeepBoth, "Remote Modification", "", 2}, // Keep both creates a copy
	}

	for _, strategy := range strategies {
		t.Run(string(strategy.name), func(t *testing.T) {
			tmpDir := t.TempDir()
			dbPath := filepath.Join(tmpDir, "test.db")

			localBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
				Type:    "sqlite",
				Enabled: true,
				DBPath:  dbPath,
			})
			if err != nil {
				t.Fatalf("Failed to create local backend: %v", err)
			}
			defer localBackend.Close()

			remoteBackend := backend.NewMockBackend()

			// Create list on both
			listID, _ := localBackend.CreateTaskList("Test List", "", "")
			remoteBackend.Lists = append(remoteBackend.Lists, backend.TaskList{
				ID:    listID,
				Name:  "Test List",
				CTags: "ctag-123",
			})
			remoteBackend.Tasks[listID] = []backend.Task{}

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
			taskUID, err := localBackend.AddTask(listID, task)
			if err != nil {
				t.Fatalf("Failed to add task: %v", err)
			}

			// Modify locally using the actual UID
			task.UID = taskUID
			task.Summary = "Local Modification"
			task.Priority = 1
			localBackend.UpdateTask(listID, task)

			// Modify remotely with the same UID
			remoteTask := task
			remoteTask.Summary = "Remote Modification"
			remoteTask.Priority = 9
			remoteTask.Modified = now.Add(time.Second)
			remoteBackend.AddTask(listID, remoteTask)

			// Change CTag to trigger sync
			remoteBackend.Lists[0].CTags = "ctag-456"

			// Create sync manager with specific strategy
			sm := sync.NewSyncManager(localBackend, remoteBackend, strategy.name)

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

			// Verify outcome based on strategy
			localTasks, _ := localBackend.GetTasks(listID, nil)
			if len(localTasks) != strategy.expectedTaskCount {
				t.Errorf("Expected %d tasks, got %d", strategy.expectedTaskCount, len(localTasks))
			}

			// Check summary of main task
			foundExpected := false
			for _, lt := range localTasks {
				if lt.Summary == strategy.expectedLocal {
					foundExpected = true
					break
				}
			}
			if !foundExpected {
				t.Errorf("Expected to find task with summary '%s' after %s strategy",
					strategy.expectedLocal, strategy.name)
			}

			// For keep_both, verify we have a copy
			if strategy.name == sync.KeepBoth {
				foundCopy := false
				for _, lt := range localTasks {
					if lt.Summary == "Local Modification (local copy)" {
						foundCopy = true
						break
					}
				}
				if !foundCopy {
					t.Error("Expected to find '(local copy)' task in keep_both strategy")
				}
			}

		})
	}
}

// TestLargeDatasetPerformance tests sync performance with 1000+ tasks
func TestLargeDatasetPerformance(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	localBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create local backend: %v", err)
	}
	defer localBackend.Close()

	remoteBackend := backend.NewMockBackend()
	sm := sync.NewSyncManager(localBackend, remoteBackend, sync.ServerWins)

	// Create list on remote
	listID, err := remoteBackend.CreateTaskList("Large Dataset", "", "")
	if err != nil {
		t.Fatalf("Failed to create remote list: %v", err)
	}
	remoteBackend.Lists[0].CTags = "ctag-large"

	// Create 1000 tasks on remote
	now := time.Now()
	taskCount := 1000

	t.Logf("Creating %d tasks on remote...", taskCount)
	for i := 1; i <= taskCount; i++ {
		_, err := remoteBackend.AddTask(listID, backend.Task{
			UID:      fmt.Sprintf("large-task-%d", i),
			Summary:  fmt.Sprintf("backend.Task %d of %d", i, taskCount),
			Status:   "NEEDS-ACTION",
			Priority: (i % 9) + 1,
			Created:  now,
			Modified: now,
		})
		if err != nil {
			t.Fatalf("Failed to add task %d: %v", i, err)
		}
	}

	// Measure sync performance
	t.Logf("Starting sync of %d tasks...", taskCount)
	startTime := time.Now()

	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	duration := time.Since(startTime)

	// Verify all tasks were synced
	if result.PulledTasks != taskCount {
		t.Errorf("Expected %d pulled tasks, got %d", taskCount, result.PulledTasks)
	}

	// Check performance requirement (30 seconds)
	if duration > 30*time.Second {
		t.Errorf("Sync took %v, exceeds 30 second limit", duration)
	}

	// Verify local has all tasks
	localTasks, err := localBackend.GetTasks(listID, nil)
	if err != nil {
		t.Fatalf("Failed to get local tasks: %v", err)
	}

	if len(localTasks) != taskCount {
		t.Errorf("Expected %d local tasks, got %d", taskCount, len(localTasks))
	}

}

// TestErrorRecoveryWithRetry tests error handling and retry logic
func TestErrorRecoveryWithRetry(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	localBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create local backend: %v", err)
	}
	defer localBackend.Close()

	remoteBackend := backend.NewMockBackend()

	// Create list
	listID, err := localBackend.CreateTaskList("Retry Test", "", "")
	if err != nil {
		t.Fatalf("Failed to create local list: %v", err)
	}

	// Setup remote
	remoteBackend.Lists = append(remoteBackend.Lists, backend.TaskList{
		ID:    listID,
		Name:  "Retry Test",
		CTags: "ctag-retry",
	})
	remoteBackend.Tasks[listID] = []backend.Task{}

	// Add task locally
	now := time.Now()
	task := backend.Task{
		UID:      "retry-task",
		Summary:  "backend.Task to retry",
		Status:   "NEEDS-ACTION",
		Created:  now,
		Modified: now,
	}
	_, err = localBackend.AddTask(listID, task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// Verify operation is queued
	ops, _ := localBackend.GetPendingSyncOperations()
	if len(ops) != 1 {
		t.Fatalf("Expected 1 pending operation, got %d", len(ops))
	}

	// Simulate network error
	remoteBackend.AddTaskErr = fmt.Errorf("network error: connection timeout")

	sm := sync.NewSyncManager(localBackend, remoteBackend, sync.ServerWins)

	// First sync attempt should fail
	result, err := sm.Sync()
	if err != nil {
		t.Logf("Expected sync to have errors: %v", err)
	}

	// Verify retry count incremented
	ops, _ = localBackend.GetPendingSyncOperations()
	if len(ops) != 1 {
		t.Fatalf("Expected 1 pending operation after failed sync, got %d", len(ops))
	}

	if ops[0].RetryCount != 1 {
		t.Errorf("Expected retry count 1, got %d", ops[0].RetryCount)
	}

	if ops[0].LastError == "" {
		t.Error("Expected last error to be set")
	}

	// Clear error and retry
	remoteBackend.AddTaskErr = nil

	result, err = sm.Sync()
	if err != nil {
		t.Fatalf("Retry sync failed: %v", err)
	}

	if result.PushedTasks != 1 {
		t.Errorf("Expected 1 pushed task on retry, got %d", result.PushedTasks)
	}

	// Verify queue is cleared
	ops, _ = localBackend.GetPendingSyncOperations()
	if len(ops) != 0 {
		t.Errorf("Expected 0 pending operations after successful retry, got %d", len(ops))
	}

	// Verify task is on remote
	remoteTasks, _ := remoteBackend.GetTasks(listID, nil)
	if len(remoteTasks) != 1 {
		t.Errorf("Expected 1 remote task after retry, got %d", len(remoteTasks))
	}

}

// TestConcurrentSyncOperations tests concurrent access to sync manager (race condition detection)
func TestConcurrentSyncOperations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	localBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create local backend: %v", err)
	}
	defer localBackend.Close()

	remoteBackend := backend.NewMockBackend()
	_, _ = remoteBackend.CreateTaskList("Concurrent Test", "", "")
	remoteBackend.Lists[0].CTags = "ctag-concurrent"

	sm := sync.NewSyncManager(localBackend, remoteBackend, sync.ServerWins)

	// Run multiple syncs concurrently
	done := make(chan bool)
	errors := make(chan error, 3)

	for i := 0; i < 3; i++ {
		go func(id int) {
			_, err := sm.Sync()
			if err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	errorCount := 0
	for err := range errors {
		t.Logf("Concurrent sync error: %v", err)
		errorCount++
	}

}

// TestHierarchicalTaskSync tests syncing tasks with parent-child relationships
func TestHierarchicalTaskSync(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	localBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create local backend: %v", err)
	}
	defer localBackend.Close()

	remoteBackend := backend.NewMockBackend()
	sm := sync.NewSyncManager(localBackend, remoteBackend, sync.ServerWins)

	// Create list on remote
	listID, _ := remoteBackend.CreateTaskList("Hierarchy Test", "", "")
	remoteBackend.Lists[0].CTags = "ctag-hierarchy"

	now := time.Now()

	// Create parent and child tasks on remote
	// IMPORTANT: Parents must come before children for foreign key constraints
	parentTask := backend.Task{
		UID:      "parent-task",
		Summary:  "Parent backend.Task",
		Status:   "NEEDS-ACTION",
		Created:  now,
		Modified: now,
	}
	remoteBackend.AddTask(listID, parentTask)

	childTask := backend.Task{
		UID:       "child-task",
		Summary:   "Child backend.Task",
		Status:    "NEEDS-ACTION",
		ParentUID: "parent-task",
		Created:   now,
		Modified:  now,
	}
	remoteBackend.AddTask(listID, childTask)

	// Sync
	result, err := sm.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.PulledTasks != 2 {
		t.Errorf("Expected 2 pulled tasks, got %d", result.PulledTasks)
	}

	// Verify hierarchy is preserved locally
	localTasks, _ := localBackend.GetTasks(listID, nil)
	if len(localTasks) != 2 {
		t.Fatalf("Expected 2 local tasks, got %d", len(localTasks))
	}

	var child *backend.Task
	for i := range localTasks {
		if localTasks[i].UID == "child-task" {
			child = &localTasks[i]
			break
		}
	}

	if child == nil {
		t.Fatal("Child task not found locally")
	}

	if child.ParentUID != "parent-task" {
		t.Errorf("Expected child ParentUID 'parent-task', got '%s'", child.ParentUID)
	}
}
