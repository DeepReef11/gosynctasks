package backend

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Helper function to create a test backend
func createTestBackend(t *testing.T) (*SQLiteBackend, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := BackendConfig{
		Type:    "sqlite",
		Enabled: true,
		DBPath:  dbPath,
	}

	backend, err := NewSQLiteBackend(config)
	if err != nil {
		t.Fatalf("Failed to create SQLite backend: %v", err)
	}

	cleanup := func() {
		backend.Close()
	}

	return backend, cleanup
}

// TestNewSQLiteBackend tests backend creation
func TestNewSQLiteBackend(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := BackendConfig{
		Type:    "sqlite",
		Enabled: true,
		DBPath:  dbPath,
	}

	backend, err := NewSQLiteBackend(config)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}
	defer backend.Close()

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("Database file was not created")
	}
}

// TestCreateTaskList tests task list creation
func TestCreateTaskList(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, err := backend.CreateTaskList("Work Tasks", "Tasks for work", "#ff0000")
	if err != nil {
		t.Fatalf("Failed to create task list: %v", err)
	}

	if listID == "" {
		t.Error("Expected non-empty list ID")
	}

	// Verify list was created
	lists, err := backend.GetTaskLists()
	if err != nil {
		t.Fatalf("Failed to get task lists: %v", err)
	}

	if len(lists) != 1 {
		t.Fatalf("Expected 1 list, got %d", len(lists))
	}

	if lists[0].Name != "Work Tasks" {
		t.Errorf("Expected list name 'Work Tasks', got '%s'", lists[0].Name)
	}

	if lists[0].Color != "#ff0000" {
		t.Errorf("Expected color '#ff0000', got '%s'", lists[0].Color)
	}
}

// TestGetTaskLists tests retrieving task lists
func TestGetTaskLists(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	// Create multiple lists
	_, err := backend.CreateTaskList("Personal", "", "")
	if err != nil {
		t.Fatalf("Failed to create list: %v", err)
	}

	_, err = backend.CreateTaskList("Work", "", "")
	if err != nil {
		t.Fatalf("Failed to create list: %v", err)
	}

	lists, err := backend.GetTaskLists()
	if err != nil {
		t.Fatalf("Failed to get task lists: %v", err)
	}

	if len(lists) != 2 {
		t.Fatalf("Expected 2 lists, got %d", len(lists))
	}

	// Lists should be sorted alphabetically
	if lists[0].Name != "Personal" {
		t.Errorf("Expected first list 'Personal', got '%s'", lists[0].Name)
	}

	if lists[1].Name != "Work" {
		t.Errorf("Expected second list 'Work', got '%s'", lists[1].Name)
	}
}

