package connectors

import (
	"net/http"
)

type NextcloudBackend struct {
	ConnectorConfig ConnectorConfig
	URL             string `json:"url" validate:"required,url"`
	Username        string `json:"username" validate:"required"`
	Password        string `json:"password" validate:"required"`
	client          *http.Client
}


