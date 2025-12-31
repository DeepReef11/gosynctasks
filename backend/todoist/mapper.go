package todoist

import (
	"gosynctasks/backend"
	"fmt"
	"strings"
	"time"
)

// toTask converts a Todoist task to gosynctasks Task
func toTask(todoistTask *TodoistTask) backend.Task {
	task := backend.Task{
		UID:         todoistTask.ID,
		Summary:     todoistTask.Content,
		Description: todoistTask.Description,
		Categories:  todoistTask.Labels,
		ParentUID:   todoistTask.ParentID,
	}

	// Map status (Todoist only has completed/not completed)
	if todoistTask.IsCompleted {
		task.Status = "DONE"
	} else {
		task.Status = "TODO"
	}

	// Map priority (Todoist: 1=normal, 4=urgent; gosynctasks: 1=highest, 9=lowest)
	// Conversion: priority 4 → 1, priority 3 → 3, priority 2 → 5, priority 1 → 7
	switch todoistTask.Priority {
	case 4: // Urgent
		task.Priority = 1
	case 3: // High
		task.Priority = 3
	case 2: // Medium
		task.Priority = 5
	case 1: // Normal
		task.Priority = 7
	default:
		task.Priority = 0 // Undefined
	}

	// Parse due date
	if todoistTask.Due != nil {
		if todoistTask.Due.Datetime != "" {
			// Has specific datetime
			if dueTime, err := time.Parse(time.RFC3339, todoistTask.Due.Datetime); err == nil {
				task.DueDate = &dueTime
			}
		} else if todoistTask.Due.Date != "" {
			// Has only date (no specific time)
			if dueTime, err := time.Parse("2006-01-02", todoistTask.Due.Date); err == nil {
				task.DueDate = &dueTime
			}
		}
	}

	// Parse created timestamp
	if todoistTask.CreatedAt != "" {
		if createdTime, err := time.Parse(time.RFC3339, todoistTask.CreatedAt); err == nil {
			task.Created = createdTime
		}
	}

	// Todoist doesn't provide modified timestamp directly
	// We'll use created time as fallback
	task.Modified = task.Created

	return task
}

// toTaskList converts a Todoist project to gosynctasks TaskList
func toTaskList(project *Project) backend.TaskList {
	return backend.TaskList{
		ID:          project.ID,
		Name:        project.Name,
		Description: fmt.Sprintf("%d comments", project.CommentCount),
		Color:       project.Color,
	}
}

// toCreateTaskRequest converts gosynctasks Task to Todoist create request
func toCreateTaskRequest(task backend.Task, projectID string) CreateTaskRequest {
	req := CreateTaskRequest{
		Content:     task.Summary,
		Description: task.Description,
		ProjectID:   projectID,
		ParentID:    task.ParentUID,
		Labels:      task.Categories,
	}

	// Map priority (gosynctasks: 1=highest; Todoist: 4=urgent)
	switch {
	case task.Priority >= 1 && task.Priority <= 2:
		req.Priority = 4 // Urgent
	case task.Priority >= 3 && task.Priority <= 4:
		req.Priority = 3 // High
	case task.Priority >= 5 && task.Priority <= 6:
		req.Priority = 2 // Medium
	case task.Priority >= 7 && task.Priority <= 9:
		req.Priority = 1 // Normal
	default:
		req.Priority = 1 // Default to normal
	}

	// Set due date
	if task.DueDate != nil && !task.DueDate.IsZero() {
		// Check if it has time component
		if task.DueDate.Hour() == 0 && task.DueDate.Minute() == 0 && task.DueDate.Second() == 0 {
			// Date only
			req.DueDate = task.DueDate.Format("2006-01-02")
		} else {
			// Date and time
			req.DueDatetime = task.DueDate.Format(time.RFC3339)
		}
	}

	return req
}

// toUpdateTaskRequest converts gosynctasks Task to Todoist update request
func toUpdateTaskRequest(task backend.Task) UpdateTaskRequest {
	req := UpdateTaskRequest{
		Content:     task.Summary,
		Description: task.Description,
		Labels:      task.Categories,
	}

	// Map priority
	switch {
	case task.Priority >= 1 && task.Priority <= 2:
		req.Priority = 4 // Urgent
	case task.Priority >= 3 && task.Priority <= 4:
		req.Priority = 3 // High
	case task.Priority >= 5 && task.Priority <= 6:
		req.Priority = 2 // Medium
	case task.Priority >= 7 && task.Priority <= 9:
		req.Priority = 1 // Normal
	default:
		req.Priority = 1 // Default to normal
	}

	// Set due date
	if task.DueDate != nil && !task.DueDate.IsZero() {
		if task.DueDate.Hour() == 0 && task.DueDate.Minute() == 0 && task.DueDate.Second() == 0 {
			req.DueDate = task.DueDate.Format("2006-01-02")
		} else {
			req.DueDatetime = task.DueDate.Format(time.RFC3339)
		}
	}

	return req
}

// parseStatusFlag converts CLI status input to Todoist-compatible status
func parseStatusFlag(statusFlag string) (string, error) {
	upper := strings.ToUpper(statusFlag)

	// Handle abbreviations
	switch upper {
	case "T":
		return "TODO", nil
	case "D":
		return "DONE", nil
	case "P":
		// Todoist doesn't have "PROCESSING" - we'll use TODO with a label
		return "PROCESSING", nil
	case "C":
		// Todoist doesn't have "CANCELLED" - we'll use a label
		return "CANCELLED", nil
	}

	// Handle full names
	switch upper {
	case "TODO", "DONE", "PROCESSING", "CANCELLED":
		return upper, nil
	}

	return "", fmt.Errorf("invalid status flag: %s (use TODO/T, DONE/D, PROCESSING/P, CANCELLED/C)", statusFlag)
}

// statusToDisplayName converts backend status to display name
func statusToDisplayName(backendStatus string) string {
	// Todoist backend uses app-style status names directly
	return backendStatus
}
