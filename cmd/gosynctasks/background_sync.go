package main

import (
	"os"
	"time"

	"gosynctasks/backend"
	"gosynctasks/backend/sync"
	"gosynctasks/internal/config"
	"gosynctasks/internal/utils"

	"github.com/spf13/cobra"
)

// newBackgroundSyncCmd creates a hidden command that runs sync in background from "_internal_background_sync"
// This is spawned as a separate process to allow the main CLI to exit immediately
func newBackgroundSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "_internal_background_sync",
		Hidden: true, // Don't show in help
		Short:  "Internal command for background sync (do not call directly)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set up background logger (PID-specific log file)
			bgLogger, err := utils.NewBackgroundLogger()
			if err != nil {
				// If logger fails to initialize, we can still continue but logging is disabled
				// The error is already handled in NewBackgroundLogger
			}
			defer bgLogger.Close()

			bgLogger.Printf("Started at %s (PID: %d)", time.Now().Format(time.RFC3339), os.Getpid())
			if bgLogger.IsEnabled() {
				bgLogger.Printf("Log file: %s", bgLogger.GetLogPath())
			}

			// Load config
			cfg := config.GetConfig()

			// Check if sync is enabled
			if !cfg.Sync.Enabled || !cfg.Sync.AutoSync {
				bgLogger.Printf("Sync or AutoSync not enabled")
				return nil // Nothing to do
			}

			// Give it a moment to ensure parent process has exited
			time.Sleep(100 * time.Millisecond) //TODO: remove this? Why wait for parent to exit?

			// Get all sync pairs (all remote backends that should be cached)
			syncPairs := cfg.GetSyncPairs()
			bgLogger.Printf("Found %d sync pairs", len(syncPairs))
			if len(syncPairs) == 0 {
				return nil // No backends to sync
			}

			// Sync all backends with pending operations
			// This ensures all backends get synced regardless of which one was just modified
			for _, pair := range syncPairs {
				bgLogger.Printf("Checking backend: %s", pair.RemoteBackendName)

				// Get cache and remote backends for this sync pair
				cacheBackend, remoteBackend, err := getSyncBackends(cfg, pair.RemoteBackendName)
				if err != nil {
					bgLogger.Printf("Failed to get sync backends for %s: %v", pair.RemoteBackendName, err)
					continue // Try next backend
				}

				// Check if there are pending operations for this backend
				ops, err := cacheBackend.GetPendingSyncOperations()
				if err != nil {
					bgLogger.Printf("Error getting pending ops for %s: %v", pair.RemoteBackendName, err)
					continue
				}
				bgLogger.Printf("Backend %s has %d pending operations", pair.RemoteBackendName, len(ops))
				if len(ops) == 0 {
					continue // No pending operations, skip this backend
				}

				// Create sync manager
				strategy := sync.ConflictResolutionStrategy(cfg.Sync.ConflictResolution)
				syncManager := sync.NewSyncManager(cacheBackend, remoteBackend, strategy)

				// Execute sync with timeout
				done := make(chan struct{})
				go func() {
					// Process pending operations
					result, err := syncManager.PushOnly()
					if err != nil {
						bgLogger.Printf("Push error for %s: %v", pair.RemoteBackendName, err)
					} else {
						bgLogger.Printf("Successfully synced %s: %d tasks pushed", pair.RemoteBackendName, result.PushedTasks)
					}
					close(done)
				}()

				// Wait up to 5 seconds for sync to complete (reduced to prevent blocking other backends)
				select {
				case <-done:
					// Success - continue to next backend
					bgLogger.Printf("Completed sync for %s", pair.RemoteBackendName)
				case <-time.After(10 * time.Second): //TODO: add config for sync timeout
					// Timeout - operations remain queued, will retry next time
					bgLogger.Printf("Timeout syncing %s - skipping to next backend", pair.RemoteBackendName)
				}
			}

			bgLogger.Printf("Finished at %s", time.Now().Format(time.RFC3339))
			return nil
		},
	}

	return cmd
}

// RunBackgroundSync executes the background sync logic
// This is exported so it can be called from operations in test mode
func RunBackgroundSync() error {
	// Set up background logger (PID-specific log file)
	bgLogger, err := utils.NewBackgroundLogger()
	if err != nil {
		// If logger fails to initialize, we can still continue but logging is disabled
	}
	defer bgLogger.Close()

	bgLogger.Printf("Started at %s (PID: %d)", time.Now().Format(time.RFC3339), os.Getpid())
	if bgLogger.IsEnabled() {
		bgLogger.Printf("Log file: %s", bgLogger.GetLogPath())
	}

	// Load config
	cfg := config.GetConfig()

	// Check if sync is enabled
	if !cfg.Sync.Enabled || !cfg.Sync.AutoSync {
		bgLogger.Printf("Sync or AutoSync not enabled")
		return nil
	}

	// Get all sync pairs
	syncPairs := cfg.GetSyncPairs()
	bgLogger.Printf("Found %d sync pairs", len(syncPairs))
	if len(syncPairs) == 0 {
		return nil
	}

	// Sync all backends with pending operations
	for _, pair := range syncPairs {
		bgLogger.Printf("Checking backend: %s", pair.RemoteBackendName)

		// Get cache and remote backends for this sync pair
		cacheBackend, remoteBackend, err := getSyncBackends(cfg, pair.RemoteBackendName)
		if err != nil {
			bgLogger.Printf("Failed to get sync backends for %s: %v", pair.RemoteBackendName, err)
			continue
		}

		// Check if there are pending operations
		ops, err := cacheBackend.GetPendingSyncOperations()
		if err != nil {
			bgLogger.Printf("Error getting pending ops for %s: %v", pair.RemoteBackendName, err)
			continue
		}
		bgLogger.Printf("Backend %s has %d pending operations", pair.RemoteBackendName, len(ops))
		if len(ops) == 0 {
			continue
		}

		// Create sync manager
		strategy := sync.ConflictResolutionStrategy(cfg.Sync.ConflictResolution)
		syncManager := sync.NewSyncManager(cacheBackend, remoteBackend, strategy)

		// Execute sync with timeout
		done := make(chan struct{})
		go func() {
			result, err := syncManager.PushOnly()
			if err != nil {
				bgLogger.Printf("Push error for %s: %v", pair.RemoteBackendName, err)
			} else {
				bgLogger.Printf("Successfully synced %s: %d tasks pushed", pair.RemoteBackendName, result.PushedTasks)
			}
			close(done)
		}()

		// Wait for sync to complete
		select {
		case <-done:
			bgLogger.Printf("Completed sync for %s", pair.RemoteBackendName)
		case <-time.After(10 * time.Second):
			bgLogger.Printf("Timeout syncing %s - skipping to next backend", pair.RemoteBackendName)
		}
	}

	bgLogger.Printf("Finished at %s", time.Now().Format(time.RFC3339))
	return nil
}

// getBackend is a helper to get a backend by name from config
func getBackend(cfg *config.Config, name string) (backend.TaskManager, error) {
	// Create registry
	registry, err := backend.NewBackendRegistry(cfg.GetEnabledBackends())
	if err != nil {
		return nil, err
	}

	// Get backend
	return registry.GetBackend(name)
}
