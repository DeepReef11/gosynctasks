package todoist

import (
	"gosynctasks/backend"
	"testing"
	"time"
)

func TestToTask(t *testing.T) {
	tests := []struct {
		name         string
		todoistTask  TodoistTask
		expectedTask backend.Task
	}{
		{
			name: "completed task with all fields",
			todoistTask: TodoistTask{
				ID:          "task123",
				Content:     "Test Task",
				Description: "Test Description",
				IsCompleted: true,
				Labels:      []string{"work", "urgent"},
				Priority:    4, // Todoist urgent
				ParentID:    "parent123",
				CreatedAt:   "2026-01-01T10:00:00Z",
				Due: &Due{
					Date:     "2026-01-15",
					Datetime: "2026-01-15T12:00:00Z",
				},
			},
			expectedTask: backend.Task{
				UID:         "task123",
				Summary:     "Test Task",
				Description: "Test Description",
				Status:      "DONE",
				Categories:  []string{"work", "urgent"},
				Priority:    1, // Maps to highest priority
				ParentUID:   "parent123",
			},
		},
		{
			name: "incomplete task with normal priority",
			todoistTask: TodoistTask{
				ID:          "task456",
				Content:     "Normal Task",
				IsCompleted: false,
				Priority:    1, // Todoist normal
				CreatedAt:   "2026-01-01T10:00:00Z",
			},
			expectedTask: backend.Task{
				UID:      "task456",
				Summary:  "Normal Task",
				Status:   "TODO",
				Priority: 7, // Maps to lower priority
			},
		},
		{
			name: "task with date only (no time)",
			todoistTask: TodoistTask{
				ID:          "task789",
				Content:     "Date Only Task",
				IsCompleted: false,
				Priority:    2, // Medium
				CreatedAt:   "2026-01-01T10:00:00Z",
				Due: &Due{
					Date: "2026-01-20",
				},
			},
			expectedTask: backend.Task{
				UID:      "task789",
				Summary:  "Date Only Task",
				Status:   "TODO",
				Priority: 5, // Maps to medium
			},
		},
		{
			name: "task with no due date",
			todoistTask: TodoistTask{
				ID:          "task-no-due",
				Content:     "No Due Date",
				IsCompleted: false,
				Priority:    3, // High
				CreatedAt:   "2026-01-01T10:00:00Z",
			},
			expectedTask: backend.Task{
				UID:      "task-no-due",
				Summary:  "No Due Date",
				Status:   "TODO",
				Priority: 3, // Maps to high
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toTask(&tt.todoistTask)

			if result.UID != tt.expectedTask.UID {
				t.Errorf("UID = %q, want %q", result.UID, tt.expectedTask.UID)
			}
			if result.Summary != tt.expectedTask.Summary {
				t.Errorf("Summary = %q, want %q", result.Summary, tt.expectedTask.Summary)
			}
			if result.Description != tt.expectedTask.Description {
				t.Errorf("Description = %q, want %q", result.Description, tt.expectedTask.Description)
			}
			if result.Status != tt.expectedTask.Status {
				t.Errorf("Status = %q, want %q", result.Status, tt.expectedTask.Status)
			}
			if result.Priority != tt.expectedTask.Priority {
				t.Errorf("Priority = %d, want %d", result.Priority, tt.expectedTask.Priority)
			}
			if result.ParentUID != tt.expectedTask.ParentUID {
				t.Errorf("ParentUID = %q, want %q", result.ParentUID, tt.expectedTask.ParentUID)
			}

			// Check categories
			if len(result.Categories) != len(tt.expectedTask.Categories) {
				t.Errorf("Categories count = %d, want %d", len(result.Categories), len(tt.expectedTask.Categories))
			}

			// Check due date was parsed if present
			if tt.todoistTask.Due != nil {
				if result.DueDate == nil {
					t.Error("DueDate is nil, expected non-nil")
				}
			}

			// Check created time was parsed
			if result.Created.IsZero() {
				t.Error("Created time is zero, expected non-zero")
			}
		})
	}
}

