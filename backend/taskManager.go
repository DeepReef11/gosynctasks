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

// Base config struct
type ConnectorConfig struct {
	URL                *url.URL `json:"url"`
	InsecureSkipVerify bool     `json:"insecure_skip_verify,omitempty"` // WARNING: Only use for self-signed certificates in dev
	SuppressSSLWarning bool     `json:"suppress_ssl_warning,omitempty"` // Suppress SSL warning when InsecureSkipVerify is true
	// Type     string `json:"type" validate:"required,oneof=nextcloud local"`
	//  Timeout  int    `json:"timeout,omitempty"`
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

type TaskManager interface {
	GetTaskLists() ([]TaskList, error)
	GetTasks(listID string, taskFilter *TaskFilter) ([]Task, error)
	FindTasksBySummary(listID string, summary string) ([]Task, error) // Search tasks by summary (exact and partial matches)
	AddTask(listID string, task Task) (error)
	UpdateTask(listID string, task Task) (error)
	SortTasks(tasks []Task) // Sort tasks according to backend's preferred order
	GetPriorityColor(priority int) string // Get ANSI color code for priority
	// Tasks() iter.Seq[Task]
	// Task(string) (Task, bool)
	// Create(Task) error
}

type TaskFilter struct {
	Statuses     *[]string // "NEEDS-ACTION", "COMPLETED", "IN-PROCESS", "CANCELLED"
	DueAfter     *time.Time
	DueBefore    *time.Time
	CreatedAfter *time.Time
}

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

type Task struct {
	UID         string     `json:"uid"`
	Summary     string     `json:"summary"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`   // NEEDS-ACTION, IN-PROCESS, COMPLETED, CANCELLED
	Priority    int        `json:"priority"` // 0-9 (0=undefined, 1=highest, 9=lowest)
	Created     time.Time  `json:"created"`
	Modified    time.Time  `json:"modified"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	Completed   *time.Time `json:"completed,omitempty"`
	Categories  []string   `json:"categories,omitempty"`
	ParentUID   string     `json:"parent_uid,omitempty"` // For subtasks
}

func (t Task) String() string {
	return t.FormatWithView("basic", nil, "2006-01-02")
}

func (t Task) FormatWithView(view string, backend TaskManager, dateFormat string) string {
	var result strings.Builder

	// Status indicator
	statusColor := ""
	statusSymbol := "○"
	switch t.Status {
	case "COMPLETED":
		statusColor = "\033[32m" // Green
		statusSymbol = "✓"
	case "IN-PROCESS":
		statusColor = "\033[33m" // Yellow
		statusSymbol = "●"
	case "CANCELLED":
		statusColor = "\033[31m" // Red
		statusSymbol = "✗"
	default: // NEEDS-ACTION
		statusColor = "\033[37m" // White
		statusSymbol = "○"
	}

	// Get priority color from backend
	priorityColor := ""
	if t.Priority > 0 && backend != nil {
		priorityColor = backend.GetPriorityColor(t.Priority)
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

	// Main line: status + colored summary (by priority) + due
	summaryColor := priorityColor
	if summaryColor == "" {
		summaryColor = "\033[1m" // Bold if no priority color
	} else {
		summaryColor = summaryColor + "\033[1m" // Bold + priority color
	}
	result.WriteString(fmt.Sprintf("  %s%s\033[0m %s%s\033[0m%s\n",
		statusColor, statusSymbol, summaryColor, t.Summary, dueStr))

	// Description (if present)
	if t.Description != "" {
		desc := strings.ReplaceAll(t.Description, "\n", " ")
		if len(desc) > 70 {
			desc = desc[:67] + "..."
		}
		result.WriteString(fmt.Sprintf("     \033[2m%s\033[0m\n", desc))
	}

	// Metadata line: UID, created, modified, priority (only for "all" view)
	if view == "all" {
		var metadata []string

		// Always show UID in all view
		if t.UID != "" {
			metadata = append(metadata, fmt.Sprintf("UID: %s", t.UID))
		}

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
			result.WriteString(fmt.Sprintf("     \033[2m%s\033[0m\n", strings.Join(metadata, " | ")))
		}
	}

	return result.String()
}

type TaskList struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
	URL         string `json:"url"`
	CTags       string `json:"ctags,omitempty"`
}

func (t TaskList) String() string {
	var result strings.Builder

	// Build the title text
	title := t.Name
	if t.Description != "" {
		title = fmt.Sprintf(" - %s", t.Description)
	} else {
		title = ""
	}

	// Top border with corner and title
	result.WriteString(fmt.Sprintf("\n\033[1;36m┌─ %s%s ", t.Name, title))
	result.WriteString(strings.Repeat("─", 50))
	result.WriteString("┐\033[0m\n")

	return result.String()
}

func (t TaskList) BottomBorder() string {
	// Bottom border
	return fmt.Sprintf("\033[1;36m└%s┘\033[0m\n", strings.Repeat("─", 60))
}
