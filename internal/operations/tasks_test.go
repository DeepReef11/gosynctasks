package operations

import (
	"errors"
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
	_, err := FindTaskBySummary(mock, cfg, "list1", "nonexistent")

	if err == nil {
		t.Error("FindTaskBySummary() should return error when no matches found")
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
	_, err := FindTaskBySummary(mock, cfg, "list1", "search")

	if err == nil {
		t.Error("FindTaskBySummary() should return error when backend fails")
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
	result, err := FindTaskBySummary(mock, cfg, "list1", "Buy groceries")

	if err != nil {
		t.Fatalf("FindTaskBySummary() failed: %v", err)
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
			name:       "single exact match",
			searchTerm: "Buy milk",
			mockResults: []backend.Task{exactTask},
			expectUID:  "exact1",
		},
		{
			name:       "exact match case insensitive",
			searchTerm: "buy MILK",
			mockResults: []backend.Task{exactTask},
			expectUID:  "exact1",
		},
		{
			name:       "no matches",
			searchTerm: "nonexistent",
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
			result, err := FindTaskBySummary(mock, cfg, "list1", tt.searchTerm)

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