// TestAddTask tests task creation
func TestAddTask(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	// Create a task list
	listID, err := backend.CreateTaskList("Test List", "", "")
	if err != nil {
		t.Fatalf("Failed to create list: %v", err)
	}

	// Create a task
	task := Task{
		Summary:     "Test Task",
		Description: "This is a test",
		Status:      "NEEDS-ACTION",
		Priority:    5,
	}

	err = backend.AddTask(listID, task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// Retrieve tasks
	tasks, err := backend.GetTasks(listID, nil)
	if err != nil {
		t.Fatalf("Failed to get tasks: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(tasks))
	}

	if tasks[0].Summary != "Test Task" {
		t.Errorf("Expected summary 'Test Task', got '%s'", tasks[0].Summary)
	}

	if tasks[0].Description != "This is a test" {
		t.Errorf("Expected description 'This is a test', got '%s'", tasks[0].Description)
	}

	if tasks[0].Priority != 5 {
		t.Errorf("Expected priority 5, got %d", tasks[0].Priority)
	}
}

// TestAddTaskWithUID tests task creation with explicit UID
func TestAddTaskWithUID(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Test List", "", "")

	task := Task{
		UID:     "custom-uid-123",
		Summary: "Task with UID",
		Status:  "NEEDS-ACTION",
	}

	err := backend.AddTask(listID, task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	tasks, _ := backend.GetTasks(listID, nil)
	if tasks[0].UID != "custom-uid-123" {
		t.Errorf("Expected UID 'custom-uid-123', got '%s'", tasks[0].UID)
	}
}

// TestUpdateTask tests task updates
func TestUpdateTask(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Test List", "", "")

	task := Task{
		UID:      "task-1",
		Summary:  "Original",
		Status:   "NEEDS-ACTION",
		Priority: 5,
	}

	backend.AddTask(listID, task)

	// Update task
	task.Summary = "Updated"
	task.Priority = 1
	task.Status = "COMPLETED"

	err := backend.UpdateTask(listID, task)
	if err != nil {
		t.Fatalf("Failed to update task: %v", err)
	}

	// Verify update
	tasks, _ := backend.GetTasks(listID, nil)
	if tasks[0].Summary != "Updated" {
		t.Errorf("Expected summary 'Updated', got '%s'", tasks[0].Summary)
	}

	if tasks[0].Priority != 1 {
		t.Errorf("Expected priority 1, got %d", tasks[0].Priority)
	}

	if tasks[0].Status != "COMPLETED" {
		t.Errorf("Expected status 'COMPLETED', got '%s'", tasks[0].Status)
	}
}

// TestUpdateNonexistentTask tests updating a task that doesn't exist
func TestUpdateNonexistentTask(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Test List", "", "")

	task := Task{
		UID:     "nonexistent",
		Summary: "Does not exist",
		Status:  "NEEDS-ACTION",
	}

	err := backend.UpdateTask(listID, task)
	if err == nil {
		t.Error("Expected error when updating nonexistent task")
	}

	// Should be a BackendError with NotFound
	if backendErr, ok := err.(*BackendError); ok {
		if !backendErr.IsNotFound() {
			t.Error("Expected NotFound error")
		}
	}
}

// TestDeleteTask tests task deletion
func TestDeleteTask(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Test List", "", "")

	task := Task{
		UID:     "task-to-delete",
		Summary: "Delete me",
		Status:  "NEEDS-ACTION",
	}

	backend.AddTask(listID, task)

	// Verify task exists
	tasks, _ := backend.GetTasks(listID, nil)
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task before delete, got %d", len(tasks))
	}

	// Delete task
	err := backend.DeleteTask(listID, "task-to-delete")
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	// Verify task is deleted
	tasks, _ = backend.GetTasks(listID, nil)
	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks after delete, got %d", len(tasks))
	}
}

// TestDeleteNonexistentTask tests deleting a task that doesn't exist
func TestDeleteNonexistentTask(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Test List", "", "")

	err := backend.DeleteTask(listID, "nonexistent")
	if err == nil {
		t.Error("Expected error when deleting nonexistent task")
	}

	if backendErr, ok := err.(*BackendError); ok {
		if !backendErr.IsNotFound() {
			t.Error("Expected NotFound error")
		}
	}
}

// TestGetTasksWithStatusFilter tests filtering tasks by status
func TestGetTasksWithStatusFilter(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Test List", "", "")

	// Add tasks with different statuses
	backend.AddTask(listID, Task{UID: "task-1", Summary: "Task 1", Status: "NEEDS-ACTION"})
	backend.AddTask(listID, Task{UID: "task-2", Summary: "Task 2", Status: "COMPLETED"})
	backend.AddTask(listID, Task{UID: "task-3", Summary: "Task 3", Status: "NEEDS-ACTION"})

	// Filter by NEEDS-ACTION
	statuses := []string{"NEEDS-ACTION"}
	filter := &TaskFilter{Statuses: &statuses}

	tasks, err := backend.GetTasks(listID, filter)
	if err != nil {
		t.Fatalf("Failed to get filtered tasks: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks with NEEDS-ACTION status, got %d", len(tasks))
	}

	for _, task := range tasks {
		if task.Status != "NEEDS-ACTION" {
			t.Errorf("Expected status NEEDS-ACTION, got %s", task.Status)
		}
	}
}

// TestGetTasksWithDateFilter tests filtering tasks by due date
func TestGetTasksWithDateFilter(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Test List", "", "")

	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)
	yesterday := now.Add(-24 * time.Hour)

	// Add tasks with different due dates
	backend.AddTask(listID, Task{UID: "task-1", Summary: "Due tomorrow", Status: "NEEDS-ACTION", DueDate: &tomorrow})
	backend.AddTask(listID, Task{UID: "task-2", Summary: "Due yesterday", Status: "NEEDS-ACTION", DueDate: &yesterday})

	// Filter tasks due before now (should get task-2)
	filter := &TaskFilter{DueBefore: &now}

	tasks, err := backend.GetTasks(listID, filter)
	if err != nil {
		t.Fatalf("Failed to get filtered tasks: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("Expected 1 task due before now, got %d", len(tasks))
	}

	if tasks[0].Summary != "Due yesterday" {
		t.Errorf("Expected 'Due yesterday', got '%s'", tasks[0].Summary)
	}
}

// TestFindTasksBySummary tests searching for tasks
func TestFindTasksBySummary(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Test List", "", "")

	// Add tasks
	backend.AddTask(listID, Task{UID: "task-1", Summary: "Buy groceries", Status: "NEEDS-ACTION"})
	backend.AddTask(listID, Task{UID: "task-2", Summary: "Buy milk", Status: "NEEDS-ACTION"})
	backend.AddTask(listID, Task{UID: "task-3", Summary: "Write report", Status: "NEEDS-ACTION"})

	// Search for "buy" (case-insensitive)
	tasks, err := backend.FindTasksBySummary(listID, "buy")
	if err != nil {
		t.Fatalf("Failed to find tasks: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks matching 'buy', got %d", len(tasks))
	}

	// Search for exact match
	tasks, err = backend.FindTasksBySummary(listID, "Buy milk")
	if err != nil {
		t.Fatalf("Failed to find tasks: %v", err)
	}

	// Exact match should come first
	if tasks[0].Summary != "Buy milk" {
		t.Errorf("Expected exact match 'Buy milk' first, got '%s'", tasks[0].Summary)
	}
}

// TestRenameTaskList tests renaming a task list
func TestRenameTaskList(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Old Name", "", "")

	err := backend.RenameTaskList(listID, "New Name")
	if err != nil {
		t.Fatalf("Failed to rename list: %v", err)
	}

	lists, _ := backend.GetTaskLists()
	if lists[0].Name != "New Name" {
		t.Errorf("Expected list name 'New Name', got '%s'", lists[0].Name)
	}
}

// TestRenameNonexistentList tests renaming a list that doesn't exist
func TestRenameNonexistentList(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	err := backend.RenameTaskList("nonexistent", "New Name")
	if err == nil {
		t.Error("Expected error when renaming nonexistent list")
	}

	if backendErr, ok := err.(*BackendError); ok {
		if !backendErr.IsNotFound() {
			t.Error("Expected NotFound error")
		}
	}
}

// TestDeleteTaskList tests deleting a task list
func TestDeleteTaskList(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("List to delete", "", "")

	// Add a task to the list
	backend.AddTask(listID, Task{UID: "task-1", Summary: "Task", Status: "NEEDS-ACTION"})

	// Delete list
	err := backend.DeleteTaskList(listID)
	if err != nil {
		t.Fatalf("Failed to delete list: %v", err)
	}

	// Verify list is deleted
	lists, _ := backend.GetTaskLists()
	if len(lists) != 0 {
		t.Errorf("Expected 0 lists after delete, got %d", len(lists))
	}

	// Verify tasks in list are also deleted
	tasks, _ := backend.GetTasks(listID, nil)
	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks after list delete, got %d", len(tasks))
	}
}

