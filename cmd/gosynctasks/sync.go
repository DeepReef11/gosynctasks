package main

import (
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/internal/config"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/cobra"
)

// newSyncCmd creates the sync command with all subcommands
func newSyncCmd() *cobra.Command {
	var fullSync bool
	var dryRun bool
	var listName string

	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize tasks with remote backend",
		Long: `Synchronize local SQLite cache with remote backend (e.g., Nextcloud).

The sync command performs bidirectional synchronization:
- Pull: Download remote changes to local cache
- Push: Upload local changes to remote backend

Conflict resolution is handled according to the configured strategy
(server_wins, local_wins, merge, or keep_both).

Examples:
  gosynctasks sync                  # Perform sync
  gosynctasks sync --full          # Force full re-sync (ignore CTags)
  gosynctasks sync --dry-run       # Preview changes without applying
  gosynctasks sync -l "Work"       # Sync specific list only

  gosynctasks sync status          # Show sync status
  gosynctasks sync queue           # Show pending operations
  gosynctasks sync queue clear     # Clear failed operations`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get sync configuration
			cfg := config.GetConfig()
			if cfg.Sync == nil || !cfg.Sync.Enabled {
				return fmt.Errorf("sync is not enabled in configuration")
			}

			// Get backends for sync
			localBackend, remoteBackend, err := getSyncBackends(cfg)
			if err != nil {
				return err
			}

			// Check if offline
			isOffline, offlineReason := isBackendOffline(remoteBackend)
			if isOffline {
				fmt.Printf("⚠ Offline mode: %s\n", offlineReason)
				fmt.Println("Working with local cache. Changes will be synced when online.")
				return nil
			}

			// Create sync manager
			strategy := backend.ConflictResolutionStrategy(cfg.Sync.ConflictResolution)
			if strategy == "" {
				strategy = backend.ServerWins // Default
			}

			sm := backend.NewSyncManager(localBackend, remoteBackend, strategy)

			if dryRun {
				fmt.Println("Dry run mode - no changes will be made")
				// TODO: Implement dry run preview
				return nil
			}

			// Perform sync
			fmt.Println("Syncing...")
			var result *backend.SyncResult
			if fullSync {
				result, err = sm.FullSync()
			} else {
				result, err = sm.Sync()
			}

			if err != nil {
				return fmt.Errorf("sync failed: %w", err)
			}

			// Display results
			printSyncResult(result)
			return nil
		},
	}

	syncCmd.Flags().BoolVar(&fullSync, "full", false, "Force full re-sync (ignore CTags)")
	syncCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without applying")
	syncCmd.Flags().StringVarP(&listName, "list", "l", "", "Sync specific list only")

	// Add subcommands
	syncCmd.AddCommand(newSyncStatusCmd())
	syncCmd.AddCommand(newSyncQueueCmd())

	return syncCmd
}

// newSyncStatusCmd creates the 'sync status' command
func newSyncStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show sync status",
		Long: `Display current synchronization status including:
- Last sync time
- Number of tasks synced
- Pending operations
- Locally modified tasks
- Offline/online status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.GetConfig()
			if cfg.Sync == nil || !cfg.Sync.Enabled {
				fmt.Println("Sync is not enabled in configuration")
				return nil
			}

			// Get backends
			localBackend, remoteBackend, err := getSyncBackends(cfg)
			if err != nil {
				return err
			}

			// Check connection status
			isOffline, offlineReason := isBackendOffline(remoteBackend)

			// Get sync stats
			sm := backend.NewSyncManager(localBackend, remoteBackend, backend.ServerWins)
			stats, err := sm.GetSyncStats()
			if err != nil {
				return fmt.Errorf("failed to get sync stats: %w", err)
			}

			// Display status
			fmt.Println("\n=== Sync Status ===")
			if isOffline {
				fmt.Printf("Connection: Offline (%s)\n", offlineReason)
			} else {
				fmt.Println("Connection: Online")
			}

			fmt.Printf("Local tasks: %d\n", stats.LocalTasks)
			fmt.Printf("Local lists: %d\n", stats.LocalLists)
			fmt.Printf("Pending operations: %d\n", stats.PendingOperations)
			fmt.Printf("Locally modified: %d\n", stats.LocallyModified)
			fmt.Printf("Strategy: %s\n", cfg.Sync.ConflictResolution)

			// Get last sync time
			lastSync, err := getLastSyncTime(localBackend)
			if err == nil && !lastSync.IsZero() {
				timeSince := time.Since(lastSync)
				fmt.Printf("Last sync: %s ago\n", formatDuration(timeSince))
			} else {
				fmt.Println("Last sync: Never")
			}

			fmt.Println()
			return nil
		},
	}
}

// newSyncQueueCmd creates the 'sync queue' command
func newSyncQueueCmd() *cobra.Command {
	queueCmd := &cobra.Command{
		Use:   "queue",
		Short: "Manage sync queue",
		Long:  `Display and manage pending sync operations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.GetConfig()
			if cfg.Sync == nil || !cfg.Sync.Enabled {
				return fmt.Errorf("sync is not enabled")
			}

			localBackend, _, err := getSyncBackends(cfg)
			if err != nil {
				return err
			}

			// Get pending operations
			ops, err := localBackend.GetPendingSyncOperations()
			if err != nil {
				return fmt.Errorf("failed to get pending operations: %w", err)
			}

			if len(ops) == 0 {
				fmt.Println("No pending operations")
				return nil
			}

			fmt.Printf("\nPending Operations (%d):\n\n", len(ops))
			for _, op := range ops {
				fmt.Printf("  %s: %s (list: %s)\n", op.Operation, op.TaskUID, op.ListID)
				fmt.Printf("    Created: %s\n", op.CreatedAt.Format("2006-01-02 15:04:05"))
				if op.RetryCount > 0 {
					fmt.Printf("    Retries: %d\n", op.RetryCount)
				}
				if op.LastError != "" {
					fmt.Printf("    Error: %s\n", op.LastError)
				}
				fmt.Println()
			}

			return nil
		},
	}

	queueCmd.AddCommand(newSyncQueueClearCmd())
	queueCmd.AddCommand(newSyncQueueRetryCmd())

	return queueCmd
}

