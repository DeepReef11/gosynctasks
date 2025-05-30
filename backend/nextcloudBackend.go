package backend

import (
	"net/url"
)

type NextcloudBackend struct {
}

func (nB *NextcloudBackend) GetTaskLists() ([]TaskList, error) {
	return nil, nil

}

func (nB *NextcloudBackend) GetTasks(listID string) ([]Task, error) {
	return nil, nil
}

func NewNextcloudBackend(url *url.URL) (TaskManager, error) {
    nB := &NextcloudBackend{}
	return nB, nil
}