func TestPriorityMapping_TodoistToGosynctasks(t *testing.T) {
	tests := []struct {
		todoistPriority int
		expectedPriority int
	}{
		{4, 1}, // Urgent → Highest
		{3, 3}, // High → High
		{2, 5}, // Medium → Medium
		{1, 7}, // Normal → Lower
		{0, 0}, // Undefined → Undefined
	}

	for _, tt := range tests {
		t.Run("priority", func(t *testing.T) {
			todoistTask := TodoistTask{
				ID:          "test",
				Content:     "Test",
				Priority:    tt.todoistPriority,
				IsCompleted: false,
				CreatedAt:   time.Now().Format(time.RFC3339),
			}

			result := toTask(&todoistTask)

			if result.Priority != tt.expectedPriority {
				t.Errorf("Priority %d mapped to %d, want %d",
					tt.todoistPriority, result.Priority, tt.expectedPriority)
			}
		})
	}
}

func TestToTaskList(t *testing.T) {
	project := Project{
		ID:             "proj123",
		Name:           "My Project",
		CommentCount:   42,
		Color:          "blue",
		IsFavorite:     true,
		IsInboxProject: false,
	}

	result := toTaskList(&project)

	if result.ID != "proj123" {
		t.Errorf("ID = %q, want %q", result.ID, "proj123")
	}
	if result.Name != "My Project" {
		t.Errorf("Name = %q, want %q", result.Name, "My Project")
	}
	if result.Color != "blue" {
		t.Errorf("Color = %q, want %q", result.Color, "blue")
	}
	if result.Description != "42 comments" {
		t.Errorf("Description = %q, want %q", result.Description, "42 comments")
	}
}

func TestToCreateTaskRequest(t *testing.T) {
	dueDateTime := time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC)
	dueDate := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		task        backend.Task
		projectID   string
		checkFields func(t *testing.T, req CreateTaskRequest)
	}{
		{
			name: "task with datetime",
			task: backend.Task{
				Summary:     "Task with datetime",
				Description: "Description",
				Priority:    1, // High
				Categories:  []string{"urgent", "work"},
				DueDate:     &dueDateTime,
				ParentUID:   "parent123",
			},
			projectID: "proj123",
			checkFields: func(t *testing.T, req CreateTaskRequest) {
				if req.Content != "Task with datetime" {
					t.Errorf("Content = %q, want %q", req.Content, "Task with datetime")
				}
				if req.ProjectID != "proj123" {
					t.Errorf("ProjectID = %q, want %q", req.ProjectID, "proj123")
				}
				if req.Priority != 4 { // Priority 1 maps to Todoist 4 (urgent)
					t.Errorf("Priority = %d, want 4", req.Priority)
				}
				if req.ParentID != "parent123" {
					t.Errorf("ParentID = %q, want %q", req.ParentID, "parent123")
				}
				if req.DueDatetime == "" {
					t.Error("DueDatetime is empty, expected non-empty")
				}
				if req.DueDate != "" {
					t.Errorf("DueDate should be empty when datetime is set, got %q", req.DueDate)
				}
			},
		},
		{
			name: "task with date only",
			task: backend.Task{
				Summary:  "Task with date",
				Priority: 5, // Medium
				DueDate:  &dueDate,
			},
			projectID: "proj456",
			checkFields: func(t *testing.T, req CreateTaskRequest) {
				if req.Content != "Task with date" {
					t.Errorf("Content = %q, want %q", req.Content, "Task with date")
				}
				if req.Priority != 2 { // Priority 5 maps to Todoist 2 (medium)
					t.Errorf("Priority = %d, want 2", req.Priority)
				}
				if req.DueDate != "2026-01-20" {
					t.Errorf("DueDate = %q, want %q", req.DueDate, "2026-01-20")
				}
				if req.DueDatetime != "" {
					t.Errorf("DueDatetime should be empty, got %q", req.DueDatetime)
				}
			},
		},
		{
			name: "task with no due date",
			task: backend.Task{
				Summary:  "No due date",
				Priority: 7, // Low
			},
			projectID: "proj789",
			checkFields: func(t *testing.T, req CreateTaskRequest) {
				if req.Priority != 1 { // Priority 7 maps to Todoist 1 (normal)
					t.Errorf("Priority = %d, want 1", req.Priority)
				}
				if req.DueDate != "" {
					t.Errorf("DueDate should be empty, got %q", req.DueDate)
				}
				if req.DueDatetime != "" {
					t.Errorf("DueDatetime should be empty, got %q", req.DueDatetime)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toCreateTaskRequest(tt.task, tt.projectID)
			tt.checkFields(t, result)
		})
	}
}

