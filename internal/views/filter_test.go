package views

import (
	"gosynctasks/backend"
	"testing"
	"time"
)

func TestApplyFilters_Status(t *testing.T) {
	tasks := []backend.Task{
		{UID: "1", Summary: "Task 1", Status: "NEEDS-ACTION"},
		{UID: "2", Summary: "Task 2", Status: "COMPLETED"},
		{UID: "3", Summary: "Task 3", Status: "IN-PROCESS"},
		{UID: "4", Summary: "Task 4", Status: "NEEDS-ACTION"},
	}

	filters := &ViewFilters{
		Status: []string{"NEEDS-ACTION"},
	}

	result := ApplyFilters(tasks, filters)

	if len(result) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(result))
	}

	for _, task := range result {
		if task.Status != "NEEDS-ACTION" {
			t.Errorf("Expected status NEEDS-ACTION, got %s", task.Status)
		}
	}
}

func TestApplyFilters_Priority(t *testing.T) {
	tasks := []backend.Task{
		{UID: "1", Summary: "Task 1", Priority: 1},
		{UID: "2", Summary: "Task 2", Priority: 5},
		{UID: "3", Summary: "Task 3", Priority: 9},
		{UID: "4", Summary: "Task 4", Priority: 3},
	}

	tests := []struct {
		name        string
		priorityMin int
		priorityMax int
		expected    int
	}{
		{"Min only", 3, 0, 3},       // >= 3
		{"Max only", 0, 5, 3},       // <= 5
		{"Range", 3, 7, 2},          // 3-7
		{"No filter", 0, 0, 4},      // all
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters := &ViewFilters{
				PriorityMin: tt.priorityMin,
				PriorityMax: tt.priorityMax,
			}

			result := ApplyFilters(tasks, filters)

			if len(result) != tt.expected {
				t.Errorf("Expected %d tasks, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestApplyFilters_Tags(t *testing.T) {
	tasks := []backend.Task{
		{UID: "1", Summary: "Task 1", Categories: []string{"work", "urgent"}},
		{UID: "2", Summary: "Task 2", Categories: []string{"personal"}},
		{UID: "3", Summary: "Task 3", Categories: []string{"work", "project"}},
		{UID: "4", Summary: "Task 4", Categories: []string{"work", "urgent", "project"}},
	}

	tests := []struct {
		name     string
		tags     []string
		expected int
	}{
		{"Single tag", []string{"work"}, 3},
		{"Multiple tags - all required", []string{"work", "urgent"}, 2},
		{"No match", []string{"home"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters := &ViewFilters{
				Tags: tt.tags,
			}

			result := ApplyFilters(tasks, filters)

			if len(result) != tt.expected {
				t.Errorf("Expected %d tasks, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestApplyFilters_DueDates(t *testing.T) {
	// Use dates at midnight for clearer boundaries
	baseDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	yesterday := baseDate.AddDate(0, 0, -1)
	tomorrow := baseDate.AddDate(0, 0, 1)
	dayAfterTomorrow := baseDate.AddDate(0, 0, 2)

	tasks := []backend.Task{
		{UID: "1", Summary: "Task 1", DueDate: &yesterday},
		{UID: "2", Summary: "Task 2", DueDate: &baseDate},
		{UID: "3", Summary: "Task 3", DueDate: &tomorrow},
		{UID: "4", Summary: "Task 4", DueDate: &dayAfterTomorrow},
		{UID: "5", Summary: "Task 5", DueDate: nil},
	}

	t.Run("DueBefore", func(t *testing.T) {
		filters := &ViewFilters{
			DueBefore: &baseDate,
		}

		result := ApplyFilters(tasks, filters)

		// Should get tasks due before baseDate (only yesterday)
		if len(result) != 1 {
			t.Errorf("Expected 1 task, got %d", len(result))
		}
		if result[0].UID != "1" {
			t.Errorf("Expected task 1, got task %s", result[0].UID)
		}
	})

	t.Run("DueAfter", func(t *testing.T) {
		filters := &ViewFilters{
			DueAfter: &baseDate,
		}

		result := ApplyFilters(tasks, filters)

		// Should get tasks due after baseDate (tomorrow and day after)
		if len(result) != 2 {
			t.Errorf("Expected 2 tasks, got %d", len(result))
		}
	})
}

func TestApplyFilters_Nil(t *testing.T) {
	tasks := []backend.Task{
		{UID: "1", Summary: "Task 1"},
		{UID: "2", Summary: "Task 2"},
	}

	result := ApplyFilters(tasks, nil)

	if len(result) != len(tasks) {
		t.Errorf("Expected all tasks when filters are nil, got %d/%d", len(result), len(tasks))
	}
}

func TestApplySort_Summary(t *testing.T) {
	tasks := []backend.Task{
		{UID: "1", Summary: "Zebra"},
		{UID: "2", Summary: "Alpha"},
		{UID: "3", Summary: "Beta"},
	}

	ApplySort(tasks, "summary", "asc")

	expected := []string{"Alpha", "Beta", "Zebra"}
	for i, task := range tasks {
		if task.Summary != expected[i] {
			t.Errorf("Position %d: expected %s, got %s", i, expected[i], task.Summary)
		}
	}

	// Test descending
	ApplySort(tasks, "summary", "desc")

	expectedDesc := []string{"Zebra", "Beta", "Alpha"}
	for i, task := range tasks {
		if task.Summary != expectedDesc[i] {
			t.Errorf("Position %d: expected %s, got %s", i, expectedDesc[i], task.Summary)
		}
	}
}

func TestApplySort_Priority(t *testing.T) {
	tasks := []backend.Task{
		{UID: "1", Summary: "Task 1", Priority: 5},
		{UID: "2", Summary: "Task 2", Priority: 1},
		{UID: "3", Summary: "Task 3", Priority: 0}, // undefined
		{UID: "4", Summary: "Task 4", Priority: 9},
	}

	ApplySort(tasks, "priority", "asc")

	// Should be: 1, 5, 9, 0 (undefined goes last)
	expected := []int{1, 5, 9, 0}
	for i, task := range tasks {
		if task.Priority != expected[i] {
			t.Errorf("Position %d: expected priority %d, got %d", i, expected[i], task.Priority)
		}
	}
}

func TestApplySort_DueDate(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1)
	today := time.Now()
	tomorrow := time.Now().AddDate(0, 0, 1)

	tasks := []backend.Task{
		{UID: "1", Summary: "Task 1", DueDate: &tomorrow},
		{UID: "2", Summary: "Task 2", DueDate: &yesterday},
		{UID: "3", Summary: "Task 3", DueDate: &today},
		{UID: "4", Summary: "Task 4", DueDate: nil},
	}

	ApplySort(tasks, "due_date", "asc")

	// Should be: yesterday, today, tomorrow, nil
	if tasks[0].UID != "2" {
		t.Errorf("Expected task 2 first, got %s", tasks[0].UID)
	}
	if tasks[1].UID != "3" {
		t.Errorf("Expected task 3 second, got %s", tasks[1].UID)
	}
	if tasks[2].UID != "1" {
		t.Errorf("Expected task 1 third, got %s", tasks[2].UID)
	}
	if tasks[3].UID != "4" {
		t.Errorf("Expected task 4 last (nil date), got %s", tasks[3].UID)
	}
}

func TestApplySort_NoSort(t *testing.T) {
	tasks := []backend.Task{
		{UID: "1", Summary: "Task 1"},
		{UID: "2", Summary: "Task 2"},
		{UID: "3", Summary: "Task 3"},
	}

	original := make([]backend.Task, len(tasks))
	copy(original, tasks)

	ApplySort(tasks, "", "asc")

	// Should remain unchanged
	for i, task := range tasks {
		if task.UID != original[i].UID {
			t.Errorf("Order changed when sortBy is empty")
		}
	}
}
