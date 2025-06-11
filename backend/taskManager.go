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
	URL *url.URL `json:"url"`
	// Type     string `json:"type" validate:"required,oneof=nextcloud local"`
	//  Timeout  int    `json:"timeout,omitempty"`
}

func (c *ConnectorConfig) UnmarshalJSON(data []byte) error {
	type ConnConfig ConnectorConfig

	tmp := struct {
		*ConnConfig
		URL string `json:"url"`
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

	return nil
}

func (c *ConnectorConfig) TaskManager() (TaskManager, error) {
	switch c.URL.Scheme {
	case "nextcloud":
		return NewNextcloudBackend(c.URL)
	case "file":
		return NewFileBackend(c.URL)
	default:
		return nil, &UnsupportedSchemeError{
			Scheme: c.URL.Scheme,
		}
	}
}

type TaskManager interface {
	GetTaskLists() ([]TaskList, error)
	GetTasks(listID string, taskFilter *TaskFilter) ([]Task, error)
	AddTask(listID string, task Task) (error)
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
	data, _ := json.Marshal(t)
	var m map[string]any
	json.Unmarshal(data, &m)

	var result strings.Builder

	result.WriteString("\n___\n")
	// Priority fields
	priority := []string{"uid", "summary", "description", "status", "created", "modified"}
	for _, key := range priority {
		if v, exists := m[key]; exists && v != nil {
			if key == "summary" {
				result.WriteString(fmt.Sprintf("\033[1;34m%s: %v\033[0m\n", key, v)) // Green bold
			} else {
				result.WriteString(fmt.Sprintf("%s: %v\n", key, v))
			}
			delete(m, key)
		}
	}

	// Remaining fields
	for k, v := range m {
		if v != nil {
			result.WriteString(fmt.Sprintf("%s: %v\n", k, v))
		}
	}
	result.WriteString("‾‾‾")

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
	data, _ := json.Marshal(t)
	var m map[string]any
	json.Unmarshal(data, &m)

	var result strings.Builder

	result.WriteString("\n___\n")
	// Priority fields
	priority := []string{"id", "name", "description", "url", "color"}
	for _, key := range priority {
		if v, exists := m[key]; exists && v != nil {
			if key == "name" {
				result.WriteString(fmt.Sprintf("\033[1;34m%s: %v\033[0m\n", key, v)) // Green bold
			} else {
				result.WriteString(fmt.Sprintf("%s: %v\n", key, v))
			}
			delete(m, key)
		}
	}

	// Remaining fields
	for k, v := range m {
		if v != nil {
			result.WriteString(fmt.Sprintf("%s: %v\n", k, v))
		}
	}
	result.WriteString("‾‾‾")

	return result.String()
}
