package file

import (
	"gosynctasks/backend"
	"fmt"
	"net/url"
	"strings"
)

func init() {
	// Register File backend for URL scheme "file://"
	backend.RegisterScheme("file", NewFileBackend)

	// Register File backend for config type "file"
	backend.RegisterType("file", newFileBackendFromBackendConfig)
}

// newFileBackendFromBackendConfig creates a File backend from BackendConfig
func newFileBackendFromBackendConfig(bc backend.BackendConfig) (backend.TaskManager, error) {
	// Convert BackendConfig to ConnectorConfig
	u, err := url.Parse(bc.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL for file backend: %w", err)
	}

	connConfig := backend.ConnectorConfig{
		URL: u,
	}

	return NewFileBackend(connConfig)
}

type FileBackend struct {
	Connector backend.ConnectorConfig
}

func (fB *FileBackend) GetTaskLists() ([]backend.TaskList, error) {
	return nil, nil
}

func (fB *FileBackend) GetTasks(listID string, taskFilter *backend.TaskFilter) ([]backend.Task, error) {
	return nil, nil
}

func (fB *FileBackend) FindTasksBySummary(listID string, summary string) ([]backend.Task, error) {
	return nil, nil
}

func (fB *FileBackend) AddTask(listID string, task backend.Task) error {
	return nil
}

func (fB *FileBackend) UpdateTask(listID string, task backend.Task) error {
	return nil
}

func (fB *FileBackend) DeleteTask(listID string, taskUID string) error {
	return nil
}

func (fB *FileBackend) CreateTaskList(name, description, color string) (string, error) {
	return "", fmt.Errorf("FileBackend.CreateTaskList not yet implemented")
}

func (fB *FileBackend) DeleteTaskList(listID string) error {
	return fmt.Errorf("FileBackend.DeleteTaskList not yet implemented")
}

func (fB *FileBackend) RenameTaskList(listID, newName string) error {
	return fmt.Errorf("FileBackend.RenameTaskList not yet implemented")
}

func (fB *FileBackend) GetDeletedTaskLists() ([]backend.TaskList, error) {
	// FileBackend doesn't support trash functionality
	return []backend.TaskList{}, nil
}

func (fB *FileBackend) RestoreTaskList(listID string) error {
	return fmt.Errorf("FileBackend.RestoreTaskList not supported")
}

func (fB *FileBackend) PermanentlyDeleteTaskList(listID string) error {
	return fmt.Errorf("FileBackend.PermanentlyDeleteTaskList not supported")
}

func (fB *FileBackend) ParseStatusFlag(statusFlag string) (string, error) {
	if statusFlag == "" {
		return "", fmt.Errorf("status flag cannot be empty")
	}

	upperStatus := strings.ToUpper(statusFlag)

	// FileBackend uses display names directly (TODO, DONE, PROCESSING, CANCELLED)
	switch upperStatus {
	case "T", "TODO":
		return "TODO", nil
	case "D", "DONE":
		return "DONE", nil
	case "P", "PROCESSING":
		return "PROCESSING", nil
	case "C", "CANCELLED":
		return "CANCELLED", nil
	default:
		return "", fmt.Errorf("invalid status: %s (valid: TODO/T, DONE/D, PROCESSING/P, CANCELLED/C)", statusFlag)
	}
}

func (fB *FileBackend) StatusToDisplayName(backendStatus string) string {
	// FileBackend uses canonical display names internally
	// Normalize to ensure we return one of the canonical names
	switch strings.ToUpper(backendStatus) {
	case "TODO":
		return "TODO"
	case "DONE":
		return "DONE"
	case "PROCESSING":
		return "PROCESSING"
	case "CANCELLED":
		return "CANCELLED"
	default:
		// Unknown status, return as-is (uppercase)
		return strings.ToUpper(backendStatus)
	}
}

func (fB *FileBackend) SortTasks(tasks []backend.Task) {
	// File backend: no specific sorting
}

func (fB *FileBackend) GetPriorityColor(priority int) string {
	// File backend: default color scheme
	if priority >= 1 && priority <= 4 {
		return "\033[31m"
	}
	return ""
}

func (fB *FileBackend) GetBackendDisplayName() string {
	path := ""
	if fB.Connector.URL != nil {
		path = fB.Connector.URL.Path
	}
	return fmt.Sprintf("[file:%s]", path)
}

func (fB *FileBackend) GetBackendType() string {
	return "file"
}

func (fB *FileBackend) GetBackendContext() string {
	if fB.Connector.URL != nil {
		return fB.Connector.URL.Path
	}
	return ""
}

func NewFileBackend(connectorConfig backend.ConnectorConfig) (backend.TaskManager, error) {
	return &FileBackend{
		Connector: connectorConfig,
	}, nil
}
