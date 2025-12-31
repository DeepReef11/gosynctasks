package main

import (
	"log"
	"time"

	"gosynctasks/backend"
	"gosynctasks/backend/sqlite"
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
			// Load config
			cfg := config.GetConfig()

			// Check if sync is enabled
			if !cfg.Sync.Enabled || !cfg.Sync.AutoSync {
				return nil // Nothing to do
			}

			// Get backends
			localBackend, err := getBackend(cfg, cfg.Sync.LocalBackend)
			if err != nil {
				log.Printf("[BackgroundSync] Failed to get local backend: %v", err)
				return nil // Silent fail
			}

			remoteBackend, err := getBackend(cfg, cfg.Sync.RemoteBackend)
			if err != nil {
				log.Printf("[BackgroundSync] Failed to get remote backend: %v", err)
				return nil // Silent fail
			}

			// Ensure local is SQLite
			sqliteBackend, ok := localBackend.(*sqlite.SQLiteBackend)
			if !ok {
				return nil // Not SQLite, can't sync
			}

			// Create sync manager
			strategy := sync.ConflictResolutionStrategy(cfg.Sync.ConflictResolution)
			syncManager := sync.NewSyncManager(sqliteBackend, remoteBackend, strategy)

			// Give it a moment to ensure parent process has exited
			time.Sleep(100 * time.Millisecond)

			// Execute sync with timeout
			done := make(chan struct{})
			go func() {
				// Process pending operations
				_, err := syncManager.PushOnly()
				if err != nil {
					log.Printf("[BackgroundSync] Push error: %v", err)
				}
				close(done)
			}()

			// Wait up to 10 seconds for sync to complete
			select {
			case <-done:
				// Success - exit silently
			case <-time.After(10 * time.Second):
				// Timeout - operations remain queued
			}

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