func TestPriorityMapping_GosynctasksToTodoist(t *testing.T) {
	tests := []struct {
		gosynctasksPriority int
		expectedTodoistPriority int
	}{
		{1, 4}, // Highest → Urgent
		{2, 4}, // High → Urgent
		{3, 3}, // High → High
		{4, 3}, // High → High
		{5, 2}, // Medium → Medium
		{6, 2}, // Medium → Medium
		{7, 1}, // Low → Normal
		{8, 1}, // Low → Normal
		{9, 1}, // Lowest → Normal
		{0, 1}, // Undefined → Normal (default)
	}

	for _, tt := range tests {
		t.Run("priority", func(t *testing.T) {
			task := backend.Task{
				Summary:  "Test",
				Priority: tt.gosynctasksPriority,
			}

			result := toCreateTaskRequest(task, "proj")

			if result.Priority != tt.expectedTodoistPriority {
				t.Errorf("Priority %d mapped to %d, want %d",
					tt.gosynctasksPriority, result.Priority, tt.expectedTodoistPriority)
			}
		})
	}
}

func TestToUpdateTaskRequest(t *testing.T) {
	dueDate := time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC)

	task := backend.Task{
		UID:         "task123",
		Summary:     "Updated Task",
		Description: "Updated Description",
		Priority:    3,
		Categories:  []string{"updated"},
		DueDate:     &dueDate,
	}

	result := toUpdateTaskRequest(task)

	if result.Content != "Updated Task" {
		t.Errorf("Content = %q, want %q", result.Content, "Updated Task")
	}
	if result.Description != "Updated Description" {
		t.Errorf("Description = %q, want %q", result.Description, "Updated Description")
	}
	if result.Priority != 3 { // Priority 3 maps to Todoist 3
		t.Errorf("Priority = %d, want 3", result.Priority)
	}
	if len(result.Labels) != 1 || result.Labels[0] != "updated" {
		t.Errorf("Labels = %v, want [updated]", result.Labels)
	}
	if result.DueDate != "2026-01-25" {
		t.Errorf("DueDate = %q, want %q", result.DueDate, "2026-01-25")
	}
}

func TestParseStatusFlag(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		// Abbreviations
		{"T", "TODO", false},
		{"D", "DONE", false},
		{"P", "PROCESSING", false},
		{"C", "CANCELLED", false},

		// Full names
		{"TODO", "TODO", false},
		{"DONE", "DONE", false},
		{"PROCESSING", "PROCESSING", false},
		{"CANCELLED", "CANCELLED", false},

		// Lowercase
		{"todo", "TODO", false},
		{"done", "DONE", false},

		// Invalid
		{"INVALID", "", true},
		{"X", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseStatusFlag(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("parseStatusFlag(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStatusToDisplayName(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"TODO", "TODO"},
		{"DONE", "DONE"},
		{"PROCESSING", "PROCESSING"},
		{"CANCELLED", "CANCELLED"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := statusToDisplayName(tt.status)
			if result != tt.expected {
				t.Errorf("statusToDisplayName(%q) = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}
