package views

import (
	"gosynctasks/backend"
	"strings"
	"testing"
	"time"
)

func TestViewRenderer_RenderTask(t *testing.T) {
	trueVal := true
	// Create a minimal view
	view := &View{
		Name: "test",
		Fields: []FieldConfig{
			{Name: "status", Format: "symbol", Show: &trueVal},
			{Name: "summary", Format: "full", Show: &trueVal},
		},
		FieldOrder: []string{"status", "summary"},
		Display: DisplayOptions{
			ShowHeader:  true,
			ShowBorder:  true,
			CompactMode: false,
			DateFormat:  "2006-01-02",
		},
	}

	renderer := NewViewRenderer(view, nil, "2006-01-02")

	task := backend.Task{
		UID:     "test-1",
		Summary: "Test task",
		Status:  "NEEDS-ACTION",
	}

	result := renderer.RenderTask(task)

	// Check that result contains expected elements
	if !strings.Contains(result, "Test task") {
		t.Errorf("Expected task summary in output, got: %s", result)
	}

	if !strings.Contains(result, "○") {
		t.Errorf("Expected status symbol in output, got: %s", result)
	}
}

func TestViewRenderer_RenderTaskWithDates(t *testing.T) {
	trueVal := true
	// Create view with date fields
	view := &View{
		Name: "dates",
		Fields: []FieldConfig{
			{Name: "status", Format: "symbol", Show: &trueVal},
			{Name: "summary", Format: "full", Show: &trueVal},
			{Name: "start_date", Format: "full", Show: &trueVal, Color: true},
			{Name: "due_date", Format: "full", Show: &trueVal, Color: true},
		},
		FieldOrder: []string{"status", "summary", "start_date", "due_date"},
		Display: DisplayOptions{
			CompactMode: false,
			DateFormat:  "2006-01-02",
		},
	}

	renderer := NewViewRenderer(view, nil, "2006-01-02")

	now := time.Now()
	startDate := now.AddDate(0, 0, 1)
	dueDate := now.AddDate(0, 0, 7)

	task := backend.Task{
		UID:       "test-2",
		Summary:   "Task with dates",
		Status:    "IN-PROCESS",
		StartDate: &startDate,
		DueDate:   &dueDate,
	}

	result := renderer.RenderTask(task)

	// Check for dates in output
	if !strings.Contains(result, startDate.Format("2006-01-02")) {
		t.Errorf("Expected start date in output, got: %s", result)
	}

	if !strings.Contains(result, dueDate.Format("2006-01-02")) {
		t.Errorf("Expected due date in output, got: %s", result)
	}
}

func TestViewRenderer_CompactMode(t *testing.T) {
	trueVal := true
	// Create view with compact mode
	view := &View{
		Name: "compact",
		Fields: []FieldConfig{
			{Name: "status", Format: "short", Show: &trueVal},
			{Name: "summary", Format: "truncate", Width: 30, Show: &trueVal},
		},
		FieldOrder: []string{"status", "summary"},
		Display: DisplayOptions{
			CompactMode: true,
			DateFormat:  "2006-01-02",
		},
	}

	renderer := NewViewRenderer(view, nil, "2006-01-02")

	task := backend.Task{
		UID:     "test-3",
		Summary: "Short task",
		Status:  "COMPLETED",
	}

	result := renderer.RenderTask(task)

	// Compact mode should render on a single line
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 1 {
		t.Errorf("Expected 1 line in compact mode, got %d lines: %s", len(lines), result)
	}
}

func TestViewRenderer_RenderTasks(t *testing.T) {
	trueVal := true
	view := &View{
		Name: "test",
		Fields: []FieldConfig{
			{Name: "status", Format: "symbol", Show: &trueVal},
			{Name: "summary", Format: "full", Show: &trueVal},
		},
		Display: DisplayOptions{
			CompactMode: false,
			DateFormat:  "2006-01-02",
		},
	}

	renderer := NewViewRenderer(view, nil, "2006-01-02")

	tasks := []backend.Task{
		{UID: "1", Summary: "Task 1", Status: "NEEDS-ACTION"},
		{UID: "2", Summary: "Task 2", Status: "COMPLETED"},
		{UID: "3", Summary: "Task 3", Status: "IN-PROCESS"},
	}

	result := renderer.RenderTasks(tasks)

	// Check all tasks are in output
	for _, task := range tasks {
		if !strings.Contains(result, task.Summary) {
			t.Errorf("Expected task %s in output, got: %s", task.Summary, result)
		}
	}
}

func TestViewRenderer_FieldOrder(t *testing.T) {
	trueVal := true
	// Create view with custom field order
	view := &View{
		Name: "ordered",
		Fields: []FieldConfig{
			{Name: "status", Format: "symbol", Show: &trueVal},
			{Name: "summary", Format: "full", Show: &trueVal},
			{Name: "priority", Format: "number", Show: &trueVal},
		},
		FieldOrder: []string{"priority", "status", "summary"}, // Different order
		Display: DisplayOptions{
			CompactMode: true,
			DateFormat:  "2006-01-02",
		},
	}

	renderer := NewViewRenderer(view, nil, "2006-01-02")

	task := backend.Task{
		UID:      "test-4",
		Summary:  "Priority task",
		Status:   "NEEDS-ACTION",
		Priority: 1,
	}

	result := renderer.RenderTask(task)

	// In compact mode, fields should appear in the specified order
	// Priority should come before status symbol
	if !strings.Contains(result, "1") {
		t.Errorf("Expected priority in output, got: %s", result)
	}
}

func TestViewRenderer_HiddenFields(t *testing.T) {
	trueVal := true
	falseVal := false
	view := &View{
		Name: "hidden",
		Fields: []FieldConfig{
			{Name: "status", Format: "symbol", Show: &trueVal},
			{Name: "summary", Format: "full", Show: &trueVal},
			{Name: "description", Format: "truncate", Show: &falseVal}, // Hidden
		},
		Display: DisplayOptions{
			CompactMode: false,
			DateFormat:  "2006-01-02",
		},
	}

	renderer := NewViewRenderer(view, nil, "2006-01-02")

	task := backend.Task{
		UID:         "test-5",
		Summary:     "Task with description",
		Status:      "NEEDS-ACTION",
		Description: "This should not appear",
	}

	result := renderer.RenderTask(task)

	// Description should not appear since Show=false
	if strings.Contains(result, "This should not appear") {
		t.Errorf("Hidden field appeared in output: %s", result)
	}
}
