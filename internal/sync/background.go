package sync

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"gosynctasks/backend"
	"gosynctasks/backend/sqlite"
	backendsync "gosynctasks/backend/sync"
	"gosynctasks/internal/config"
	"gosynctasks/internal/utils"
)

// SpawnBackgroundSync spawns a detached background process to handle sync
// This allows the main CLI to exit immediately while sync continues
func SpawnBackgroundSync() error {
	// Get the current executable path
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	// Resolve symlinks
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return err
	}

	// Spawn a detached background process
	// Use _internal_background_sync as a hidden command
	cmd := exec.Command(executable, "_internal_background_sync")

	// Detach from parent process
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Start the process and don't wait for it
	if err := cmd.Start(); err != nil {
		return err
	}

	// Detach - don't wait for the process to complete
	// The parent process can exit immediately
	return nil
}

// RunBackgroundSyncInProcess runs background sync in the current process
// This is used in test mode instead of spawning a separate process
func RunBackgroundSyncInProcess() error {
	// Set up background logger
	bgLogger, err := utils.NewBackgroundLogger()
	if err != nil {
		// Continue without logging
	}
	if bgLogger != nil {
		defer bgLogger.Close()
		bgLogger.Printf("Started in-process sync at %s (PID: %d)", time.Now().Format(time.RFC3339), os.Getpid())
	}

	// Load config
	cfg := config.GetConfig()

	// Check if sync is enabled
	if cfg.Sync == nil || !cfg.Sync.Enabled || !cfg.Sync.AutoSync {
		if bgLogger != nil {
			bgLogger.Printf("Sync or AutoSync not enabled")
		}
		return nil
	}

	// Get all sync pairs
	syncPairs := cfg.GetSyncPairs()
	if bgLogger != nil {
		bgLogger.Printf("Found %d sync pairs", len(syncPairs))
	}
	if len(syncPairs) == 0 {
		return nil
	}

	// Sync all backends with pending operations
	for _, pair := range syncPairs {
		if bgLogger != nil {
			bgLogger.Printf("Checking backend: %s", pair.RemoteBackendName)
		}

		// Get backends
		cacheBackend, remoteBackend, err := getBackendsForSync(cfg, pair.RemoteBackendName)
		if err != nil {
			if bgLogger != nil {
				bgLogger.Printf("Failed to get sync backends for %s: %v", pair.RemoteBackendName, err)
			}
			continue
		}

		// Check for pending operations
		ops, err := cacheBackend.GetPendingSyncOperations()
		if err != nil {
			if bgLogger != nil {
				bgLogger.Printf("Error getting pending ops for %s: %v", pair.RemoteBackendName, err)
			}
			continue
		}
		if bgLogger != nil {
			bgLogger.Printf("Backend %s has %d pending operations", pair.RemoteBackendName, len(ops))
		}
		if len(ops) == 0 {
			continue
		}

		// Create sync manager
		strategy := backendsync.ConflictResolutionStrategy(cfg.Sync.ConflictResolution)
		syncManager := backendsync.NewSyncManager(cacheBackend, remoteBackend, strategy)

		// Execute sync with timeout
		done := make(chan struct{})
		go func() {
			result, err := syncManager.PushOnly()
			if err != nil {
				if bgLogger != nil {
					bgLogger.Printf("Push error for %s: %v", pair.RemoteBackendName, err)
				}
			} else {
				if bgLogger != nil {
					bgLogger.Printf("Successfully synced %s: %d tasks pushed", pair.RemoteBackendName, result.PushedTasks)
				}
			}
			close(done)
		}()

		// Wait for sync
		select {
		case <-done:
			if bgLogger != nil {
				bgLogger.Printf("Completed sync for %s", pair.RemoteBackendName)
			}
		case <-time.After(10 * time.Second):
			if bgLogger != nil {
				bgLogger.Printf("Timeout syncing %s", pair.RemoteBackendName)
			}
		}
	}

	if bgLogger != nil {
		bgLogger.Printf("Finished at %s", time.Now().Format(time.RFC3339))
	}
	return nil
}

// getBackendsForSync gets cache and remote backends for a sync pair
func getBackendsForSync(cfg *config.Config, remoteName string) (*sqlite.SQLiteBackend, backend.TaskManager, error) {
	// Create registry
	registry, err := backend.NewBackendRegistry(cfg.GetEnabledBackends())
	if err != nil {
		return nil, nil, err
	}

	// Get cache backend for this remote
	cacheBackendName := "cache_" + remoteName
	cacheBackend, err := registry.GetBackend(cacheBackendName)
	if err != nil {
		return nil, nil, err
	}

	// Type assert to SQLiteBackend
	sqliteBackend, ok := cacheBackend.(*sqlite.SQLiteBackend)
	if !ok {
		return nil, nil, err
	}

	// Get remote backend
	remoteBackend, err := registry.GetBackend(remoteName)
	if err != nil {
		return nil, nil, err
	}

	return sqliteBackend, remoteBackend, nil
}
