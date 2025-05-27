package connectors

import (
	"net/http"
)

type NextcloudConnector struct {
	ConnectorConfig ConnectorConfig
	URL             string `json:"url" validate:"required,url"`
	Username        string `json:"username" validate:"required"`
	Password        string `json:"password" validate:"required"`
	Timeout         int    `json:"timeout,omitempty"`
	client          *http.Client
}

func (nc *NextcloudConnector) GetTaskLists() ([]TaskList, error) {
	// Nextcloud CalDAV implementation
}

func (nc *NextcloudConnector) GetTasks(listID string) ([]Task, error) {
	// Nextcloud VTODO implementation
}
