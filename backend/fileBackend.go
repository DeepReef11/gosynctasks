package backend

import (
	"net/url"
)

type FileBackend struct {
}

func (nB *FileBackend) GetTaskLists() ([]TaskList, error) {
	return nil, nil

}

func (nB *FileBackend) GetTasks() ([]Task, error) {
	return nil, nil
}

func NewFileBackend(url *url.URL) (TaskManager, error) {
	return nil, nil
}
