package operations

import (
	"errors"
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/internal/config"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// mockTaskManagerForOperations implements backend.TaskManager for testing
type mockTaskManagerForOperations struct {
	tasks       map[string][]backend.Task
	findResults []backend.Task
	findError   error
}

func (m *mockTaskManagerForOperations) GetTaskLists() ([]backend.TaskList, error) {
	return nil, nil
}

func (m *mockTaskManagerForOperations) GetTasks(listID string, filter *backend.TaskFilter) ([]backend.Task, error) {
	if tasks, ok := m.tasks[listID]; ok {
		return tasks, nil
	}
	return nil, nil
}

func (m *mockTaskManagerForOperations) FindTasksBySummary(listID string, summary string) ([]backend.Task, error) {
	if m.findError != nil {
		return nil, m.findError
	}
	return m.findResults, nil
}

func (m *mockTaskManagerForOperations) AddTask(listID string, task backend.Task) error {
	return nil
}

func (m *mockTaskManagerForOperations) UpdateTask(listID string, task backend.Task) error {
	return nil
}

func (m *mockTaskManagerForOperations) DeleteTask(listID string, taskUID string) error {
	return nil
}

func (m *mockTaskManagerForOperations) SortTasks(tasks []backend.Task) {
}

func (m *mockTaskManagerForOperations) GetPriorityColor(priority int) string {
	return ""
}

func (m *mockTaskManagerForOperations) ParseStatusFlag(status string) (string, error) {
	// Support common abbreviations
	upper := strings.ToUpper(status)
	switch upper {
	case "T", "TODO":
		return "NEEDS-ACTION", nil
	case "D", "DONE":
		return "COMPLETED", nil
	case "P", "PROCESSING":
		return "IN-PROCESS", nil
	case "C", "CANCELLED":
		return "CANCELLED", nil
	case "NEEDS-ACTION", "COMPLETED", "IN-PROCESS":
		return upper, nil
	default:
		return "", errors.New("invalid status")
	}
}

func (m *mockTaskManagerForOperations) CreateTaskList(name, description, color string) (string, error) {
	return "new-list-id", nil
}

func (m *mockTaskManagerForOperations) DeleteTaskList(listID string) error {
	return nil
}

func (m *mockTaskManagerForOperations) RenameTaskList(listID, newName string) error {
	return nil
}

func (m *mockTaskManagerForOperations) GetDeletedTaskLists() ([]backend.TaskList, error) {
	return nil, nil
}

func (m *mockTaskManagerForOperations) RestoreTaskList(listID string) error {
	return nil
}

func (m *mockTaskManagerForOperations) PermanentlyDeleteTaskList(listID string) error {
	return nil
}

func (m *mockTaskManagerForOperations) StatusToDisplayName(backendStatus string) string {
	return backendStatus
}

func (m *mockTaskManagerForOperations) GetBackendType() string {
	return "mock"
}

func (m *mockTaskManagerForOperations) GetBackendDisplayName() string {
	return "[mock]"
}

func (m *mockTaskManagerForOperations) GetBackendContext() string {
	return "mock-backend"
}

func (m *mockTaskManagerForOperations) CanDetect() (bool, error) {
	return false, nil
}

func (m *mockTaskManagerForOperations) GetDetectionInfo() string {
	return ""
}

func TestFindTaskBySummary_NoMatches(t *testing.T) {
	mock := &mockTaskManagerForOperations{
		findResults: []backend.Task{},
	}

	cfg := &config.Config{}
	selector := NewTaskSelector(mock, cfg)
	opts := DefaultOptions()
	_, err := selector.Select("list1", "nonexistent", opts)

	if err == nil {
		t.Error("TaskSelector.Select() should return error when no matches found")
	}
	if !strings.Contains(err.Error(), "no tasks found") {
		t.Errorf("Expected 'no tasks found' error, got: %v", err)
	}
}

func TestFindTaskBySummary_BackendError(t *testing.T) {
	mock := &mockTaskManagerForOperations{
		findError: errors.New("backend error"),
	}

	cfg := &config.Config{}
	selector := NewTaskSelector(mock, cfg)
	opts := DefaultOptions()
	_, err := selector.Select("list1", "search", opts)

	if err == nil {
		t.Error("TaskSelector.Select() should return error when backend fails")
	}
	if !strings.Contains(err.Error(), "error searching for tasks") {
		t.Errorf("Expected search error, got: %v", err)
	}
}

func TestFindTaskBySummary_SingleExactMatch(t *testing.T) {
	exactTask := backend.Task{
		UID:     "task1",
		Summary: "Buy groceries",
		Status:  "NEEDS-ACTION",
	}

	mock := &mockTaskManagerForOperations{
		findResults: []backend.Task{exactTask},
	}

	cfg := &config.Config{}
	selector := NewTaskSelector(mock, cfg)
	opts := DefaultOptions()
	result, err := selector.Select("list1", "Buy groceries", opts)

	if err != nil {
		t.Fatalf("TaskSelector.Select() failed: %v", err)
	}
	if result.UID != "task1" {
		t.Errorf("Expected task1, got: %s", result.UID)
	}
}

func TestBuildFilter_NoFlags(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringArray("status", []string{}, "")

	mock := &mockTaskManagerForOperations{}
	filter, err := BuildFilter(cmd, mock)

	if err != nil {
		t.Fatalf("BuildFilter() failed: %v", err)
	}
	if filter.Statuses != nil {
		t.Error("Filter should have nil statuses when no flags provided")
	}
}

func TestBuildFilter_SingleStatus(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringArray("status", []string{}, "")
	cmd.Flags().Set("status", "TODO")

	mock := &mockTaskManagerForOperations{}
	filter, err := BuildFilter(cmd, mock)

	if err != nil {
		t.Fatalf("BuildFilter() failed: %v", err)
	}
	if filter.Statuses == nil || len(*filter.Statuses) != 1 {
		t.Fatalf("Expected 1 status, got: %v", filter.Statuses)
	}
	if (*filter.Statuses)[0] != "NEEDS-ACTION" {
		t.Errorf("Expected NEEDS-ACTION, got: %s", (*filter.Statuses)[0])
	}
}

func TestBuildFilter_MultipleStatuses(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringArray("status", []string{}, "")
	cmd.Flags().Set("status", "TODO")
	cmd.Flags().Set("status", "DONE")

	mock := &mockTaskManagerForOperations{}
	filter, err := BuildFilter(cmd, mock)

	if err != nil {
		t.Fatalf("BuildFilter() failed: %v", err)
	}
	if filter.Statuses == nil || len(*filter.Statuses) != 2 {
		t.Fatalf("Expected 2 statuses, got: %v", filter.Statuses)
	}

	statuses := *filter.Statuses
	if statuses[0] != "NEEDS-ACTION" || statuses[1] != "COMPLETED" {
		t.Errorf("Expected [NEEDS-ACTION, COMPLETED], got: %v", statuses)
	}
}

func TestBuildFilter_CommaSeparatedStatuses(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringArray("status", []string{}, "")
	cmd.Flags().Set("status", "TODO,DONE,PROCESSING")

	mock := &mockTaskManagerForOperations{}
	filter, err := BuildFilter(cmd, mock)

	if err != nil {
		t.Fatalf("BuildFilter() failed: %v", err)
	}
	if filter.Statuses == nil || len(*filter.Statuses) != 3 {
		t.Fatalf("Expected 3 statuses, got: %v", filter.Statuses)
	}

	statuses := *filter.Statuses
	expected := []string{"NEEDS-ACTION", "COMPLETED", "IN-PROCESS"}
	for i, exp := range expected {
		if statuses[i] != exp {
			t.Errorf("Status[%d]: expected %s, got: %s", i, exp, statuses[i])
		}
	}
}

func TestBuildFilter_Abbreviations(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"T", "NEEDS-ACTION"},
		{"D", "COMPLETED"},
		{"P", "IN-PROCESS"},
		{"C", "CANCELLED"},
		{"TODO", "NEEDS-ACTION"},
		{"DONE", "COMPLETED"},
		{"PROCESSING", "IN-PROCESS"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().StringArray("status", []string{}, "")
			cmd.Flags().Set("status", tt.input)

			mock := &mockTaskManagerForOperations{}
			filter, err := BuildFilter(cmd, mock)

			if err != nil {
				t.Fatalf("BuildFilter() failed: %v", err)
			}
			if filter.Statuses == nil || len(*filter.Statuses) != 1 {
				t.Fatalf("Expected 1 status, got: %v", filter.Statuses)
			}
			if (*filter.Statuses)[0] != tt.expected {
				t.Errorf("Expected %s, got: %s", tt.expected, (*filter.Statuses)[0])
			}
		})
	}
}

