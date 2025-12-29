package views

import (
	"gosynctasks/backend"
	"testing"
)

func TestExcludeStatusFilter(t *testing.T) {
	tasks := []backend.Task{
		{UID: "1", Summary: "Todo task", Status: "TODO"},
		{UID: "2", Summary: "Done task", Status: "DONE"},
		{UID: "3", Summary: "Processing task", Status: "PROCESSING"},
		{UID: "4", Summary: "Completed task", Status: "COMPLETED"},
		{UID: "5", Summary: "Cancelled task", Status: "CANCELLED"},
	}

	filters := &ViewFilters{
		ExcludeStatuses: []string{"DONE", "COMPLETED", "CANCELLED"},
	}

	filtered := ApplyFilters(tasks, filters)

	// Should only have TODO and PROCESSING tasks
	if len(filtered) != 2 {
		t.Errorf("Expected 2 tasks after filtering, got %d", len(filtered))
	}

	for _, task := range filtered {
		if task.Status == "DONE" || task.Status == "COMPLETED" || task.Status == "CANCELLED" {
			t.Errorf("Task with status %s should have been filtered out", task.Status)
		}
	}

	// Verify the correct tasks remain
	expectedUIDs := map[string]bool{"1": true, "3": true}
	for _, task := range filtered {
		if !expectedUIDs[task.UID] {
			t.Errorf("Unexpected task UID %s in filtered results", task.UID)
		}
	}
}

func TestExcludeStatusFilterCaseInsensitive(t *testing.T) {
	tasks := []backend.Task{
		{UID: "1", Summary: "Todo task", Status: "todo"},
		{UID: "2", Summary: "Done task", Status: "done"},
	}

	filters := &ViewFilters{
		ExcludeStatuses: []string{"DONE"},
	}

	filtered := ApplyFilters(tasks, filters)

	// Should only have TODO task (case-insensitive matching)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 task after filtering, got %d", len(filtered))
	}

	if filtered[0].UID != "1" {
		t.Errorf("Expected task UID 1, got %s", filtered[0].UID)
	}
}

func TestExcludeAndIncludeStatusFilters(t *testing.T) {
	tasks := []backend.Task{
		{UID: "1", Summary: "Todo task", Status: "TODO"},
		{UID: "2", Summary: "Done task", Status: "DONE"},
		{UID: "3", Summary: "Processing task", Status: "PROCESSING"},
	}

	// Exclude takes precedence - if a status is both included and excluded, it should be excluded
	filters := &ViewFilters{
		Status:          []string{"TODO", "DONE"},
		ExcludeStatuses: []string{"DONE"},
	}

	filtered := ApplyFilters(tasks, filters)

	// Should only have TODO task (DONE is excluded even though it's in include list)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 task after filtering, got %d", len(filtered))
	}

	if filtered[0].UID != "1" {
		t.Errorf("Expected task UID 1, got %s", filtered[0].UID)
	}
}
