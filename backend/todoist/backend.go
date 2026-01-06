package todoist

import (
	"fmt"
	"strings"

	"gosynctasks/backend"
	"gosynctasks/internal/credentials"
	"gosynctasks/internal/utils"
)

func init() {
	// Register Todoist backend for config type "todoist"
	backend.RegisterType("todoist", newTodoistBackendWrapper)
}

// newTodoistBackendWrapper wraps NewTodoistBackend to match BackendConfigConstructor signature
func newTodoistBackendWrapper(config backend.BackendConfig) (backend.TaskManager, error) {
	return NewTodoistBackend(config)
}

// TodoistBackend implements backend.TaskManager for Todoist
type TodoistBackend struct {
	config         backend.BackendConfig
	apiClient      *APIClient
	apiToken       string
	BackendName    string // Backend name for credential resolution
	ConfigUsername string // Username hint from config (typically "token" for API keys)
}

// NewTodoistBackend creates a new Todoist backend instance
func NewTodoistBackend(config backend.BackendConfig) (*TodoistBackend, error) {
	tb := &TodoistBackend{
		config:         config,
		BackendName:    config.Name,
		ConfigUsername: config.Username,
	}

	// Resolve API token from credentials (keyring > env > config)
	apiToken, err := tb.getAPIToken()
	if err != nil {
		return nil, err
	}

	tb.apiToken = apiToken
	tb.apiClient = NewAPIClient(apiToken)

	// Validate token by attempting to fetch projects
	if _, err := tb.apiClient.GetProjects(); err != nil {
		return nil, fmt.Errorf("failed to validate Todoist API token: %w", err)
	}

	return tb, nil
}

// getAPIToken retrieves the API token from credentials with priority:
// 1. Keyring (if username is provided, typically "token")
// 2. Environment variable (GOSYNCTASKS_TODOIST_PASSWORD or GOSYNCTASKS_<BACKEND_NAME>_PASSWORD)
// 3. Config file (api_token field)
func (tb *TodoistBackend) getAPIToken() (string, error) {
	// Try credential resolver first if backend name is available
	if tb.BackendName != "" {
		resolver := credentials.NewResolver()

		// For Todoist, we use username as a hint (typically "token")
		// The API token is stored as the "password" in the keyring
		username := tb.ConfigUsername
		if username == "" {
			username = "token" // Default username hint for API tokens
		}

		creds, err := resolver.Resolve(tb.BackendName, username, "", nil)
		if err == nil && creds.Password != "" {
			// API token found in keyring or environment
			return creds.Password, nil
		}
		// If error is not "not found", log but continue to fallback
	}

	// Fallback to config.APIToken
	if tb.config.APIToken != "" {
		return tb.config.APIToken, nil
	}

	// No token found anywhere
	return "", fmt.Errorf("todoist API token not found (tried: keyring, environment variables, config)\n"+
		"Set it with: gosynctasks credentials set %s token --prompt\n"+
		"Or add 'api_token' to your config file", tb.BackendName)
}

// GetTaskLists retrieves all Todoist projects as task lists
func (tb *TodoistBackend) GetTaskLists() ([]backend.TaskList, error) {
	projects, err := tb.apiClient.GetProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	lists := make([]backend.TaskList, len(projects))
	for i, project := range projects {
		lists[i] = toTaskList(&project)
	}

	return lists, nil
}