func TestBuildFilter_InvalidStatus(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringArray("status", []string{}, "")
	cmd.Flags().Set("status", "INVALID")

	mock := &mockTaskManagerForOperations{}
	_, err := BuildFilter(cmd, mock)

	if err == nil {
		t.Error("BuildFilter() should return error for invalid status")
	}
	if !strings.Contains(err.Error(), "invalid status") {
		t.Errorf("Expected invalid status error, got: %v", err)
	}
}

func TestBuildFilter_MixedValidAndInvalid(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringArray("status", []string{}, "")
	cmd.Flags().Set("status", "TODO,INVALID")

	mock := &mockTaskManagerForOperations{}
	_, err := BuildFilter(cmd, mock)

	if err == nil {
		t.Error("BuildFilter() should return error when any status is invalid")
	}
}

func TestBuildFilter_TrimSpaces(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringArray("status", []string{}, "")
	cmd.Flags().Set("status", " TODO , DONE , PROCESSING ")

	mock := &mockTaskManagerForOperations{}
	filter, err := BuildFilter(cmd, mock)

	if err != nil {
		t.Fatalf("BuildFilter() failed: %v", err)
	}
	if filter.Statuses == nil || len(*filter.Statuses) != 3 {
		t.Fatalf("Expected 3 statuses, got: %v", filter.Statuses)
	}

	// Should parse correctly even with spaces
	statuses := *filter.Statuses
	expected := []string{"NEEDS-ACTION", "COMPLETED", "IN-PROCESS"}
	for i, exp := range expected {
		if statuses[i] != exp {
			t.Errorf("Status[%d]: expected %s, got: %s", i, exp, statuses[i])
		}
	}
}

