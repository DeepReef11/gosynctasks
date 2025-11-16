package backend

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type UnsupportedSchemeError struct {
	Scheme string
}

func (e *UnsupportedSchemeError) Error() string {
	return fmt.Sprintf("unsupported scheme: %q", e.Scheme)
}

// Base config struct (deprecated - use BackendConfig for new configurations)
type ConnectorConfig struct {
	URL                *url.URL `json:"url"`
	InsecureSkipVerify bool     `json:"insecure_skip_verify,omitempty"` // WARNING: Only use for self-signed certificates in dev
	SuppressSSLWarning bool     `json:"suppress_ssl_warning,omitempty"` // Suppress SSL warning when InsecureSkipVerify is true
	// Type     string `json:"type" validate:"required,oneof=nextcloud local"`
	//  Timeout  int    `json:"timeout,omitempty"`
}

// BackendConfig represents configuration for a single backend in the multi-backend system.
// Each backend has a type (nextcloud, git, file, sqlite) and type-specific configuration.
type BackendConfig struct {
	Type               string   `json:"type" validate:"required,oneof=nextcloud git file sqlite"`
	Enabled            bool     `json:"enabled"`
	URL                string   `json:"url,omitempty"`                  // Used by: nextcloud, file
	InsecureSkipVerify bool     `json:"insecure_skip_verify,omitempty"` // Used by: nextcloud
	SuppressSSLWarning bool     `json:"suppress_ssl_warning,omitempty"` // Used by: nextcloud
	File               string   `json:"file,omitempty"`                 // Used by: git (default: "TODO.md")
	AutoDetect         bool     `json:"auto_detect,omitempty"`          // Used by: git
	FallbackFiles      []string `json:"fallback_files,omitempty"`       // Used by: git
	AutoCommit         bool     `json:"auto_commit,omitempty"`          // Used by: git
	DBPath             string   `json:"db_path,omitempty"`              // Used by: sqlite
}