// GetTasks retrieves tasks from a specific project
func (tb *TodoistBackend) GetTasks(listID string, filter *backend.TaskFilter) ([]backend.Task, error) {
	todoistTasks, err := tb.apiClient.GetTasks(listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	var tasks []backend.Task
	for i := range todoistTasks {
		task := toTask(&todoistTasks[i])

		// Apply filter if provided
		if filter != nil && !tb.matchesFilter(task, filter) {
			continue
		}

		tasks = append(tasks, task)
	}

	// Sort tasks
	tb.SortTasks(tasks)

	return tasks, nil
}

// matchesFilter checks if a task matches the given filter
func (tb *TodoistBackend) matchesFilter(task backend.Task, filter *backend.TaskFilter) bool {
	// Check status filter
	if filter.Statuses != nil && len(*filter.Statuses) > 0 {
		matchesStatus := false
		for _, status := range *filter.Statuses {
			if task.Status == status {
				matchesStatus = true
				break
			}
		}
		if !matchesStatus {
			return false
		}
	}

	// Check due date filters
	if filter.DueAfter != nil && task.DueDate != nil && !task.DueDate.IsZero() {
		if task.DueDate.Before(*filter.DueAfter) {
			return false
		}
	}

	if filter.DueBefore != nil && task.DueDate != nil && !task.DueDate.IsZero() {
		if task.DueDate.After(*filter.DueBefore) {
			return false
		}
	}

	// Check created after filter
	if filter.CreatedAfter != nil && !task.Created.IsZero() {
		if task.Created.Before(*filter.CreatedAfter) {
			return false
		}
	}

	return true
}

// FindTasksBySummary searches for tasks by content
func (tb *TodoistBackend) FindTasksBySummary(listID string, summary string) ([]backend.Task, error) {
	tasks, err := tb.GetTasks(listID, nil)
	if err != nil {
		return nil, err
	}

	summary = strings.ToLower(summary)
	var matches []backend.Task

	for _, task := range tasks {
		if strings.Contains(strings.ToLower(task.Summary), summary) {
			matches = append(matches, task)
		}
	}

	return matches, nil
}

// AddTask creates a new task in Todoist
func (tb *TodoistBackend) AddTask(listID string, task backend.Task) (string, error) {
	req := toCreateTaskRequest(task, listID)

	createdTask, err := tb.apiClient.CreateTask(req)
	if err != nil {
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	// Return the Todoist-assigned task ID
	return createdTask.ID, nil
}

// UpdateTask modifies an existing task
func (tb *TodoistBackend) UpdateTask(listID string, task backend.Task) error {
	// Update other task properties FIRST (before closing/reopening)
	// Todoist API doesn't allow updating closed tasks
	req := toUpdateTaskRequest(task)

	// Debug: Log the request being sent
	contentStr := ""
	if req.Content != nil {
		contentStr = *req.Content
	}
	descStr := ""
	if req.Description != nil {
		descStr = *req.Description
	}
	priorityInt := 0
	if req.Priority != nil {
		priorityInt = *req.Priority
	}
	dueDateStr := ""
	if req.DueDate != nil {
		dueDateStr = *req.DueDate
	}
	dueDatetimeStr := ""
	if req.DueDatetime != nil {
		dueDatetimeStr = *req.DueDatetime
	}
	utils.Debugf("Todoist UpdateTask: ID=%s, Content=%q, Description=%q, Labels=%v, Priority=%d, DueDate=%q, DueDatetime=%q",
		task.UID, contentStr, descStr, req.Labels, priorityInt, dueDateStr, dueDatetimeStr)

	if err := tb.apiClient.UpdateTask(task.UID, req); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Handle status changes AFTER updating properties
	if task.Status == "COMPLETED" {
		// Close the task after updating
		utils.Debugf("[TODOIST] Closing task %s (content: %s)", task.UID, task.Summary)
		if err := tb.apiClient.CloseTask(task.UID); err != nil {
			utils.Debugf("[TODOIST] ERROR: Failed to close task %s: %v", task.UID, err)
			return fmt.Errorf("failed to close task: %w", err)
		}
		utils.Debugf("[TODOIST] ✅ Task %s closed successfully", task.UID)
	} else if task.Status == "TODO" {
		// Reopen if it was completed
		utils.Debugf("[TODOIST] Reopening task %s", task.UID)
		if err := tb.apiClient.ReopenTask(task.UID); err != nil {
			// It might not be closed, so we'll continue
			utils.Debugf("[TODOIST] Failed to reopen task (might not be closed): %v", err)
		}
	}

	return nil
}

// DeleteTask removes a task from Todoist
func (tb *TodoistBackend) DeleteTask(listID string, taskUID string) error {
	if err := tb.apiClient.DeleteTask(taskUID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return backend.NewBackendError("DeleteTask", 404, fmt.Sprintf("task %q not found", taskUID))
		}
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

// CreateTaskList creates a new Todoist project
func (tb *TodoistBackend) CreateTaskList(name, description, color string) (string, error) {
	req := CreateProjectRequest{
		Name:  name,
		Color: color,
	}

	project, err := tb.apiClient.CreateProject(req)
	if err != nil {
		return "", fmt.Errorf("failed to create project: %w", err)
	}

	return project.ID, nil
}

// DeleteTaskList deletes a Todoist project
func (tb *TodoistBackend) DeleteTaskList(listID string) error {
	if err := tb.apiClient.DeleteProject(listID); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}

// RenameTaskList renames a Todoist project
func (tb *TodoistBackend) RenameTaskList(listID, newName string) error {
	req := UpdateProjectRequest{
		Name: newName,
	}

	if err := tb.apiClient.UpdateProject(listID, req); err != nil {
		return fmt.Errorf("failed to rename project: %w", err)
	}

	return nil
}

// GetDeletedTaskLists retrieves deleted projects (not supported by Todoist)
func (tb *TodoistBackend) GetDeletedTaskLists() ([]backend.TaskList, error) {
	// Todoist doesn't have a trash/archive API for projects
	return []backend.TaskList{}, nil
}

// RestoreTaskList restores a deleted project (not supported by Todoist)
func (tb *TodoistBackend) RestoreTaskList(listID string) error {
	return fmt.Errorf("TodoistBackend.RestoreTaskList not supported (Todoist has no trash feature)")
}

// PermanentlyDeleteTaskList permanently deletes a project (not supported separately)
func (tb *TodoistBackend) PermanentlyDeleteTaskList(listID string) error {
	// In Todoist, delete is already permanent
	return tb.DeleteTaskList(listID)
}

// ParseStatusFlag converts user input to Todoist status
func (tb *TodoistBackend) ParseStatusFlag(statusFlag string) (string, error) {
	return parseStatusFlag(statusFlag)
}

// StatusToDisplayName converts Todoist status to display name
func (tb *TodoistBackend) StatusToDisplayName(backendStatus string) string {
	return statusToDisplayName(backendStatus)
}

// SortTasks sorts tasks by priority and creation date
func (tb *TodoistBackend) SortTasks(tasks []backend.Task) {
	// Simple bubble sort (sufficient for typical task lists)
	for i := 0; i < len(tasks); i++ {
		for j := i + 1; j < len(tasks); j++ {
			// Priority 0 goes last
			iPrio := tasks[i].Priority
			jPrio := tasks[j].Priority
			if iPrio == 0 {
				iPrio = 100
			}
			if jPrio == 0 {
				jPrio = 100
			}

			// Lower priority number = higher priority
			if iPrio > jPrio {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			} else if iPrio == jPrio {
				// Same priority, sort by creation date (older first)
				if tasks[i].Created.After(tasks[j].Created) {
					tasks[i], tasks[j] = tasks[j], tasks[i]
				}
			}
		}
	}
}

// GetPriorityColor returns ANSI color code for priority
func (tb *TodoistBackend) GetPriorityColor(priority int) string {
	switch {
	case priority >= 1 && priority <= 2: // Urgent
		return "\033[31m" // Red
	case priority >= 3 && priority <= 4: // High
		return "\033[33m" // Yellow
	case priority >= 5 && priority <= 6: // Medium
		return "\033[36m" // Cyan
	case priority >= 7 && priority <= 9: // Low
		return "\033[34m" // Blue
	default:
		return "" // No color
	}
}

// GetBackendDisplayName returns formatted display name
func (tb *TodoistBackend) GetBackendDisplayName() string {
	return "[todoist]"
}

// GetBackendType returns the backend type identifier
func (tb *TodoistBackend) GetBackendType() string {
	return "todoist"
}

// GetBackendContext returns contextual details
func (tb *TodoistBackend) GetBackendContext() string {
	// We could fetch user info from API, but for now just return the type
	return "todoist"
}