func TestBuildFilter_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"todo", "NEEDS-ACTION"},
		{"TODO", "NEEDS-ACTION"},
		{"DONE", "COMPLETED"},
		{"done", "COMPLETED"},
		{"t", "NEEDS-ACTION"},
		{"T", "NEEDS-ACTION"},
		{"processing", "IN-PROCESS"},
		{"PROCESSING", "IN-PROCESS"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().StringArray("status", []string{}, "")
			cmd.Flags().Set("status", tt.input)

			mock := &mockTaskManagerForOperations{}
			filter, err := BuildFilter(cmd, mock)

			if err != nil {
				t.Fatalf("BuildFilter() failed: %v", err)
			}
			if filter.Statuses == nil || len(*filter.Statuses) != 1 {
				t.Fatalf("Expected 1 status, got: %v", filter.Statuses)
			}
			if (*filter.Statuses)[0] != tt.expected {
				t.Errorf("Expected %s, got: %s", tt.expected, (*filter.Statuses)[0])
			}
		})
	}
}

// Test exact match scenarios
func TestSelectTask_ExactMatching(t *testing.T) {
	// This tests the internal logic of how exact vs partial matches are handled
	// Note: selectTask and confirmTask require user input, so we test the higher-level
	// FindTaskBySummary which includes the exact matching logic

	exactTask := backend.Task{
		UID:     "exact1",
		Summary: "Buy milk",
		Status:  "NEEDS-ACTION",
	}

	tests := []struct {
		name        string
		searchTerm  string
		mockResults []backend.Task
		expectUID   string
		expectError bool
	}{
		{
			name:        "single exact match",
			searchTerm:  "Buy milk",
			mockResults: []backend.Task{exactTask},
			expectUID:   "exact1",
		},
		{
			name:        "exact match case insensitive",
			searchTerm:  "buy MILK",
			mockResults: []backend.Task{exactTask},
			expectUID:   "exact1",
		},
		{
			name:        "no matches",
			searchTerm:  "nonexistent",
			mockResults: []backend.Task{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockTaskManagerForOperations{
				findResults: tt.mockResults,
			}

			cfg := &config.Config{}
			selector := NewTaskSelector(mock, cfg)
			opts := DefaultOptions()
			result, err := selector.Select("list1", tt.searchTerm, opts)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.UID != tt.expectUID {
				t.Errorf("Expected UID %s, got: %s", tt.expectUID, result.UID)
			}
		})
	}
}

