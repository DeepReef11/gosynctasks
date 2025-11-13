package backend

import (
	"fmt"
	"strings"
)

type FileBackend struct {
	Connector ConnectorConfig
}

func (fB *FileBackend) GetTaskLists() ([]TaskList, error) {
	return nil, nil
}

func (fB *FileBackend) GetTasks(listID string, taskFilter *TaskFilter) ([]Task, error) {
	return nil, nil
}

func (fB *FileBackend) FindTasksBySummary(listID string, summary string) ([]Task, error) {
	return nil, nil
}

func (fB *FileBackend) AddTask(listID string, task Task) error {
	return nil
}

func (fB *FileBackend) UpdateTask(listID string, task Task) error {
	return nil
}

func (fB *FileBackend) DeleteTask(listID string, taskUID string) error {
	return nil
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

func (fB *FileBackend) SortTasks(tasks []Task) {
	// File backend: no specific sorting
}

func (fB *FileBackend) GetPriorityColor(priority int) string {
	// File backend: default color scheme
	if priority >= 1 && priority <= 4 {
		return "\033[31m"
	}
	return ""
}

func NewFileBackend(connectorConfig ConnectorConfig) (TaskManager, error) {
	return &FileBackend{
		Connector: connectorConfig,
	}, nil
}
