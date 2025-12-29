package sync

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"gosynctasks/backend"
	"gosynctasks/internal/config"
)

// SyncCoordinator orchestrates automatic background synchronization
// between local SQLite and remote backends
type SyncCoordinator struct {
	config      *config.Config
	syncManager *backend.SyncManager
	local       *backend.SQLiteBackend

	// Goroutine management
	wg sync.WaitGroup

	// Sync state tracking (prevent duplicate syncs)
	pushSyncing atomic.Bool
	pullSyncing map[string]*atomic.Bool // per-list tracking
	mu          sync.RWMutex            // protect pullSyncing map

	// Logging (silent errors)
	logger *log.Logger

	// Shutdown flag
	shutdown atomic.Bool
}

// NewSyncCoordinator creates a new sync coordinator
func NewSyncCoordinator(cfg *config.Config, local *backend.SQLiteBackend, remote backend.TaskManager) (*SyncCoordinator, error) {
	if cfg == nil || local == nil || remote == nil {
		return nil, fmt.Errorf("config, local backend, and remote backend are required")
	}

	if !cfg.Sync.Enabled {
		return nil, fmt.Errorf("sync is not enabled in configuration")
	}

	// Create sync manager
	// Convert conflict resolution string to strategy type
	strategy := backend.ConflictResolutionStrategy(cfg.Sync.ConflictResolution)
	syncManager := backend.NewSyncManager(local, remote, strategy)

	// Create logger for silent error logging
	logger := log.New(os.Stderr, "[AutoSync] ", log.LstdFlags)

	sc := &SyncCoordinator{
		config:      cfg,
		syncManager: syncManager,
		local:       local,
		pullSyncing: make(map[string]*atomic.Bool),
		logger:      logger,
	}

	return sc, nil
}

// TriggerPushSync triggers a background push sync (for writes: add/update/delete)
// This is non-blocking and returns immediately
func (sc *SyncCoordinator) TriggerPushSync() {
	// Check if shutting down
	if sc.shutdown.Load() {
		return
	}

	// Check if already syncing
	if !sc.pushSyncing.CompareAndSwap(false, true) {
		// Already syncing, skip
		return
	}

	// Launch background goroutine immediately (don't block checking online status)
	sc.wg.Add(1)
	go sc.doPushSync()
}

// doPushSync performs the actual push synchronization
func (sc *SyncCoordinator) doPushSync() {
	defer sc.wg.Done()
	defer sc.pushSyncing.Store(false)

	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			sc.logger.Printf("Panic in push sync: %v", r)
		}
	}()

	// Check if online (happens in background, doesn't block caller)
	if !sc.isOnline() {
		sc.logger.Printf("Skipping push sync: offline")
		return
	}

	// Execute push (only pending operations from queue)
	result, err := sc.syncManager.PushOnly()
	if err != nil {
		sc.logger.Printf("Push sync error: %v", err)
		return
	}

	if result.PushedTasks > 0 {
		sc.logger.Printf("Background push completed: %d tasks synced", result.PushedTasks)
	}
}

// TriggerPullSync triggers a background pull sync (for reads: get)
// This is non-blocking and returns immediately
// If listID is empty, syncs all lists
func (sc *SyncCoordinator) TriggerPullSync(listID string) {
	// Check if shutting down
	if sc.shutdown.Load() {
		return
	}

	// Get or create atomic bool for this list
	sc.mu.Lock()
	pullFlag, exists := sc.pullSyncing[listID]
	if !exists {
		pullFlag = &atomic.Bool{}
		sc.pullSyncing[listID] = pullFlag
	}
	sc.mu.Unlock()

	// Check if already syncing this list
	if !pullFlag.CompareAndSwap(false, true) {
		// Already syncing this list, skip
		return
	}

	// Launch background goroutine immediately (don't block checking online status)
	sc.wg.Add(1)
	go sc.doPullSync(listID, pullFlag)
}

// doPullSync performs the actual pull synchronization
func (sc *SyncCoordinator) doPullSync(listID string, pullFlag *atomic.Bool) {
	defer sc.wg.Done()
	defer pullFlag.Store(false)

	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			sc.logger.Printf("Panic in pull sync: %v", r)
		}
	}()

	// Check if online (happens in background, doesn't block caller)
	if !sc.isOnline() {
		sc.logger.Printf("Skipping pull sync: offline")
		return
	}

	// Execute full sync (pull + push)
	result, err := sc.syncManager.Sync()
	if err != nil {
		sc.logger.Printf("Pull sync error: %v", err)
		return
	}

	if result.PulledTasks > 0 || result.PushedTasks > 0 {
		sc.logger.Printf("Background sync completed: %d pulled, %d pushed",
			result.PulledTasks, result.PushedTasks)
	}
}

// IsStale checks if the data for a given list is stale based on sync_interval
// Returns true if data should be refreshed
func (sc *SyncCoordinator) IsStale(listID string) (bool, error) {
	// Safety check: ensure config and sync section exist
	if sc.config == nil || sc.config.Sync == nil {
		return false, fmt.Errorf("sync coordinator has invalid configuration")
	}

	// If sync interval is 0, never stale (always use cached data)
	if sc.config.Sync.SyncInterval == 0 {
		return false, nil
	}

	// Get database connection
	db, err := sc.local.GetDB()
	if err != nil {
		return true, fmt.Errorf("failed to get database: %w", err)
	}

	// Query last sync time from database
	var lastSyncUnix sql.NullInt64
	query := `SELECT last_full_sync FROM list_sync_metadata WHERE list_id = ?`

	err = db.QueryRow(query, listID).Scan(&lastSyncUnix)
	if err == sql.ErrNoRows {
		// Never synced, definitely stale
		return true, nil
	}
	if err != nil {
		return true, fmt.Errorf("failed to check staleness: %w", err)
	}

	// If no sync timestamp, consider stale
	if !lastSyncUnix.Valid {
		return true, nil
	}

	// Calculate staleness threshold
	lastSync := time.Unix(lastSyncUnix.Int64, 0)
	staleThreshold := time.Duration(sc.config.Sync.SyncInterval) * time.Minute
	timeSinceSync := time.Since(lastSync)

	return timeSinceSync > staleThreshold, nil
}

// isOnline checks if the remote backend is reachable
// Uses a short timeout (3 seconds) to avoid blocking
func (sc *SyncCoordinator) isOnline() bool {
	// Create a timeout context
	done := make(chan bool, 1)

	go func() {
		// Try to get task lists (lightweight operation)
		_, err := sc.syncManager.GetRemote().GetTaskLists()
		done <- (err == nil)
	}()

	// Wait for result or timeout
	select {
	case online := <-done:
		return online
	case <-time.After(3 * time.Second):
		return false
	}
}

// Shutdown gracefully shuts down the coordinator, waiting for pending syncs
func (sc *SyncCoordinator) Shutdown(timeout time.Duration) {
	sc.shutdown.Store(true)

	// Wait for pending syncs with timeout
	done := make(chan struct{})
	go func() {
		sc.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Syncs completed successfully
	case <-time.After(timeout):
		sc.logger.Printf("Warning: Pending syncs did not complete within %v", timeout)
	}
}

// GetSyncManager returns the underlying sync manager for direct access
func (sc *SyncCoordinator) GetSyncManager() *backend.SyncManager {
	return sc.syncManager
}