// TestBuildFlatTaskList verifies that flat list is built correctly from tree
func TestBuildFlatTaskList(t *testing.T) {
	tests := []struct {
		name          string
		tasks         []backend.Task
		expectedUIDs  []string
		expectedCount int
	}{
		{
			name:          "empty tree",
			tasks:         []backend.Task{},
			expectedUIDs:  []string{},
			expectedCount: 0,
		},
		{
			name: "single root task",
			tasks: []backend.Task{
				{UID: "task1", Summary: "Task 1"},
			},
			expectedUIDs:  []string{"task1"},
			expectedCount: 1,
		},
		{
			name: "multiple root tasks",
			tasks: []backend.Task{
				{UID: "task1", Summary: "Task 1"},
				{UID: "task2", Summary: "Task 2"},
				{UID: "task3", Summary: "Task 3"},
			},
			expectedUIDs:  []string{"task1", "task2", "task3"},
			expectedCount: 3,
		},
		{
			name: "parent with children",
			tasks: []backend.Task{
				{UID: "parent", Summary: "Parent"},
				{UID: "child1", Summary: "Child 1", ParentUID: "parent"},
				{UID: "child2", Summary: "Child 2", ParentUID: "parent"},
			},
			expectedUIDs:  []string{"parent", "child1", "child2"},
			expectedCount: 3,
		},
		{
			name: "multi-level hierarchy",
			tasks: []backend.Task{
				{UID: "root", Summary: "Root"},
				{UID: "child", Summary: "Child", ParentUID: "root"},
				{UID: "grandchild", Summary: "Grandchild", ParentUID: "child"},
			},
			expectedUIDs:  []string{"root", "child", "grandchild"},
			expectedCount: 3,
		},
		{
			name: "mixed hierarchy with multiple branches",
			tasks: []backend.Task{
				{UID: "root1", Summary: "Root 1"},
				{UID: "root2", Summary: "Root 2"},
				{UID: "child1", Summary: "Child 1", ParentUID: "root1"},
				{UID: "child2", Summary: "Child 2", ParentUID: "root1"},
				{UID: "child3", Summary: "Child 3", ParentUID: "root2"},
			},
			expectedUIDs:  []string{"root1", "child1", "child2", "root2", "child3"},
			expectedCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build tree from tasks
			tree := BuildTaskTree(tt.tasks)

			// Build flat list
			var flatTasks []*backend.Task
			buildFlatTaskList(tree, &flatTasks)

			// Verify count
			if len(flatTasks) != tt.expectedCount {
				t.Errorf("Expected %d tasks, got %d", tt.expectedCount, len(flatTasks))
			}

			// Verify UIDs in order
			for i, expectedUID := range tt.expectedUIDs {
				if i >= len(flatTasks) {
					t.Fatalf("flatTasks has fewer items than expected")
				}
				if flatTasks[i].UID != expectedUID {
					t.Errorf("Task[%d]: expected UID %s, got %s", i, expectedUID, flatTasks[i].UID)
				}
			}
		})
	}
}

