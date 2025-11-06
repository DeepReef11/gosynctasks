package backend

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