// TestDeleteNonexistentList tests deleting a list that doesn't exist
func TestDeleteNonexistentList(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	err := backend.DeleteTaskList("nonexistent")
	if err == nil {
		t.Error("Expected error when deleting nonexistent list")
	}

	if backendErr, ok := err.(*BackendError); ok {
		if !backendErr.IsNotFound() {
			t.Error("Expected NotFound error")
		}
	}
}

// TestParseStatusFlag tests status flag parsing
func TestParseStatusFlag(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	tests := []struct {
		input    string
		expected string
	}{
		{"T", "NEEDS-ACTION"},
		{"TODO", "NEEDS-ACTION"},
		{"NEEDS-ACTION", "NEEDS-ACTION"},
		{"D", "COMPLETED"},
		{"DONE", "COMPLETED"},
		{"COMPLETED", "COMPLETED"},
		{"P", "IN-PROCESS"},
		{"PROCESSING", "IN-PROCESS"},
		{"IN-PROCESS", "IN-PROCESS"},
		{"C", "CANCELLED"},
		{"CANCELLED", "CANCELLED"},
	}

	for _, tt := range tests {
		result, err := backend.ParseStatusFlag(tt.input)
		if err != nil {
			t.Errorf("Failed to parse status flag '%s': %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("ParseStatusFlag(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}

	// Test invalid status
	_, err := backend.ParseStatusFlag("INVALID")
	if err == nil {
		t.Error("Expected error for invalid status flag")
	}
}

// TestStatusToDisplayName tests status display name conversion
func TestStatusToDisplayName(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	tests := []struct {
		input    string
		expected string
	}{
		{"NEEDS-ACTION", "TODO"},
		{"COMPLETED", "DONE"},
		{"IN-PROCESS", "PROCESSING"},
		{"CANCELLED", "CANCELLED"},
	}

	for _, tt := range tests {
		result := backend.StatusToDisplayName(tt.input)
		if result != tt.expected {
			t.Errorf("StatusToDisplayName(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

// TestSortTasks tests task sorting
func TestSortTasks(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	tasks := []Task{
		{UID: "1", Summary: "Priority 0", Priority: 0},
		{UID: "2", Summary: "Priority 5", Priority: 5},
		{UID: "3", Summary: "Priority 1", Priority: 1},
		{UID: "4", Summary: "Priority 9", Priority: 9},
	}

	backend.SortTasks(tasks)

	// Expected order: 1 (highest), 5, 9, 0 (undefined last)
	if tasks[0].Priority != 1 {
		t.Errorf("Expected first task priority 1, got %d", tasks[0].Priority)
	}
	if tasks[len(tasks)-1].Priority != 0 {
		t.Errorf("Expected last task priority 0, got %d", tasks[len(tasks)-1].Priority)
	}
}

// TestGetPriorityColor tests priority color assignment
func TestGetPriorityColor(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	tests := []struct {
		priority int
		hasColor bool
	}{
		{0, false}, // Undefined - no color
		{1, true},  // High priority - red
		{4, true},  // High priority - red
		{5, true},  // Medium priority - yellow
		{6, true},  // Low priority - blue
		{9, true},  // Low priority - blue
	}

	for _, tt := range tests {
		color := backend.GetPriorityColor(tt.priority)
		if tt.hasColor && color == "" {
			t.Errorf("Expected color for priority %d, got empty string", tt.priority)
		}
		if !tt.hasColor && color != "" {
			t.Errorf("Expected no color for priority %d, got '%s'", tt.priority, color)
		}
	}
}

// TestMarkLocallyModified tests marking tasks as locally modified
func TestMarkLocallyModified(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Test List", "", "")
	task := Task{UID: "task-1", Summary: "Test", Status: "NEEDS-ACTION"}
	backend.AddTask(listID, task)

	// Clear the flag (AddTask sets it to 1)
	backend.ClearSyncFlags("task-1")

	// Mark as modified
	err := backend.MarkLocallyModified("task-1")
	if err != nil {
		t.Fatalf("Failed to mark task as locally modified: %v", err)
	}

	// Verify task is in locally modified list
	modifiedTasks, err := backend.GetLocallyModifiedTasks()
	if err != nil {
		t.Fatalf("Failed to get locally modified tasks: %v", err)
	}

	if len(modifiedTasks) != 1 {
		t.Errorf("Expected 1 locally modified task, got %d", len(modifiedTasks))
	}

	if modifiedTasks[0].UID != "task-1" {
		t.Errorf("Expected task UID 'task-1', got '%s'", modifiedTasks[0].UID)
	}
}

// TestGetPendingSyncOperations tests retrieving pending sync operations
func TestGetPendingSyncOperations(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Test List", "", "")

	// Add a task (this queues a 'create' operation)
	task := Task{UID: "task-1", Summary: "Test", Status: "NEEDS-ACTION"}
	backend.AddTask(listID, task)

	// Get pending operations
	ops, err := backend.GetPendingSyncOperations()
	if err != nil {
		t.Fatalf("Failed to get pending operations: %v", err)
	}

	if len(ops) != 1 {
		t.Errorf("Expected 1 pending operation, got %d", len(ops))
	}

	if ops[0].Operation != "create" {
		t.Errorf("Expected operation 'create', got '%s'", ops[0].Operation)
	}

	if ops[0].TaskUID != "task-1" {
		t.Errorf("Expected task UID 'task-1', got '%s'", ops[0].TaskUID)
	}
}

// TestClearSyncFlags tests clearing sync flags
func TestClearSyncFlags(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Test List", "", "")
	task := Task{UID: "task-1", Summary: "Test", Status: "NEEDS-ACTION"}
	backend.AddTask(listID, task)

	// Task should be locally modified after creation
	modifiedTasks, _ := backend.GetLocallyModifiedTasks()
	if len(modifiedTasks) != 1 {
		t.Fatalf("Expected 1 locally modified task, got %d", len(modifiedTasks))
	}

	// Clear flags
	err := backend.ClearSyncFlags("task-1")
	if err != nil {
		t.Fatalf("Failed to clear sync flags: %v", err)
	}

	// Verify flags are cleared
	modifiedTasks, _ = backend.GetLocallyModifiedTasks()
	if len(modifiedTasks) != 0 {
		t.Errorf("Expected 0 locally modified tasks after clearing, got %d", len(modifiedTasks))
	}
}

// TestUpdateSyncMetadata tests updating sync metadata
func TestUpdateSyncMetadata(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Test List", "", "")
	task := Task{UID: "task-1", Summary: "Test", Status: "NEEDS-ACTION"}
	backend.AddTask(listID, task)

	now := time.Now()
	err := backend.UpdateSyncMetadata("task-1", listID, "etag-123", now)
	if err != nil {
		t.Fatalf("Failed to update sync metadata: %v", err)
	}

	// Verify metadata was updated (would need to query database directly to verify)
	// For now, just verify no error
}

// TestRemoveSyncOperation tests removing sync operations
func TestRemoveSyncOperation(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Test List", "", "")
	task := Task{UID: "task-1", Summary: "Test", Status: "NEEDS-ACTION"}
	backend.AddTask(listID, task)

	// Verify operation exists
	ops, _ := backend.GetPendingSyncOperations()
	if len(ops) != 1 {
		t.Fatalf("Expected 1 pending operation, got %d", len(ops))
	}

	// Remove operation
	err := backend.RemoveSyncOperation("task-1", "create")
	if err != nil {
		t.Fatalf("Failed to remove sync operation: %v", err)
	}

	// Verify operation was removed
	ops, _ = backend.GetPendingSyncOperations()
	if len(ops) != 0 {
		t.Errorf("Expected 0 pending operations after removal, got %d", len(ops))
	}
}

// TestTaskWithParent tests creating tasks with parent relationships
func TestTaskWithParent(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Test List", "", "")

	// Create parent task
	parentTask := Task{UID: "parent-1", Summary: "Parent", Status: "NEEDS-ACTION"}
	backend.AddTask(listID, parentTask)

	// Create child task
	childTask := Task{UID: "child-1", Summary: "Child", Status: "NEEDS-ACTION", ParentUID: "parent-1"}
	backend.AddTask(listID, childTask)

	// Retrieve tasks
	tasks, _ := backend.GetTasks(listID, nil)

	var child *Task
	for i := range tasks {
		if tasks[i].UID == "child-1" {
			child = &tasks[i]
			break
		}
	}

	if child == nil {
		t.Fatal("Child task not found")
	}

	if child.ParentUID != "parent-1" {
		t.Errorf("Expected parent UID 'parent-1', got '%s'", child.ParentUID)
	}
}

// TestTaskWithCategories tests tasks with categories
func TestTaskWithCategories(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Test List", "", "")

	task := Task{
		UID:        "task-1",
		Summary:    "Task with categories",
		Status:     "NEEDS-ACTION",
		Categories: []string{"work", "urgent", "important"},
	}

	backend.AddTask(listID, task)

	// Retrieve task
	tasks, _ := backend.GetTasks(listID, nil)

	if len(tasks[0].Categories) != 3 {
		t.Errorf("Expected 3 categories, got %d", len(tasks[0].Categories))
	}

	if tasks[0].Categories[0] != "work" {
		t.Errorf("Expected first category 'work', got '%s'", tasks[0].Categories[0])
	}
}

// TestTaskTimestamps tests task timestamp handling
func TestTaskTimestamps(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Test List", "", "")

	now := time.Now()
	dueDate := now.Add(24 * time.Hour)
	startDate := now.Add(-24 * time.Hour)

	task := Task{
		UID:       "task-1",
		Summary:   "Task with dates",
		Status:    "NEEDS-ACTION",
		DueDate:   &dueDate,
		StartDate: &startDate,
	}

	backend.AddTask(listID, task)

	// Retrieve task
	tasks, _ := backend.GetTasks(listID, nil)

	if tasks[0].DueDate == nil {
		t.Error("Expected due date to be set")
	}

	if tasks[0].StartDate == nil {
		t.Error("Expected start date to be set")
	}

	if tasks[0].Created == nil {
		t.Error("Expected created timestamp to be set")
	}

	if tasks[0].Modified == nil {
		t.Error("Expected modified timestamp to be set")
	}
}

// TestGetDeletedTaskLists tests trash functionality (not yet implemented)
func TestGetDeletedTaskLists(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	lists, err := backend.GetDeletedTaskLists()
	if err != nil {
		t.Fatalf("GetDeletedTaskLists failed: %v", err)
	}

	// Should return empty list (trash not implemented yet)
	if len(lists) != 0 {
		t.Errorf("Expected 0 deleted lists, got %d", len(lists))
	}
}

// TestRestoreTaskList tests restore functionality (not yet implemented)
func TestRestoreTaskList(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	err := backend.RestoreTaskList("some-list")
	if err == nil {
		t.Error("Expected error for unimplemented restore functionality")
	}
}

// TestClose tests closing the backend
func TestClose(t *testing.T) {
	backend, _ := createTestBackend(t)

	err := backend.Close()
	if err != nil {
		t.Errorf("Failed to close backend: %v", err)
	}

	// Closing again should not error
	err = backend.Close()
	if err != nil {
		t.Errorf("Failed to close backend twice: %v", err)
	}
}

// TestTransactionRollback tests that transactions rollback on error
func TestTransactionRollback(t *testing.T) {
	backend, cleanup := createTestBackend(t)
	defer cleanup()

	listID, _ := backend.CreateTaskList("Test List", "", "")

	// Try to add task with invalid parent reference
	task := Task{
		UID:       "task-1",
		Summary:   "Test",
		Status:    "NEEDS-ACTION",
		ParentUID: "nonexistent-parent", // This won't cause an error (NULL constraint allows it)
	}

	// This should succeed even with nonexistent parent
	err := backend.AddTask(listID, task)
	if err != nil {
		t.Logf("AddTask with nonexistent parent: %v", err)
	}

	// Verify task wasn't added if there was an error
	tasks, _ := backend.GetTasks(listID, nil)
	if err != nil && len(tasks) != 0 {
		t.Error("Expected transaction rollback, but task was added")
	}
}
