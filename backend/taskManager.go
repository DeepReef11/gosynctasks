package backend

import (
	"time"
)

type TaskManager interface {
	GetTaskLists() ([]TaskList, error)
	GetTasks(listID string) ([]Task, error)
	// Tasks() iter.Seq[Task]
	// Task(string) (Task, bool)
	// Create(Task) error
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

type TaskList struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
	URL         string `json:"url"`
	CTags       string `json:"ctags,omitempty"`
}
