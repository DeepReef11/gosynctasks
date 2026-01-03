package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"gosynctasks/backend"
	"gosynctasks/backend/sync"
	"gosynctasks/internal/config"

	"github.com/spf13/cobra"
)

// newBackgroundSyncCmd creates a hidden command that runs sync in background
// This is spawned as a separate process to allow the main CLI to exit immediately
func newBackgroundSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "_internal_background_sync",
		Hidden: true, // Don't show in help
		Short:  "Internal command for background sync (do not call directly)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set up debug logging to file
			logPath := filepath.Join(os.TempDir(), "gosynctasks-background-sync.log")
			logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err == nil {
				defer logFile.Close()
				log.SetOutput(logFile)
				log.Printf("[BackgroundSync] Started at %s", time.Now().Format(time.RFC3339))
			}

			// Load config
			cfg := config.GetConfig()

			// Check if sync is enabled
			if !cfg.Sync.Enabled || !cfg.Sync.AutoSync {
				log.Printf("[BackgroundSync] Sync or AutoSync not enabled")
				return nil // Nothing to do
			}

			// Give it a moment to ensure parent process has exited
			time.Sleep(100 * time.Millisecond)

			// Get all sync pairs (all remote backends that should be cached)
			syncPairs := cfg.GetSyncPairs()
			log.Printf("[BackgroundSync] Found %d sync pairs", len(syncPairs))
			if len(syncPairs) == 0 {
				return nil // No backends to sync
			}

			// Sync all backends with pending operations
			// This ensures all backends get synced regardless of which one was just modified
			for _, pair := range syncPairs {
				log.Printf("[BackgroundSync] Checking backend: %s", pair.RemoteBackendName)

				// Get cache and remote backends for this sync pair
				cacheBackend, remoteBackend, err := getSyncBackends(cfg, pair.RemoteBackendName)
				if err != nil {
					log.Printf("[BackgroundSync] Failed to get sync backends for %s: %v", pair.RemoteBackendName, err)
					continue // Try next backend
				}

				// Check if there are pending operations for this backend
				ops, err := cacheBackend.GetPendingSyncOperations()
				if err != nil {
					log.Printf("[BackgroundSync] Error getting pending ops for %s: %v", pair.RemoteBackendName, err)
					continue
				}
				log.Printf("[BackgroundSync] Backend %s has %d pending operations", pair.RemoteBackendName, len(ops))
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
						log.Printf("[BackgroundSync] Push error for %s: %v", pair.RemoteBackendName, err)
					} else {
						log.Printf("[BackgroundSync] Successfully synced %s: %d tasks pushed", pair.RemoteBackendName, result.PushedTasks)
					}
					close(done)
				}()

				// Wait up to 5 seconds for sync to complete (reduced to prevent blocking other backends)
				select {
				case <-done:
					// Success - continue to next backend
					log.Printf("[BackgroundSync] Completed sync for %s", pair.RemoteBackendName)
				case <-time.After(5 * time.Second):
					// Timeout - operations remain queued, will retry next time
					log.Printf("[BackgroundSync] Timeout syncing %s - skipping to next backend", pair.RemoteBackendName)
				}
			}

			log.Printf("[BackgroundSync] Finished at %s", time.Now().Format(time.RFC3339))
			return nil
		},
	}

	return cmd
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