func (c *ConnectorConfig) UnmarshalJSON(data []byte) error {
	type ConnConfig ConnectorConfig

	tmp := struct {
		*ConnConfig
		URL                string `json:"url"`
		InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
		SuppressSSLWarning bool   `json:"suppress_ssl_warning,omitempty"`
	}{
		ConnConfig: (*ConnConfig)(c),
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	u, err := url.Parse(tmp.URL)
	if err != nil {
		return err
	}

	tmp.ConnConfig.URL = u
	tmp.ConnConfig.InsecureSkipVerify = tmp.InsecureSkipVerify
	tmp.ConnConfig.SuppressSSLWarning = tmp.SuppressSSLWarning

	return nil
}

func (c *ConnectorConfig) TaskManager() (TaskManager, error) {
	switch c.URL.Scheme {
	case "nextcloud":
		return NewNextcloudBackend(*c)
	case "file":
		return NewFileBackend(*c)
	default:
		return nil, &UnsupportedSchemeError{
			Scheme: c.URL.Scheme,
		}
	}
}

// TaskManager creates a TaskManager instance from BackendConfig.
// This is the new multi-backend approach for creating task managers.
func (bc *BackendConfig) TaskManager() (TaskManager, error) {
	if !bc.Enabled {
		return nil, fmt.Errorf("backend is disabled")
	}

	switch bc.Type {
	case "nextcloud":
		// Convert BackendConfig to ConnectorConfig for backward compatibility
		u, err := url.Parse(bc.URL)
		if err != nil {
			return nil, fmt.Errorf("invalid URL for nextcloud backend: %w", err)
		}
		connConfig := ConnectorConfig{
			URL:                u,
			InsecureSkipVerify: bc.InsecureSkipVerify,
			SuppressSSLWarning: bc.SuppressSSLWarning,
		}
		return NewNextcloudBackend(connConfig)

	case "file":
		// Convert BackendConfig to ConnectorConfig for backward compatibility
		u, err := url.Parse(bc.URL)
		if err != nil {
			return nil, fmt.Errorf("invalid URL for file backend: %w", err)
		}
		connConfig := ConnectorConfig{
			URL: u,
		}
		return NewFileBackend(connConfig)

	case "git":
		// Create Git backend
		return NewGitBackend(*bc)

	case "sqlite":
		// Create SQLite backend
		return NewSQLiteBackend(*bc)

	default:
		return nil, &UnsupportedSchemeError{
			Scheme: bc.Type,
		}
	}
}

// TaskManager defines the interface for task management backends.
// Each backend (Nextcloud, File, SQLite, etc.) implements this interface
// to provide task operations with backend-specific behavior.
//
// Implementations must be safe for concurrent use if accessed from multiple goroutines.
type TaskManager interface {
	// GetTaskLists retrieves all available task lists from the backend.
	// Returns an error if the backend is unreachable or authentication fails.
	GetTaskLists() ([]TaskList, error)

	// GetTasks retrieves tasks from a specific list, optionally filtered by the provided TaskFilter.
	// The listID parameter identifies the task list to query.
	// If taskFilter is nil, all tasks are returned.
	GetTasks(listID string, taskFilter *TaskFilter) ([]Task, error)

	// FindTasksBySummary searches for tasks by summary text (case-insensitive).
	// Returns all tasks with summaries that contain the search string (exact and partial matches).
	// This is used for interactive task selection in update/delete operations.
	FindTasksBySummary(listID string, summary string) ([]Task, error)

	// AddTask creates a new task in the specified list.
	// The task.UID may be generated by the backend if not provided.
	// Returns an error if the task cannot be created.
	AddTask(listID string, task Task) error

	// UpdateTask modifies an existing task identified by task.UID.
	// All task fields will be updated to match the provided task.
	// Returns an error if the task doesn't exist or cannot be updated.
	UpdateTask(listID string, task Task) error

	// DeleteTask permanently removes a task from the specified list.
	// Returns a BackendError with IsNotFound() == true if the task doesn't exist.
	DeleteTask(listID string, taskUID string) error

	// CreateTaskList creates a new task list with the given name and optional description.
	// The color parameter is optional and may be ignored by backends that don't support it.
	// Returns the ID of the newly created list or an error if creation fails.
	CreateTaskList(name, description, color string) (string, error)

	// DeleteTaskList permanently removes a task list and all tasks within it.
	// Returns an error if the list doesn't exist or cannot be deleted.
	DeleteTaskList(listID string) error

	// RenameTaskList changes the name of an existing task list.
	// Returns an error if the list doesn't exist or the new name is already in use.
	RenameTaskList(listID, newName string) error

	// GetDeletedTaskLists retrieves all task lists that have been moved to trash.
	// This is backend-specific and may not be supported by all backends.
	// For Nextcloud, returns calendars with the DeletedAt field set.
	// Returns an empty list if trash is not supported or empty.
	GetDeletedTaskLists() ([]TaskList, error)

	// RestoreTaskList restores a deleted task list from trash.
	// The listID parameter identifies the list to restore.
	// Returns an error if the list doesn't exist in trash or cannot be restored.
	RestoreTaskList(listID string) error

	// PermanentlyDeleteTaskList permanently deletes a task list from trash.
	// This operation is irreversible and removes the list completely.
	// Returns an error if the list doesn't exist in trash or cannot be deleted.
	PermanentlyDeleteTaskList(listID string) error

	// ParseStatusFlag converts user input (abbreviations, app names, or backend names)
	// to the backend's internal status format.
	// Examples: "T" → "NEEDS-ACTION" (Nextcloud), "T" → "TODO" (File)
	// Returns an error if the status flag is invalid.
	ParseStatusFlag(statusFlag string) (string, error)

	// StatusToDisplayName converts backend-specific status to display name.
	// Returns one of: "TODO", "DONE", "PROCESSING", "CANCELLED"
	// This is used for user-facing messages and output.
	StatusToDisplayName(backendStatus string) string

	// SortTasks sorts tasks in-place according to the backend's preferred order.
	// For example, Nextcloud sorts by priority (1=highest, 0=undefined last).
	SortTasks(tasks []Task)

	// GetPriorityColor returns an ANSI color code for the given priority.
	// Returns an empty string if no color should be applied.
	// Priority range: 0-9 (0=undefined, 1=highest, 9=lowest)
	GetPriorityColor(priority int) string

	// GetBackendDisplayName returns a formatted string for display in task list headers.
	// Examples: "[nextcloud:admin@localhost]", "[sqlite → nextcloud]", "[git:gosynctasks/TODO.md]"
	// This provides user context about which backend is being used.
	GetBackendDisplayName() string

	// GetBackendType returns the backend type identifier.
	// Returns one of: "nextcloud", "git", "sqlite", "file"
	GetBackendType() string

	// GetBackendContext returns contextual details specific to the backend.
	// Examples: "admin@localhost:8080" (nextcloud), "/path/to/repo/TODO.md" (git), "/path/to/tasks.db" (sqlite)
	GetBackendContext() string
}

// DetectableBackend extends TaskManager with auto-detection capabilities.
// Backends implementing this interface can be automatically detected based on
// the current environment (e.g., git repos, file system state).
type DetectableBackend interface {
	TaskManager

	// CanDetect checks if this backend can be used in the current environment.
	// For example, a Git backend would check for a git repository and TODO.md file.
	// Returns true if the backend is detected and usable, false otherwise.
	// This method should be fast and non-destructive.
	CanDetect() (bool, error)

	// DetectionInfo returns a human-readable description of what was detected.
	// This is used for informational messages when showing detected backends.
	// Example: "Git repository with TODO.md at /path/to/repo"
	DetectionInfo() string
}

// TaskFilter specifies filtering criteria for task queries.
// All filter fields are optional (nil means no filtering on that field).
// Multiple filter criteria are combined with AND logic.
type TaskFilter struct {
	// Statuses filters tasks by their status values.
	// Should contain backend-specific status names (e.g., "NEEDS-ACTION" for Nextcloud).
	// Multiple statuses are combined with OR logic.
	Statuses *[]string

	// DueAfter filters tasks due after this time (inclusive).
	DueAfter *time.Time

	// DueBefore filters tasks due before this time (inclusive).
	DueBefore *time.Time

	// CreatedAfter filters tasks created after this time (inclusive).
	CreatedAfter *time.Time

	// CreatedBefore filters tasks created before this time (inclusive).
	CreatedBefore *time.Time
}

// StatusStringTranslateToStandardStatus converts app status names to CalDAV standard statuses.
// This function translates: TODO→NEEDS-ACTION, DONE→COMPLETED, PROCESSING→IN-PROCESS.
// Unknown statuses are passed through unchanged.
//
// Deprecated: Use TaskManager.ParseStatusFlag() instead for backend-specific translation.
func StatusStringTranslateToStandardStatus(status *[]string) *[]string {
	if status == nil {
		return nil
	}
	statusMap := map[string]string{
		"TODO":       "NEEDS-ACTION",
		"DONE":       "COMPLETED",
		"PROCESSING": "IN-PROCESS",
		"CANCELLED":  "CANCELLED",
	}

	result := make([]string, len(*status))
	for i, s := range *status {
		if normalized, ok := statusMap[strings.ToUpper(s)]; ok {
			result[i] = normalized
		} else {
			result[i] = s
		}
	}

	return &result
}

func StatusStringTranslateToAppStatus(status *[]string) *[]string {
	if status == nil {
		return nil
	}
	statusMap := map[string]string{
		"NEEDS-ACTION": "TODO",
		"COMPLETED":    "DONE",
		"IN-PROCESS":   "PROCESSING",
		"CANCELLED":    "CANCELLED",
	}

	result := make([]string, len(*status))
	for i, s := range *status {
		if normalized, ok := statusMap[strings.ToUpper(s)]; ok {
			result[i] = normalized
		} else {
			result[i] = s
		}
	}

	return &result
}

// Task represents a single task/todo item.
// The struct follows the iCalendar VTODO specification for maximum compatibility.
// Status values should use backend-specific formats (e.g., CalDAV statuses for Nextcloud).
type Task struct {
	// UID uniquely identifies the task within a backend.
	// Generated automatically if not provided during creation.
	UID string `json:"uid"`

	// Summary is the task title/name (required).
	Summary string `json:"summary"`

	// Description provides additional details about the task (optional).
	Description string `json:"description,omitempty"`

	// Status indicates the task's current state.
	// Backend-specific values: NEEDS-ACTION, IN-PROCESS, COMPLETED, CANCELLED (CalDAV)
	Status string `json:"status"`

	// Priority indicates task importance: 0-9 (0=undefined, 1=highest, 9=lowest).
	// Backend-specific interpretation (e.g., Nextcloud: 1-4=high, 5=medium, 6-9=low).
	Priority int `json:"priority"`

	// Created timestamp when the task was first created.
	Created time.Time `json:"created"`

	// Modified timestamp when the task was last modified.
	Modified time.Time `json:"modified"`

	// DueDate is the deadline for task completion (optional).
	DueDate *time.Time `json:"due_date,omitempty"`

	// StartDate is when work on the task should begin (optional).
	StartDate *time.Time `json:"start_date,omitempty"`

	// Completed timestamp when the task was marked as COMPLETED (optional).
	Completed *time.Time `json:"completed,omitempty"`

	// Categories are task tags/labels for organization (optional).
	Categories []string `json:"categories,omitempty"`

	// ParentUID links this task as a subtask of another task (optional).
	ParentUID string `json:"parent_uid,omitempty"`
}

// String returns a basic formatted string representation of the task.
// For more control over formatting, use FormatWithView.
func (t Task) String() string {
	return t.FormatWithView("basic", nil, "2006-01-02")
}

// FormatWithView formats the task for display with customizable view options.
//
// Parameters:
//   - view: "basic" (summary + status) or "all" (includes metadata like dates and priority)
//   - backend: TaskManager for priority coloring (can be nil)
//   - dateFormat: Go time format string for date display
//
// The output includes ANSI color codes for terminal display:
//   - Status symbols: ✓ (done), ● (in progress), ✗ (cancelled), ○ (todo)
//   - Priority colors: determined by backend.GetPriorityColor()
//   - Start date colors: cyan (past), yellow (within 3 days), gray (future)
//   - Due date colors: red (overdue), yellow (due soon), gray (future)
func (t Task) FormatWithView(view string, backend TaskManager, dateFormat string) string {
	return t.formatWithIndent(view, backend, dateFormat, 0)
}

// FormatWithIndentLevel formats the task for display with a specific indentation level.
// This is useful for displaying hierarchical task structures where subtasks should be indented.
//
// Parameters:
//   - view: "basic" (summary + status) or "all" (includes metadata like dates and priority)
//   - backend: TaskManager for priority coloring (can be nil)
//   - dateFormat: Go time format string for date display
//   - indentLevel: number of indentation levels (0 = no indent, 1 = one level, etc.)
func (t Task) FormatWithIndentLevel(view string, backend TaskManager, dateFormat string, indentLevel int) string {
	return t.formatWithIndent(view, backend, dateFormat, indentLevel)
}

// formatWithIndent formats the task for display with indentation for subtasks.
// The indentLevel parameter specifies how many levels deep the task is in the hierarchy.
func (t Task) formatWithIndent(view string, backend TaskManager, dateFormat string, indentLevel int) string {
	var result strings.Builder

	// Convert backend-specific status to canonical display name
	displayStatus := t.Status
	if backend != nil {
		displayStatus = backend.StatusToDisplayName(t.Status)
	}

	// Status indicator (using canonical status names)
	statusColor := ""
	statusSymbol := "○"
	switch displayStatus {
	case "DONE":
		statusColor = "\033[32m" // Green
		statusSymbol = "✓"
	case "PROCESSING":
		statusColor = "\033[33m" // Yellow
		statusSymbol = "●"
	case "CANCELLED":
		statusColor = "\033[31m" // Red
		statusSymbol = "✗"
	default: // TODO or any other status
		statusColor = "\033[37m" // White
		statusSymbol = "○"
	}

	// Get priority color from backend
	priorityColor := ""
	if t.Priority > 0 && backend != nil {
		priorityColor = backend.GetPriorityColor(t.Priority)
	}

	// Start date
	startStr := ""
	if t.StartDate != nil {
		now := time.Now()
		start := *t.StartDate
		hoursDiff := start.Sub(now).Hours()

		if start.Before(now) || start.Equal(now) {
			// Past/present: work should have begun (cyan)
			startStr = fmt.Sprintf(" \033[36m(starts: %s)\033[0m", start.Format(dateFormat))
		} else if hoursDiff <= 72 { // Within 3 days (inclusive)
			// Within 3 days (yellow) - includes exactly 72 hours
			startStr = fmt.Sprintf(" \033[33m(starts: %s)\033[0m", start.Format(dateFormat))
		} else {
			// Future beyond 3 days (gray)
			startStr = fmt.Sprintf(" \033[90m(starts: %s)\033[0m", start.Format(dateFormat))
		}
	}

	// Due date
	dueStr := ""
	if t.DueDate != nil {
		now := time.Now()
		due := *t.DueDate
		if due.Before(now) {
			dueStr = fmt.Sprintf(" \033[31m(overdue: %s)\033[0m", due.Format(dateFormat))
		} else if due.Sub(now).Hours() < 24 {
			dueStr = fmt.Sprintf(" \033[33m(due: %s)\033[0m", due.Format(dateFormat))
		} else {
			dueStr = fmt.Sprintf(" \033[90m(due: %s)\033[0m", due.Format(dateFormat))
		}
	}

	// Main line: status + colored summary (by priority) + start + due
	// Add indentation for subtasks (2 spaces per level, plus the base 2 spaces)
	indent := strings.Repeat("  ", indentLevel)
	summaryColor := priorityColor
	if summaryColor == "" {
		summaryColor = "\033[1m" // Bold if no priority color
	} else {
		summaryColor = summaryColor + "\033[1m" // Bold + priority color
	}
	result.WriteString(fmt.Sprintf("  %s%s%s\033[0m %s%s\033[0m%s%s\n",
		indent, statusColor, statusSymbol, summaryColor, t.Summary, startStr, dueStr))

	// Description (if present)
	if t.Description != "" {
		desc := strings.ReplaceAll(t.Description, "\n", " ")
		if len(desc) > 70 {
			desc = desc[:67] + "..."
		}
		result.WriteString(fmt.Sprintf("     %s\033[2m%s\033[0m\n", indent, desc))
	}

	// Metadata line: created, modified, priority (only for "all" view)
	if view == "all" {
		var metadata []string

		if !t.Created.IsZero() {
			metadata = append(metadata, fmt.Sprintf("Created: %s", t.Created.Format(dateFormat)))
		}

		if !t.Modified.IsZero() {
			metadata = append(metadata, fmt.Sprintf("Modified: %s", t.Modified.Format(dateFormat)))
		}

		if t.Priority > 0 {
			metadata = append(metadata, fmt.Sprintf("Priority: %d", t.Priority))
		}

		if len(metadata) > 0 {
			result.WriteString(fmt.Sprintf("     %s\033[2m%s\033[0m\n", indent, strings.Join(metadata, " | ")))
		}
	}

	return result.String()
}

// TaskWithLevel represents a task and its hierarchical depth level.
// This is used when displaying tasks in a hierarchy where subtasks are indented.
type TaskWithLevel struct {
	Task  Task
	Level int
}

// OrganizeTasksHierarchically organizes tasks into a hierarchical structure where
// subtasks appear immediately after their parent tasks with appropriate indentation levels.
// Tasks without parents (or whose parents don't exist in the list) are treated as root tasks.
func OrganizeTasksHierarchically(tasks []Task) []TaskWithLevel {
	if len(tasks) == 0 {
		return nil
	}

	// Build maps for quick lookups
	taskByUID := make(map[string]*Task)
	childrenMap := make(map[string][]Task)

	for i := range tasks {
		task := &tasks[i]
		taskByUID[task.UID] = task

		if task.ParentUID != "" {
			childrenMap[task.ParentUID] = append(childrenMap[task.ParentUID], *task)
		}
	}

	// Find root tasks (tasks without parents or whose parents don't exist)
	var rootTasks []Task
	for _, task := range tasks {
		if task.ParentUID == "" || taskByUID[task.ParentUID] == nil {
			rootTasks = append(rootTasks, task)
		}
	}

	// Recursively build the hierarchical list
	var result []TaskWithLevel
	visited := make(map[string]bool)

	var addTaskWithChildren func(task Task, level int)
	addTaskWithChildren = func(task Task, level int) {
		// Prevent infinite loops in case of circular references
		if visited[task.UID] {
			return
		}
		visited[task.UID] = true

		// Add the current task
		result = append(result, TaskWithLevel{Task: task, Level: level})

		// Add children recursively
		if children, ok := childrenMap[task.UID]; ok {
			for _, child := range children {
				addTaskWithChildren(child, level+1)
			}
		}
	}

	// Process all root tasks
	for _, task := range rootTasks {
		addTaskWithChildren(task, 0)
	}

	return result
}

// TaskList represents a collection/category of tasks.
// In CalDAV, this corresponds to a calendar that supports VTODO components.
// Each backend may have its own interpretation (e.g., file directory, database table).
type TaskList struct {
	// ID uniquely identifies the list within the backend.
	ID string `json:"id"`

	// Name is the human-readable list name.
	Name string `json:"name"`

	// Description provides additional context about the list (optional).
	Description string `json:"description,omitempty"`

	// Color is a hex color code for UI display (optional, e.g., "#0082c9").
	Color string `json:"color,omitempty"`

	// URL is the backend-specific URL to access the list (e.g., CalDAV URL).
	URL string `json:"url"`

	// CTags is a synchronization token that changes when the list is modified.
	// Used for efficient sync operations (CalDAV-specific, optional).
	CTags string `json:"ctags,omitempty"`

	// DeletedAt indicates when the list was deleted (moved to trash).
	// Empty string means the list is not deleted.
	// Used by Nextcloud to track trashed calendars (Nextcloud-specific, optional).
	DeletedAt string `json:"deleted_at,omitempty"`
}

func (t TaskList) String() string {
	return t.StringWithWidth(80) // Default width
}

func (t TaskList) StringWithWidth(termWidth int) string {
	var result strings.Builder

	// Calculate border width
	borderWidth := termWidth - 2
	if borderWidth < 40 {
		borderWidth = 40
	}
	if borderWidth > 100 {
		borderWidth = 100
	}

	// Build the title text
	titleText := "─ " + t.Name
	if t.Description != "" {
		titleText += " - " + t.Description
	}
	titleText += " "

	// Calculate padding for header
	headerPadding := borderWidth - len(titleText) - 1
	if headerPadding < 0 {
		headerPadding = 0
	}

	// Top border with corner and title
	result.WriteString(fmt.Sprintf("\n\033[1;36m┌%s%s┐\033[0m\n", titleText, strings.Repeat("─", headerPadding)))

	return result.String()
}

func (t TaskList) BottomBorder() string {
	return t.BottomBorderWithWidth(80) // Default width
}

func (t TaskList) BottomBorderWithWidth(termWidth int) string {
	// Calculate border width
	borderWidth := termWidth - 2
	if borderWidth < 40 {
		borderWidth = 40
	}
	if borderWidth > 100 {
		borderWidth = 100
	}

	// Bottom border
	return fmt.Sprintf("\033[1;36m└%s┘\033[0m\n", strings.Repeat("─", borderWidth))
}

// StringWithBackend returns the list header with backend information displayed on the right side.
// The backend parameter can be nil, in which case no backend info is shown.
func (t TaskList) StringWithBackend(backend TaskManager) string {
	return t.StringWithWidthAndBackend(80, backend)
}

// StringWithWidthAndBackend formats the list header with backend information.
// The backend info is positioned on the right side of the header, adapting to terminal width.
func (t TaskList) StringWithWidthAndBackend(termWidth int, backend TaskManager) string {
	// If no backend provided, fall back to standard display
	if backend == nil {
		return t.StringWithWidth(termWidth)
	}

	var result strings.Builder

	// Calculate border width
	borderWidth := termWidth - 2
	if borderWidth < 40 {
		borderWidth = 40
	}
	if borderWidth > 100 {
		borderWidth = 100
	}

	// Get backend display name
	backendInfo := backend.GetBackendDisplayName()

	// Build the title text
	titleText := "─ " + t.Name
	if t.Description != "" {
		titleText += " - " + t.Description
	}
	titleText += " "

	// Calculate available space for padding between title and backend info
	// Format: ┌─ Title ──────── [backend] ┐
	titleLen := len(titleText)
	backendLen := len(backendInfo)
	totalContentLen := titleLen + backendLen + 2 // +2 for space and corner

	if totalContentLen >= borderWidth {
		// Not enough space, truncate description or show without backend
		maxTitleLen := borderWidth - backendLen - 3
		if maxTitleLen < 10 {
			// If still not enough space, show without backend info
			return t.StringWithWidth(termWidth)
		}
		// Truncate title
		if len(titleText) > maxTitleLen {
			titleText = titleText[:maxTitleLen-3] + "... "
		}
	}

	// Calculate padding between title and backend info
	paddingLen := borderWidth - len(titleText) - len(backendInfo) - 1
	if paddingLen < 1 {
		paddingLen = 1
	}

	// Top border with corner, title, padding, backend info
	result.WriteString(fmt.Sprintf("\n\033[1;36m┌%s%s%s┐\033[0m\n",
		titleText,
		strings.Repeat("─", paddingLen),
		backendInfo))

	return result.String()
}
