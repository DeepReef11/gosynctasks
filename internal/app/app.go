package app

import (
	"gosynctasks/backend"
	"gosynctasks/internal/cache"
	"gosynctasks/internal/config"
	"gosynctasks/internal/operations"
	"log"

	"github.com/spf13/cobra"
)

// App holds the application state
type App struct {
	taskLists   []backend.TaskList
	taskManager backend.TaskManager
	config      *config.Config
}

// NewApp creates and initializes a new App instance
func NewApp() (*App, error) {
	cfg := config.GetConfig()
	taskManager, err := cfg.Connector.TaskManager()
	if err != nil {
		return nil, err
	}

	app := &App{
		config:      cfg,
		taskManager: taskManager,
	}

	// Load task lists with cache fallback
	app.taskLists, err = cache.LoadTaskListsWithFallback(taskManager)
	if err != nil {
		log.Printf("Warning: Could not load task lists: %v", err)
	}

	return app, nil
}

// GetTaskLists returns the cached task lists
func (a *App) GetTaskLists() []backend.TaskList {
	return a.taskLists
}

// GetTaskManager returns the task manager
func (a *App) GetTaskManager() backend.TaskManager {
	return a.taskManager
}

// Run is a thin wrapper that delegates to operations
func (a *App) Run(cmd *cobra.Command, args []string) error {
	// Refresh task lists from remote for actual operations
	lists, err := cache.RefreshAndCacheTaskLists(a.taskManager)
	if err != nil {
		// Check if it's a backend error that should be surfaced to the user
		if backendErr, ok := err.(*backend.BackendError); ok {
			// Authentication or connection errors should stop execution
			if backendErr.IsUnauthorized() {
				return backendErr
			}
			// Other HTTP errors should also stop execution
			if backendErr.StatusCode >= 400 {
				return backendErr
			}
		}
		// For other errors, log warning but try to continue
		log.Printf("Warning: Could not refresh task lists: %v", err)
	} else {
		a.taskLists = lists
	}

	return operations.ExecuteAction(a.taskManager, a.config, a.taskLists, cmd, args)
}
