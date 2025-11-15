package backend

// This file contains shared test helpers and mocks used across backend tests.
// These are available to all _test.go files in the backend package.

// MockBackend implements TaskManager for testing
type MockBackend struct {
	lists         []TaskList
	tasks         map[string][]Task // listID -> tasks
	addTaskErr    error
	updateTaskErr error
	deleteTaskErr error
	name          string // For tests that need to identify the mock
}

// NewMockBackend creates a new mock backend instance
func NewMockBackend() *MockBackend {
	return &MockBackend{
		lists: []TaskList{},
		tasks: make(map[string][]Task),
	}
}

// NewMockBackendWithName creates a new mock backend with a name
func NewMockBackendWithName(name string) *MockBackend {
	mb := NewMockBackend()
	mb.name = name
	return mb
}

func (mb *MockBackend) GetTaskLists() ([]TaskList, error) {
	return mb.lists, nil
}

func (mb *MockBackend) GetTasks(listID string, filter *TaskFilter) ([]Task, error) {
	tasks, ok := mb.tasks[listID]
	if !ok {
		return []Task{}, nil
	}
	return tasks, nil
}

func (mb *MockBackend) FindTasksBySummary(listID string, summary string) ([]Task, error) {
	return []Task{}, nil
}

func (mb *MockBackend) AddTask(listID string, task Task) error {
	if mb.addTaskErr != nil {
		return mb.addTaskErr
	}

	tasks := mb.tasks[listID]
	tasks = append(tasks, task)
	mb.tasks[listID] = tasks
	return nil
}

func (mb *MockBackend) UpdateTask(listID string, task Task) error {
	if mb.updateTaskErr != nil {
		return mb.updateTaskErr
	}

	tasks := mb.tasks[listID]
	for i, t := range tasks {
		if t.UID == task.UID {
			tasks[i] = task
			mb.tasks[listID] = tasks
			return nil
		}
	}
	return NewBackendError("UpdateTask", 404, "task not found")
}

func (mb *MockBackend) DeleteTask(listID string, taskUID string) error {
	if mb.deleteTaskErr != nil {
		return mb.deleteTaskErr
	}

	tasks := mb.tasks[listID]
	for i, t := range tasks {
		if t.UID == taskUID {
			tasks = append(tasks[:i], tasks[i+1:]...)
			mb.tasks[listID] = tasks
			return nil
		}
	}
	return NewBackendError("DeleteTask", 404, "task not found")
}

func (mb *MockBackend) CreateTaskList(name, description, color string) (string, error) {
	listID := generateUID()
	mb.lists = append(mb.lists, TaskList{
		ID:    listID,
		Name:  name,
		Color: color,
		CTags: "ctag-initial",
	})
	mb.tasks[listID] = []Task{}
	return listID, nil
}

func (mb *MockBackend) DeleteTaskList(listID string) error {
	delete(mb.tasks, listID)
	for i, list := range mb.lists {
		if list.ID == listID {
			mb.lists = append(mb.lists[:i], mb.lists[i+1:]...)
			return nil
		}
	}
	return NewBackendError("DeleteTaskList", 404, "list not found")
}

func (mb *MockBackend) RenameTaskList(listID, newName string) error {
	for i, list := range mb.lists {
		if list.ID == listID {
			mb.lists[i].Name = newName
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