// newSyncQueueClearCmd creates the 'sync queue clear' command
func newSyncQueueClearCmd() *cobra.Command {
	var failed bool

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear sync queue",
		Long:  `Clear pending sync operations. Use --failed to clear only failed operations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.GetConfig()
			if cfg.Sync == nil || !cfg.Sync.Enabled {
				return fmt.Errorf("sync is not enabled")
			}

			localBackend, _, err := getSyncBackends(cfg)
			if err != nil {
				return err
			}

			// Get pending operations
			ops, err := localBackend.GetPendingSyncOperations()
			if err != nil {
				return fmt.Errorf("failed to get pending operations: %w", err)
			}

			cleared := 0
			for _, op := range ops {
				if failed && op.RetryCount == 0 {
					continue // Skip non-failed operations
				}

				err := localBackend.RemoveSyncOperation(op.TaskUID, op.Operation)
				if err != nil {
					fmt.Printf("Warning: failed to remove operation for %s: %v\n", op.TaskUID, err)
					continue
				}
				cleared++
			}

			fmt.Printf("Cleared %d operations\n", cleared)
			return nil
		},
	}

	cmd.Flags().BoolVar(&failed, "failed", false, "Clear only failed operations")
	return cmd
}

// newSyncQueueRetryCmd creates the 'sync queue retry' command
func newSyncQueueRetryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "retry",
		Short: "Retry failed sync operations",
		Long:  `Retry all failed sync operations by resetting their retry count.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.GetConfig()
			if cfg.Sync == nil || !cfg.Sync.Enabled {
				return fmt.Errorf("sync is not enabled")
			}

			localBackend, _, err := getSyncBackends(cfg)
			if err != nil {
				return err
			}

			// Get failed operations
			ops, err := localBackend.GetPendingSyncOperations()
			if err != nil {
				return fmt.Errorf("failed to get pending operations: %w", err)
			}

			retried := 0
			for _, op := range ops {
				if op.RetryCount == 0 {
					continue // Skip non-failed
				}

				// Reset retry count by updating database directly
				db, err := localBackend.getDB()
				if err != nil {
					return err
				}

				_, err = db.Exec("UPDATE sync_queue SET retry_count = 0, last_error = '' WHERE id = ?", op.ID)
				if err != nil {
					fmt.Printf("Warning: failed to reset retry for operation %d: %v\n", op.ID, err)
					continue
				}
				retried++
			}

			fmt.Printf("Reset %d failed operations for retry\n", retried)
			return nil
		},
	}
}

// Helper functions