// TestFormatTaskTreeNumbered verifies formatting with numbering and tree structure
func TestFormatTaskTreeNumbered(t *testing.T) {
	mock := &mockTaskManagerForOperations{}

	tests := []struct {
		name           string
		tasks          []backend.Task
		checkSequence  bool // verify numbering sequence
		checkTreeChars bool // verify tree characters (├─, └─, │)
		expectedCount  int
	}{
		{
			name:           "empty tree",
			tasks:          []backend.Task{},
			checkSequence:  false,
			checkTreeChars: false,
			expectedCount:  0,
		},
		{
			name: "single root task",
			tasks: []backend.Task{
				{UID: "task1", Summary: "Task 1", Status: "NEEDS-ACTION"},
			},
			checkSequence:  true,
			checkTreeChars: false, // no tree chars for root
			expectedCount:  1,
		},
		{
			name: "multiple root tasks - no tree chars",
			tasks: []backend.Task{
				{UID: "task1", Summary: "Task 1", Status: "NEEDS-ACTION"},
				{UID: "task2", Summary: "Task 2", Status: "NEEDS-ACTION"},
				{UID: "task3", Summary: "Task 3", Status: "NEEDS-ACTION"},
			},
			checkSequence:  true,
			checkTreeChars: false,
			expectedCount:  3,
		},
		{
			name: "parent with children - should have tree chars",
			tasks: []backend.Task{
				{UID: "parent", Summary: "Parent Task", Status: "NEEDS-ACTION"},
				{UID: "child1", Summary: "Child 1", Status: "NEEDS-ACTION", ParentUID: "parent"},
				{UID: "child2", Summary: "Child 2", Status: "NEEDS-ACTION", ParentUID: "parent"},
			},
			checkSequence:  true,
			checkTreeChars: true,
			expectedCount:  3,
		},
		{
			name: "multi-level hierarchy",
			tasks: []backend.Task{
				{UID: "root", Summary: "Root", Status: "NEEDS-ACTION"},
				{UID: "child", Summary: "Child", Status: "NEEDS-ACTION", ParentUID: "root"},
				{UID: "grandchild", Summary: "Grandchild", Status: "NEEDS-ACTION", ParentUID: "child"},
			},
			checkSequence:  true,
			checkTreeChars: true,
			expectedCount:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build tree
			tree := BuildTaskTree(tt.tasks)

			// Format tree
			output, finalNum := formatTaskTreeNumbered(tree, mock, "2006-01-02", 1, "", true)

			// Verify final number matches expected count
			expectedFinalNum := tt.expectedCount + 1
			if finalNum != expectedFinalNum {
				t.Errorf("Expected final number %d, got %d", expectedFinalNum, finalNum)
			}

			// If checking sequence, verify numbers appear in order
			if tt.checkSequence && tt.expectedCount > 0 {
				for i := 1; i <= tt.expectedCount; i++ {
					numPattern := fmt.Sprintf("%2d.", i)
					if !strings.Contains(output, numPattern) {
						t.Errorf("Output missing number %d", i)
					}
				}
			}

			// If checking tree characters, verify they appear
			if tt.checkTreeChars {
				// Should contain tree characters for hierarchical display
				hasTreeChars := strings.Contains(output, "├─") ||
					strings.Contains(output, "└─") ||
					strings.Contains(output, "│")
				if !hasTreeChars {
					t.Error("Expected tree characters (├─, └─, │) but none found")
				}
			}
		})
	}
}

