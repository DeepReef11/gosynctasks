package app

import (
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/internal/cache"
	"gosynctasks/internal/config"
	"gosynctasks/internal/operations"
	"log"

	"github.com/spf13/cobra"
)

// App holds the application state
type App struct {
	taskLists       []backend.TaskList
	taskManager     backend.TaskManager
	config          *config.Config
	registry        *backend.BackendRegistry
	selector        *backend.BackendSelector
	selectedBackend string
}

// NewApp creates and initializes a new App instance
// explicitBackend can be empty (will use default/auto-detection)
func NewApp(explicitBackend string) (*App, error) {
	cfg := config.GetConfig()

	// Create backend registry
	registry, err := backend.NewBackendRegistry(cfg.GetEnabledBackends())
	if err != nil {
		return nil, fmt.Errorf("failed to create backend registry: %w", err)
	}

	// Create backend selector
	selector := backend.NewBackendSelector(registry)

	// Select backend based on priority
	selectedBackend, taskManager, err := selector.Select(
		explicitBackend,
		cfg.AutoDetectBackend,
		cfg.DefaultBackend,
		cfg.BackendPriority,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to select backend: %w", err)
	}

	app := &App{
		config:          cfg,
		taskManager:     taskManager,
		registry:        registry,
		selector:        selector,
		selectedBackend: selectedBackend,
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

// RefreshTaskLists refreshes the task list cache from the backend
func (a *App) RefreshTaskLists() error {
	lists, err := cache.RefreshAndCacheTaskLists(a.taskManager)
	if err != nil {
		return err
	}
	a.taskLists = lists
	return nil
}

// RefreshTaskListsOrWarn refreshes the task list cache, printing a warning on error
// This is a convenience wrapper for non-critical cache refresh operations
func (a *App) RefreshTaskListsOrWarn() {
	if err := a.RefreshTaskLists(); err != nil {
		fmt.Printf("Warning: failed to refresh cache: %v\n", err)
	}
}

// ListBackends displays all configured backends and their status
func (a *App) ListBackends() error {
	fmt.Println("\n=== Configured Backends ===")

	infos := a.registry.ListBackends()
	if len(infos) == 0 {
		fmt.Println("No backends configured")
		return nil
	}

	for _, info := range infos {
		fmt.Println(info.String())
		if info.Name == a.selectedBackend {
			fmt.Println("  ✓ Currently selected")
		}
	}

	fmt.Println()
	return nil
}

// DetectBackends displays all auto-detected backends
func (a *App) DetectBackends() error {
	fmt.Println("\n=== Auto-Detected Backends ===")

	detected := a.selector.DetectAll()
	if len(detected) == 0 {
		fmt.Println("No backends detected in current environment")
		return nil
	}

	for _, info := range detected {
		fmt.Printf("%s | %s\n", info.Name, info.Type)
		if info.DetectionInfo != "" {
			fmt.Printf("  %s\n", info.DetectionInfo)
		}
	}

	fmt.Println()
	return nil
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
