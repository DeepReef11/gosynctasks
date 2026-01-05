package backend

import (
	"fmt"
	"math/rand"
	"time"
)

// This file contains shared test helpers and mocks used across backend tests.
// These are available to all _test.go files in the backend package.

// generateUID generates a unique identifier for testing
func generateUID() string {
	return fmt.Sprintf("task-%d-%s", time.Now().Unix(), randomString(8))
}

// randomString generates a random string of given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// MockBackend implements TaskManager for testing
type MockBackend struct {
	Lists         []TaskList
	Tasks         map[string][]Task // listID -> tasks
	AddTaskErr    error
	UpdateTaskErr error
	DeleteTaskErr error
	Name          string // For tests that need to identify the mock
}

// NewMockBackend creates a new mock backend instance
func NewMockBackend() *MockBackend {
	return &MockBackend{
		Lists: []TaskList{},
		Tasks: make(map[string][]Task),
	}
}

// NewMockBackendWithName creates a new mock backend with a name
func NewMockBackendWithName(name string) *MockBackend {
	mb := NewMockBackend()
	mb.Name = name
	return mb
}

func (mb *MockBackend) GetTaskLists() ([]TaskList, error) {
	return mb.Lists, nil
}

func (mb *MockBackend) GetTasks(listID string, filter *TaskFilter) ([]Task, error) {
	tasks, ok := mb.Tasks[listID]
	if !ok {
		return []Task{}, nil
	}
	return tasks, nil
}

func (mb *MockBackend) FindTasksBySummary(listID string, summary string) ([]Task, error) {
	return []Task{}, nil
}

func (mb *MockBackend) AddTask(listID string, task Task) (string, error) {
	if mb.AddTaskErr != nil {
		return "", mb.AddTaskErr
	}

	if task.UID == "" {
		task.UID = "mock-task-id"
	}

	tasks := mb.Tasks[listID]
	tasks = append(tasks, task)
	mb.Tasks[listID] = tasks
	return task.UID, nil
}

func (mb *MockBackend) UpdateTask(listID string, task Task) error {
	if mb.UpdateTaskErr != nil {
		return mb.UpdateTaskErr
	}

	tasks := mb.Tasks[listID]
	for i, t := range tasks {
		if t.UID == task.UID {
			tasks[i] = task
			mb.Tasks[listID] = tasks
			return nil
		}
	}
	return NewBackendError("UpdateTask", 404, "task not found")
}

func (mb *MockBackend) DeleteTask(listID string, taskUID string) error {
	if mb.DeleteTaskErr != nil {
		return mb.DeleteTaskErr
	}

	tasks := mb.Tasks[listID]
	for i, t := range tasks {
		if t.UID == taskUID {
			tasks = append(tasks[:i], tasks[i+1:]...)
			mb.Tasks[listID] = tasks
			return nil
		}
	}
	return NewBackendError("DeleteTask", 404, "task not found")
}

func (mb *MockBackend) CreateTaskList(name, description, color string) (string, error) {
	listID := generateUID()
	mb.Lists = append(mb.Lists, TaskList{
		ID:    listID,
		Name:  name,
		Color: color,
		CTags: "ctag-initial",
	})
	mb.Tasks[listID] = []Task{}
	return listID, nil
}

func (mb *MockBackend) DeleteTaskList(listID string) error {
	delete(mb.Tasks, listID)
	for i, list := range mb.Lists {
		if list.ID == listID {
			mb.Lists = append(mb.Lists[:i], mb.Lists[i+1:]...)
			return nil
		}
	}
	return NewBackendError("DeleteTaskList", 404, "list not found")
}

func (mb *MockBackend) RenameTaskList(listID, newName string) error {
	for i, list := range mb.Lists {
		if list.ID == listID {
			mb.Lists[i].Name = newName
			return nil
		}
	}
	return NewBackendError("RenameTaskList", 404, "list not found")
}

func (mb *MockBackend) GetDeletedTaskLists() ([]TaskList, error) {
	return []TaskList{}, nil
}

func (mb *MockBackend) RestoreTaskList(listID string) error {
	return nil
}

func (mb *MockBackend) PermanentlyDeleteTaskList(listID string) error {
	return nil
}

func (mb *MockBackend) ParseStatusFlag(statusFlag string) (string, error) {
	// Simple mock implementation
	switch statusFlag {
	case "T", "TODO":
		return "NEEDS-ACTION", nil
	case "D", "DONE":
		return "COMPLETED", nil
	case "P", "PROCESSING":
		return "IN-PROCESS", nil
	case "C", "CANCELLED":
		return "CANCELLED", nil
	default:
		return statusFlag, nil
	}
}

func (mb *MockBackend) StatusToDisplayName(backendStatus string) string {
	switch backendStatus {
	case "NEEDS-ACTION":
		return "TODO"
	case "COMPLETED":
		return "DONE"
	case "IN-PROCESS":
		return "PROCESSING"
	case "CANCELLED":
		return "CANCELLED"
	default:
		return backendStatus
	}
}

func (mb *MockBackend) SortTasks(tasks []Task) {
	// No-op for mock
}

func (mb *MockBackend) GetPriorityColor(priority int) string {
	return ""
}

func (mb *MockBackend) GetBackendDisplayName() string {
	if mb.Name != "" {
		return "[mock:" + mb.Name + "]"
	}
	return "[mock]"
}

func (mb *MockBackend) GetBackendType() string {
	return "mock"
}

func (mb *MockBackend) GetBackendContext() string {
	if mb.Name != "" {
		return mb.Name
	}
	return "mock-backend"
}

// MockDetectableBackend is a mock that implements DetectableBackend
type MockDetectableBackend struct {
	MockBackend
	canDetect     bool
	detectionInfo string
}

// NewMockDetectableBackend creates a new mock detectable backend
func NewMockDetectableBackend(canDetect bool, detectionInfo string) *MockDetectableBackend {
	return &MockDetectableBackend{
		MockBackend:   *NewMockBackend(),
		canDetect:     canDetect,
		detectionInfo: detectionInfo,
	}
}

func (m *MockDetectableBackend) CanDetect() (bool, error) {
	return m.canDetect, nil
}

func (m *MockDetectableBackend) DetectionInfo() string {
	return m.detectionInfo
}
