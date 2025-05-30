package connectors

import (
	// "net/http"
	// "gosynctasks/connectors"
	"gosynctasks/backend"
	"encoding/json"
	"fmt"
	"net/url"
	// "time"
	// "strings"
)

type UnsupportedSchemeError struct {
	Scheme string
}

func (e *UnsupportedSchemeError) Error() string {
	return fmt.Sprintf("unsupported scheme: %q", e.Scheme)
}


// type TaskConnector interface {
//    GetTaskLists() ([]TaskList, error)
//     GetTasks(listID string) ([]Task, error)
//     // CreateTask(listID string, task Task) error
//     // UpdateTask(listID string, task Task) error
//     // DeleteTask(listID string, taskID string) error
// }

// Base config struct
type ConnectorConfig struct {
	URL *url.URL `json:"url"`
	// Type     string `json:"type" validate:"required,oneof=nextcloud local"`
	//  Timeout  int    `json:"timeout,omitempty"`
}

func (c *ConnectorConfig) UnmarshalJSON(data []byte) error {
	type ConnConfig ConnectorConfig

	tmp := struct {
		*ConnConfig
		URL string `json:"url"`
	}{
		ConnConfig: (*ConnConfig)(c),
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	u, err := url.Parse(tmp.URL)
	if err != nil {
		return err
	}

	tmp.ConnConfig.URL = u

	return nil
}

func (c *ConnectorConfig) TaskManager() (backend.TaskManager, error) {
	switch c.URL.Scheme {
	case "nextcloud":
		return backend.NewNextcloudBackend(c.URL)
	case "file":
		return backend.NewFileBackend(c.URL)
	default:
		return nil, &UnsupportedSchemeError{
			Scheme: c.URL.Scheme,
		}
	}
}



