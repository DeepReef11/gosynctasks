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

func (fB *FileBackend) AddTask(listID string, task Task) error {
	return nil
}

func NewFileBackend(connectorConfig ConnectorConfig) (TaskManager, error) {
	return &FileBackend{
		Connector: connectorConfig,
	}, nil
}