// getSyncBackends returns the local and remote backends for sync
func getSyncBackends(cfg *config.Config) (*backend.SQLiteBackend, backend.TaskManager, error) {
	if cfg.Sync.LocalBackend == "" {
		return nil, nil, fmt.Errorf("local_backend not configured for sync")
	}
	if cfg.Sync.RemoteBackend == "" {
		return nil, nil, fmt.Errorf("remote_backend not configured for sync")
	}

	// Get local backend
	localCfg, err := cfg.GetBackend(cfg.Sync.LocalBackend)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get local backend config: %w", err)
	}

	if localCfg.Type != "sqlite" {
		return nil, nil, fmt.Errorf("local backend must be SQLite, got %s", localCfg.Type)
	}

	local, err := backend.NewSQLiteBackend(*localCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create local backend: %w", err)
	}

	// Get remote backend
	remoteCfg, err := cfg.GetBackend(cfg.Sync.RemoteBackend)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get remote backend config: %w", err)
	}

	remote, err := remoteCfg.TaskManager()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create remote backend: %w", err)
	}

	return local, remote, nil
}

// isBackendOffline checks if the backend is reachable
func isBackendOffline(taskManager backend.TaskManager) (bool, string) {
	// Try to get task lists with timeout
	done := make(chan error, 1)
	go func() {
		_, err := taskManager.GetTaskLists()
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			return false, ""
		}

		// Analyze error to determine offline reason
		errStr := err.Error()
		if isNetworkError(err) {
			return true, "Network unreachable"
		}
		if isDNSError(err) {
			return true, "DNS resolution failed"
		}
		if isConnectionRefused(err) {
			return true, "Connection refused"
		}
		if isTimeout(err) {
			return true, "Connection timeout"
		}

		// Unknown error - assume online but backend issue
		return false, ""

	case <-time.After(5 * time.Second):
		return true, "Connection timeout"
	}
}

// isNetworkError checks if error is a network error
func isNetworkError(err error) bool {
	if _, ok := err.(*net.OpError); ok {
		return true
	}
	if _, ok := err.(*url.Error); ok {
		return true
	}
	return false
}

// isDNSError checks if error is a DNS resolution error
func isDNSError(err error) bool {
	if dnsErr, ok := err.(*net.DNSError); ok {
		return dnsErr != nil
	}
	return false
}

// isConnectionRefused checks if error is connection refused
func isConnectionRefused(err error) bool {
	if opErr, ok := err.(*net.OpError); ok {
		return opErr.Op == "dial"
	}
	return false
}

// isTimeout checks if error is a timeout
func isTimeout(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	if urlErr, ok := err.(*url.Error); ok {
		return urlErr.Timeout()
	}
	return false
}

// printSyncResult displays sync result in a user-friendly format
func printSyncResult(result *backend.SyncResult) {
	fmt.Println("\n=== Sync Complete ===")
	fmt.Printf("Pulled tasks: %d\n", result.PulledTasks)
	fmt.Printf("Pushed tasks: %d\n", result.PushedTasks)

	if result.ConflictsFound > 0 {
		fmt.Printf("Conflicts found: %d\n", result.ConflictsFound)
		fmt.Printf("Conflicts resolved: %d\n", result.ConflictsResolved)
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\n⚠ Errors: %d\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("  - %v\n", err)
		}
	}

	fmt.Printf("Duration: %s\n", result.Duration.Round(time.Millisecond))
	fmt.Println()
}

// getLastSyncTime retrieves the most recent sync timestamp
func getLastSyncTime(local *backend.SQLiteBackend) (time.Time, error) {
	db, err := local.getDB()
	if err != nil {
		return time.Time{}, err
	}

	var lastSync int64
	err = db.QueryRow("SELECT MAX(last_full_sync) FROM list_sync_metadata").Scan(&lastSync)
	if err != nil {
		return time.Time{}, err
	}

	if lastSync == 0 {
		return time.Time{}, nil
	}

	return time.Unix(lastSync, 0), nil
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

// Auto-sync functionality

// shouldAutoSync checks if auto-sync should run
func shouldAutoSync(cfg *config.Config) bool {
	if cfg.Sync == nil || !cfg.Sync.Enabled || !cfg.Sync.AutoSync {
		return false
	}

	// Check if enough time has passed since last sync
	if cfg.Sync.SyncInterval <= 0 {
		return false
	}

	// TODO: Implement last sync time checking
	return false
}

// performAutoSync runs auto-sync in the background
func performAutoSync(cfg *config.Config) {
	if !shouldAutoSync(cfg) {
		return
	}

	// Run sync in background (non-blocking)
	go func() {
		localBackend, remoteBackend, err := getSyncBackends(cfg)
		if err != nil {
			return // Silently fail for auto-sync
		}

		// Check if online
		isOffline, _ := isBackendOffline(remoteBackend)
		if isOffline {
			return // Skip auto-sync if offline
		}

		strategy := backend.ConflictResolutionStrategy(cfg.Sync.ConflictResolution)
		if strategy == "" {
			strategy = backend.ServerWins
		}

		sm := backend.NewSyncManager(localBackend, remoteBackend, strategy)
		sm.Sync()
	}()
}
