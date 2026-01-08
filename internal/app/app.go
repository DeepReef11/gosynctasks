package app

import (
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/internal/cache"
	"gosynctasks/internal/config"
	"gosynctasks/internal/operations"
	"log"
	"time"

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
	// syncCoordinator disabled - needs redesign for multi-remote architecture
	// syncCoordinator *sync.SyncCoordinator
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

	// Prepare sync configuration
	syncEnabled := cfg.Sync != nil && cfg.Sync.Enabled
	syncLocalBackend := ""
	cachePath := ""
	if cfg.Sync != nil {
		syncLocalBackend = cfg.Sync.LocalBackend
		if syncEnabled {
			cachePath, _ = cfg.GetCacheDatabasePath()
		}
	}

	// Select backend based on priority
	// When sync is enabled, the local backend is automatically selected
	selectedBackend, taskManager, err := selector.Select(
		explicitBackend,
		cfg.AutoDetectBackend,
		cfg.DefaultBackend,
		cfg.BackendPriority,
		syncEnabled,
		syncLocalBackend,
		cachePath,
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

// GetRemoteBackend returns the remote backend when sync is enabled, otherwise returns the main task manager
// This is used for list operations which should be performed on the remote, not the cache
func (a *App) GetRemoteBackend() backend.TaskManager {
	// If sync is not enabled, just return the main task manager
	if a.config.Sync == nil || !a.config.Sync.Enabled {
		return a.taskManager
	}

	// Determine which remote backend to use
	var remoteBackendName string
	if a.config.DefaultBackend != "" {
		remoteBackendName = a.config.DefaultBackend
	} else if len(a.config.BackendPriority) > 0 {
		remoteBackendName = a.config.BackendPriority[0]
	}

	// Try to get the remote backend from registry
	if remoteBackendName != "" {
		if remoteBackend, err := a.registry.GetBackend(remoteBackendName); err == nil {
			return remoteBackend
		}
	}

	// Fallback to the main task manager
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

// RefreshTaskListsFromRemote refreshes the task list cache from the remote backend (when sync is enabled)
// This is used after list operations to ensure the cache is up-to-date with the remote
func (a *App) RefreshTaskListsFromRemote() error {
	// Get the remote backend
	remoteBackend := a.GetRemoteBackend()

	// Fetch lists from remote
	lists, err := cache.RefreshAndCacheTaskLists(remoteBackend)
	if err != nil {
		return err
	}
	a.taskLists = lists
	return nil
}

// RefreshTaskListsFromRemoteOrWarn is a convenience wrapper that prints a warning on error
func (a *App) RefreshTaskListsFromRemoteOrWarn() {
	if err := a.RefreshTaskListsFromRemote(); err != nil {
		fmt.Printf("Warning: failed to refresh cache from remote: %v\n", err)
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

	return operations.ExecuteAction(a.taskManager, a.config, a.taskLists, cmd, args, a)
}

// initializeSyncCoordinator is currently disabled - needs redesign for multi-remote architecture
// TODO: Implement multi-remote sync coordinator
func (a *App) initializeSyncCoordinator() error {
	return fmt.Errorf("sync coordinator not yet implemented for multi-remote architecture")
}

// GetSyncCoordinator is disabled - needs redesign for multi-remote architecture
// Returns interface{} to avoid circular dependencies
func (a *App) GetSyncCoordinator() interface{} {
	return nil // Disabled for now
}

// Shutdown gracefully shuts down the application
func (a *App) Shutdown() {
	a.ShutdownWithTimeout(5 * time.Second)
}

// ShutdownWithTimeout gracefully shuts down with a custom timeout
func (a *App) ShutdownWithTimeout(timeout time.Duration) {
	// Sync coordinator disabled for now
	// if a.syncCoordinator != nil {
	// 	a.syncCoordinator.Shutdown(timeout)
	// }
}