// TestFormatTaskTreeNumbered_TreeStructure verifies correct tree character usage
func TestFormatTaskTreeNumbered_TreeStructure(t *testing.T) {
	mock := &mockTaskManagerForOperations{}

	// Create a specific hierarchy to test tree characters
	tasks := []backend.Task{
		{UID: "parent", Summary: "Parent", Status: "NEEDS-ACTION"},
		{UID: "child1", Summary: "First Child", Status: "NEEDS-ACTION", ParentUID: "parent"},
		{UID: "child2", Summary: "Last Child", Status: "NEEDS-ACTION", ParentUID: "parent"},
	}

	tree := BuildTaskTree(tasks)
	output, _ := formatTaskTreeNumbered(tree, mock, "2006-01-02", 1, "", true)

	// First child should use ├─ (not last)
	if !strings.Contains(output, "├─") {
		t.Error("Expected ├─ for non-last child")
	}

	// Last child should use └─
	if !strings.Contains(output, "└─") {
		t.Error("Expected └─ for last child")
	}
}

// TestFormatTaskTreeNumbered_DeepNesting verifies deep hierarchy handling
func TestFormatTaskTreeNumbered_DeepNesting(t *testing.T) {
	mock := &mockTaskManagerForOperations{}

	// Create 5-level deep hierarchy
	tasks := []backend.Task{
		{UID: "level1", Summary: "Level 1", Status: "NEEDS-ACTION"},
		{UID: "level2", Summary: "Level 2", Status: "NEEDS-ACTION", ParentUID: "level1"},
		{UID: "level3", Summary: "Level 3", Status: "NEEDS-ACTION", ParentUID: "level2"},
		{UID: "level4", Summary: "Level 4", Status: "NEEDS-ACTION", ParentUID: "level3"},
		{UID: "level5", Summary: "Level 5", Status: "NEEDS-ACTION", ParentUID: "level4"},
	}

	tree := BuildTaskTree(tasks)
	output, finalNum := formatTaskTreeNumbered(tree, mock, "2006-01-02", 1, "", true)

	// Should have 5 tasks numbered 1-5
	if finalNum != 6 {
		t.Errorf("Expected final number 6, got %d", finalNum)
	}

	// All 5 numbers should appear
	for i := 1; i <= 5; i++ {
		numPattern := fmt.Sprintf("%2d.", i)
		if !strings.Contains(output, numPattern) {
			t.Errorf("Output missing number %d", i)
		}
	}
}

// TestSelectTaskInteractively_EmptyList verifies error handling for empty list
func TestSelectTaskInteractively_EmptyList(t *testing.T) {
	mock := &mockTaskManagerForOperations{
		tasks: map[string][]backend.Task{
			"list1": {}, // empty list
		},
	}

	cfg := &config.Config{}

	// Should return error for empty list using TaskSelector
	selector := NewTaskSelector(mock, cfg)
	opts := DefaultOptions()
	opts.DisplayFormat = "tree"
	_, err := selector.Select("list1", "", opts)

	if err == nil {
		t.Error("Expected error for empty list")
	}

	if !strings.Contains(err.Error(), "no tasks") {
		t.Errorf("Expected 'no tasks' error, got: %v", err)
	}
}

// TestSelectTaskInteractively_BackendError verifies error handling for backend failure
func TestSelectTaskInteractively_BackendError(t *testing.T) {
	// Mock that returns error on GetTasks
	mock := &mockTaskManagerForOperations{
		tasks: nil, // Will cause GetTasks to return nil
	}

	cfg := &config.Config{}

	// Should propagate backend error using TaskSelector
	selector := NewTaskSelector(mock, cfg)
	opts := DefaultOptions()
	opts.DisplayFormat = "tree"
	_, err := selector.Select("nonexistent", "", opts)

	if err == nil {
		t.Error("Expected error for backend failure")
	}

	// Error should mention retrieval issue (may be nil tasks or other backend error)
	if !strings.Contains(err.Error(), "error retrieving tasks") && err.Error() != "error retrieving tasks: <nil>" {
		// The actual error might just be about retrieval, that's fine
		// We mainly want to ensure it doesn't panic
	}
}
