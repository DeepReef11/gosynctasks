package sync

import (
	"gosynctasks/backend"
	"gosynctasks/backend/sqlite"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ConflictResolutionStrategy defines how to handle sync conflicts
type ConflictResolutionStrategy string

const (
	ServerWins ConflictResolutionStrategy = "server_wins" // Discard local changes, use server version
	LocalWins  ConflictResolutionStrategy = "local_wins"  // Overwrite server with local version
	Merge      ConflictResolutionStrategy = "merge"       // Combine non-conflicting fields
	KeepBoth   ConflictResolutionStrategy = "keep_both"   // Create duplicate with suffix
)

// SyncManager coordinates synchronization between local SQLite and remote backend
type SyncManager struct {
	local    *sqlite.SQLiteBackend
	remote   backend.TaskManager
	strategy ConflictResolutionStrategy
}

// NewSyncManager creates a new sync manager
func NewSyncManager(local *sqlite.SQLiteBackend, remote backend.TaskManager, strategy ConflictResolutionStrategy) *SyncManager {
	return &SyncManager{
		local:    local,
		remote:   remote,
		strategy: strategy,
	}
}

// SyncResult contains statistics about the sync operation
type SyncResult struct {
	PulledTasks       int
	PushedTasks       int
	ConflictsFound    int
	ConflictsResolved int
	Errors            []error
	Duration          time.Duration
}

// Sync performs bidirectional synchronization
func (sm *SyncManager) Sync() (*SyncResult, error) {
	startTime := time.Now()
	result := &SyncResult{}

	// Phase 1: Pull remote changes
	pullResult, err := sm.pull()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("pull phase failed: %w", err))
		// Continue to push phase even if pull fails
	} else {
		result.PulledTasks = pullResult.PulledTasks
		result.ConflictsFound = pullResult.ConflictsFound
		result.ConflictsResolved = pullResult.ConflictsResolved
	}

	// Phase 2: Push local changes
	pushResult, err := sm.push()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("push phase failed: %w", err))
	} else {
		result.PushedTasks = pushResult.PushedTasks
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// pullResult contains statistics from the pull phase
type pullResult struct {
	PulledTasks       int
	ConflictsFound    int
	ConflictsResolved int
}

// pull retrieves remote changes and applies them locally
func (sm *SyncManager) pull() (*pullResult, error) {
	result := &pullResult{}

	// Get all remote task lists
	remoteLists, err := sm.remote.GetTaskLists()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote lists: %w", err)
	}

	// Sync each list
	for _, remoteList := range remoteLists {
		// Check if list exists locally
		localLists, err := sm.local.GetTaskLists()
		if err != nil {
			return nil, fmt.Errorf("failed to get local lists: %w", err)
		}

		// Find or create list locally
		listExists := false
		var localCTag string
		for _, localList := range localLists {
			if localList.ID == remoteList.ID {
				listExists = true
				localCTag = localList.CTags
				break
			}
		}

		// Check if list changed (CTag comparison)
		if listExists && localCTag == remoteList.CTags {
			// No changes, skip this list
			continue
		}

		// Create list if it doesn't exist
		if !listExists {
			// Insert list metadata
			db, err := sm.local.GetDB()
			if err != nil {
				return nil, err
			}

			now := time.Now().Unix()
			_, err = db.Exec(`
				INSERT INTO list_sync_metadata (list_id, backend_name, list_name, list_color, last_ctag, last_full_sync, created_at, modified_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, remoteList.ID, sm.getBackendName(), remoteList.Name, remoteList.Color, remoteList.CTags, now, now, now)
			if err != nil {
				return nil, fmt.Errorf("failed to create local list: %w", err)
			}
		} else {
			// Update list CTag
			db, err := sm.local.GetDB()
			if err != nil {
				return nil, err
			}

			_, err = db.Exec(`
				UPDATE list_sync_metadata
				SET last_ctag = ?, last_full_sync = ?
				WHERE backend_name = ? AND list_id = ?
			`, remoteList.CTags, time.Now().Unix(), sm.getBackendName(), remoteList.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to update list CTag: %w", err)
			}
		}

		// Get all remote tasks for this list
		remoteTasks, err := sm.remote.GetTasks(remoteList.ID, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get remote tasks for list %s: %w", remoteList.ID, err)
		}

		// Sort remote tasks so parents come before children (important for foreign key constraints)
		remoteTasks = sortTasksByHierarchy(remoteTasks)

		// Get all local tasks for this list
		localTasks, err := sm.local.GetTasks(remoteList.ID, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get local tasks for list %s: %w", remoteList.ID, err)
		}

		// Create map of local tasks for quick lookup
		localTaskMap := make(map[string]*backend.Task)
		for i := range localTasks {
			localTaskMap[localTasks[i].UID] = &localTasks[i]
		}

		// Process each remote task
		for _, remoteTask := range remoteTasks {
			localTask, exists := localTaskMap[remoteTask.UID]

			if !exists {
				// New remote task - insert locally
				err := sm.insertTaskLocally(remoteList.ID, remoteTask)
				if err != nil {
					return nil, fmt.Errorf("failed to insert task %s: %w", remoteTask.UID, err)
				}
				result.PulledTasks++
			} else {
				// backend.Task exists locally - check for conflict
				isLocallyModified, err := sm.isTaskLocallyModified(remoteTask.UID)
				if err != nil {
					return nil, err
				}

				isRemoteModified, err := sm.isTaskRemoteModified(remoteTask)
				if err != nil {
					return nil, err
				}

				if isLocallyModified && isRemoteModified {
					// Both modified - real conflict
					result.ConflictsFound++
					err := sm.resolveConflict(remoteList.ID, *localTask, remoteTask)
					if err != nil {
						return nil, fmt.Errorf("failed to resolve conflict for task %s: %w", remoteTask.UID, err)
					}
					result.ConflictsResolved++
				} else if isLocallyModified {
					// Only local modified - will be pushed in push phase, don't update local
					// Do nothing here, let push phase handle it
				} else {
					// Remote modified or neither modified - update local with remote
					err := sm.updateTaskLocally(remoteList.ID, remoteTask)
					if err != nil {
						return nil, fmt.Errorf("failed to update task %s: %w", remoteTask.UID, err)
					}
					result.PulledTasks++
				}
			}

			// Remove from map (for deletion detection)
			delete(localTaskMap, remoteTask.UID)
		}

		// Remaining tasks in map were deleted remotely
		for _, deletedTask := range localTaskMap {
			isLocallyModified, err := sm.isTaskLocallyModified(deletedTask.UID)
			if err != nil {
				return nil, err
			}

			if !isLocallyModified {
				// Delete locally
				err := sm.deleteTaskLocally(remoteList.ID, deletedTask.UID)
				if err != nil {
					return nil, fmt.Errorf("failed to delete task %s: %w", deletedTask.UID, err)
				}
			}
			// If locally modified, keep it (will be pushed in push phase)
		}
	}

	return result, nil
}

// pushResult contains statistics from the push phase
type pushResult struct {
	PushedTasks int
}

// push sends local changes to remote backend
func (sm *SyncManager) push() (*pushResult, error) {
	result := &pushResult{}

	// Get pending sync operations
	operations, err := sm.local.GetPendingSyncOperations()
	if err != nil {
		return nil, fmt.Errorf("failed to get pending operations: %w", err)
	}

	// Process each operation
	for _, op := range operations {
		// Skip if too many retries
		if op.RetryCount >= 5 {
			continue
		}

		var pushErr error

		switch op.Operation {
		case "create":
			pushErr = sm.pushCreate(op)
		case "update":
			pushErr = sm.pushUpdate(op)
		case "delete":
			pushErr = sm.pushDelete(op)
		default:
			pushErr = fmt.Errorf("unknown operation: %s", op.Operation)
		}

		if pushErr != nil {
			// Increment retry count
			db, err := sm.local.GetDB()
			if err != nil {
				return nil, err
			}

			_, err = db.Exec(`
				UPDATE sync_queue
				SET retry_count = retry_count + 1, last_error = ?
				WHERE id = ?
			`, pushErr.Error(), op.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to update retry count: %w", err)
			}

			// Apply exponential backoff
			backoffSeconds := 1 << op.RetryCount // 2^retryCount
			if backoffSeconds > 300 {
				backoffSeconds = 300 // Max 5 minutes
			}
			time.Sleep(time.Duration(backoffSeconds) * time.Second)
		} else {
			// Success - pushCreate already handles clearing flags for create operations
			// Only clear for update/delete operations
			if op.Operation != "create" {
				err := sm.local.ClearSyncFlagsAndQueue(op.TaskUID)
				if err != nil {
					return nil, fmt.Errorf("failed to clear sync flags and queue: %w", err)
				}
			}

			result.PushedTasks++
		}
	}

	return result, nil
}

// pushCreate pushes a create operation to remote
func (sm *SyncManager) pushCreate(op sqlite.SyncOperation) error {
	// Get task from local
	tasks, err := sm.local.GetTasks(op.ListID, nil)
	if err != nil {
		return err
	}

	var task *backend.Task
	for i := range tasks {
		if tasks[i].UID == op.TaskUID {
			task = &tasks[i]
			break
		}
	}

	if task == nil {
		// Task was deleted locally, remove from queue
		return nil
	}

	// Add to remote and get the remote-assigned UID
	remoteUID, err := sm.remote.AddTask(op.ListID, *task)
	if err != nil {
		return fmt.Errorf("failed to create task on remote: %w", err)
	}

	// If the remote backend assigned a different UID, update local task
	if remoteUID != task.UID {
		// Update the local task's UID to match the remote
		// This is critical for Todoist and other backends that generate their own IDs
		err = sm.updateLocalTaskUID(op.ListID, task.UID, remoteUID)
		if err != nil {
			return fmt.Errorf("failed to update local task UID: %w", err)
		}

		// Clear sync flags and queue using the NEW UID (after update)
		err = sm.local.ClearSyncFlagsAndQueue(remoteUID)
		if err != nil {
			return fmt.Errorf("failed to clear sync flags and queue: %w", err)
		}
	} else {
		// UID didn't change, clear flags using existing UID
		err = sm.local.ClearSyncFlagsAndQueue(task.UID)
		if err != nil {
			return fmt.Errorf("failed to clear sync flags and queue: %w", err)
		}
	}

	return nil
}

// pushUpdate pushes an update operation to remote
func (sm *SyncManager) pushUpdate(op sqlite.SyncOperation) error {
	// Get task from local
	tasks, err := sm.local.GetTasks(op.ListID, nil)
	if err != nil {
		return err
	}

	var task *backend.Task
	for i := range tasks {
		if tasks[i].UID == op.TaskUID {
			task = &tasks[i]
			break
		}
	}

	if task == nil {
		// backend.Task was deleted locally, remove from queue
		return nil
	}

	// Update on remote
	err = sm.remote.UpdateTask(op.ListID, *task)
	if err != nil {
		return fmt.Errorf("failed to update task on remote: %w", err)
	}

	return nil
}

// pushDelete pushes a delete operation to remote
func (sm *SyncManager) pushDelete(op sqlite.SyncOperation) error {
	err := sm.remote.DeleteTask(op.ListID, op.TaskUID)
	if err != nil {
		// If task doesn't exist on remote, that's ok
		if backendErr, ok := err.(*backend.BackendError); ok && backendErr.IsNotFound() {
			return nil
		}
		return fmt.Errorf("failed to delete task on remote: %w", err)
	}

	return nil
}

// isTaskLocallyModified checks if a task is locally modified
func (sm *SyncManager) isTaskLocallyModified(taskUID string) (bool, error) {
	db, err := sm.local.GetDB()
	if err != nil {
		return false, err
	}

	var locallyModified int
	err = db.QueryRow(`
		SELECT COALESCE(sm.locally_modified, 0)
		FROM sync_metadata sm
		INNER JOIN tasks t ON sm.task_internal_id = t.internal_id
		WHERE t.uid = ? AND t.backend_name = ?
	`, taskUID, sm.getBackendName()).Scan(&locallyModified)
	if err != nil {
		// If no sync metadata, treat as not modified
		return false, nil
	}

	return locallyModified == 1, nil
}

// isTaskRemoteModified checks if a remote task has been modified since last sync
func (sm *SyncManager) isTaskRemoteModified(remoteTask backend.Task) (bool, error) {
	db, err := sm.local.GetDB()
	if err != nil {
		return false, err
	}

	var remoteModifiedAt sql.NullInt64
	err = db.QueryRow(`
		SELECT sm.remote_modified_at
		FROM sync_metadata sm
		INNER JOIN tasks t ON sm.task_internal_id = t.internal_id
		WHERE t.uid = ? AND t.backend_name = ?
	`, remoteTask.UID, sm.getBackendName()).Scan(&remoteModifiedAt)
	if err != nil {
		// If no sync metadata exists, treat as modified (new from our perspective)
		return true, nil
	}

	// If we don't have a remote modified timestamp, treat as modified
	if !remoteModifiedAt.Valid {
		return true, nil
	}

	// Compare remote task's Modified timestamp with stored remote_modified_at
	// Truncate to second precision since we store timestamps as Unix seconds
	lastRemoteModified := time.Unix(remoteModifiedAt.Int64, 0)
	currentRemoteModified := time.Unix(remoteTask.Modified.Unix(), 0)

	// If remote task's Modified is newer than our stored timestamp, it's been modified
	if !remoteTask.Modified.IsZero() && currentRemoteModified.After(lastRemoteModified) {
		return true, nil
	}

	return false, nil
}

// resolveConflict resolves a conflict between local and remote versions
func (sm *SyncManager) resolveConflict(listID string, localTask, remoteTask backend.Task) error {
	switch sm.strategy {
	case ServerWins:
		return sm.resolveServerWins(listID, localTask, remoteTask)
	case LocalWins:
		return sm.resolveLocalWins(listID, localTask, remoteTask)
	case Merge:
		return sm.resolveMerge(listID, localTask, remoteTask)
	case KeepBoth:
		return sm.resolveKeepBoth(listID, localTask, remoteTask)
	default:
		return fmt.Errorf("unknown conflict resolution strategy: %s", sm.strategy)
	}
}

// resolveServerWins discards local changes and uses server version
func (sm *SyncManager) resolveServerWins(listID string, localTask, remoteTask backend.Task) error {
	// Update local with remote version
	err := sm.updateTaskLocally(listID, remoteTask)
	if err != nil {
		return err
	}

	// Clear locally modified flag AND remove pending operations
	// Server wins means we discard local changes and don't push them
	return sm.local.ClearSyncFlagsAndQueue(remoteTask.UID)
}

// resolveLocalWins keeps local changes for push to server
func (sm *SyncManager) resolveLocalWins(listID string, localTask, remoteTask backend.Task) error {
	// Keep local version, mark for push
	// Local task already has locally_modified=1, so it will be pushed
	// Just update sync metadata with remote info
	if !remoteTask.Modified.IsZero() {
		return sm.local.UpdateSyncMetadata(localTask.UID, listID, "", remoteTask.Modified)
	}
	return nil
}

// resolveMerge intelligently merges local and remote changes
func (sm *SyncManager) resolveMerge(listID string, localTask, remoteTask backend.Task) error {
	mergedTask := remoteTask // Start with remote as base

	// Preserve local description if remote hasn't changed
	if localTask.Description != "" && remoteTask.Description == "" {
		mergedTask.Description = localTask.Description
	}

	// Use higher priority
	if localTask.Priority > 0 && localTask.Priority < remoteTask.Priority {
		mergedTask.Priority = localTask.Priority
	}

	// Union categories
	categorySet := make(map[string]bool)
	for _, cat := range remoteTask.Categories {
		categorySet[cat] = true
	}
	for _, cat := range localTask.Categories {
		categorySet[cat] = true
	}
	mergedTask.Categories = make([]string, 0, len(categorySet))
	for cat := range categorySet {
		mergedTask.Categories = append(mergedTask.Categories, cat)
	}

	// Use most recent timestamps
	if localTask.DueDate != nil && (remoteTask.DueDate == nil || localTask.DueDate.After(*remoteTask.DueDate)) {
		mergedTask.DueDate = localTask.DueDate
	}

	// Update locally with merged version
	err := sm.updateTaskLocally(listID, mergedTask)
	if err != nil {
		return err
	}

	// Mark for push to propagate merge
	return sm.local.MarkLocallyModified(mergedTask.UID)
}

// resolveKeepBoth creates a copy of the local version
func (sm *SyncManager) resolveKeepBoth(listID string, localTask, remoteTask backend.Task) error {
	// Update local task with remote version
	err := sm.updateTaskLocally(listID, remoteTask)
	if err != nil {
		return err
	}

	// Create a copy of the local version with new UID
	localCopy := localTask
	localCopy.UID = sqlite.GenerateUID()
	localCopy.Summary = localTask.Summary + " (local copy)"

	// Insert the copy
	_, err = sm.local.AddTask(listID, localCopy)
	if err != nil {
		return err
	}

	// Clear original task's sync flags AND remove pending operations
	// We're accepting the remote version for the original, local copy is separate
	return sm.local.ClearSyncFlagsAndQueue(remoteTask.UID)
}

// insertTaskLocally inserts a remote task into local storage
func (sm *SyncManager) insertTaskLocally(listID string, task backend.Task) error {
	db, err := sm.local.GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Insert task
	result, err := tx.Exec(`
		INSERT INTO tasks (
			uid, backend_name, list_id, summary, description, status, priority,
			created_at, modified_at, due_date, start_date, completed_at,
			parent_uid, categories
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		task.UID,
		sm.getBackendName(),
		listID,
		task.Summary,
		sqlite.NullString(task.Description),
		task.Status,
		task.Priority,
		sqlite.TimeValueToNullInt64(task.Created),
		sqlite.TimeValueToNullInt64(task.Modified),
		sqlite.TimeToNullInt64(task.DueDate),
		sqlite.TimeToNullInt64(task.StartDate),
		sqlite.TimeToNullInt64(task.Completed),
		sqlite.NullString(task.ParentUID),
		sqlite.NullString(strings.Join(task.Categories, ",")),
	)
	if err != nil {
		return err
	}

	// Get the internal_id that was just created
	internalID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	// Insert sync metadata (not locally modified since it came from server)
	now := time.Now().Unix()
	remoteModifiedAt := int64(0)
	if !task.Modified.IsZero() {
		remoteModifiedAt = task.Modified.Unix()
	}

	_, err = tx.Exec(`
		INSERT INTO sync_metadata (
			task_internal_id, backend_name, list_id, last_synced_at, remote_modified_at,
			locally_modified, locally_deleted
		) VALUES (?, ?, ?, ?, ?, 0, 0)
	`, internalID, sm.getBackendName(), listID, now, remoteModifiedAt)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// updateTaskLocally updates a local task with remote data
func (sm *SyncManager) updateTaskLocally(listID string, task backend.Task) error {
	db, err := sm.local.GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Get internal_id for this task
	var internalID int64
	err = tx.QueryRow("SELECT internal_id FROM tasks WHERE backend_name = ? AND uid = ? AND list_id = ?",
		sm.getBackendName(), task.UID, listID).Scan(&internalID)
	if err != nil {
		return err
	}

	// Update task
	_, err = tx.Exec(`
		UPDATE tasks
		SET summary = ?, description = ?, status = ?, priority = ?,
		    modified_at = ?, due_date = ?, start_date = ?, completed_at = ?,
		    parent_uid = ?, categories = ?
		WHERE uid = ? AND backend_name = ? AND list_id = ?
	`,
		task.Summary,
		sqlite.NullString(task.Description),
		task.Status,
		task.Priority,
		sqlite.TimeValueToNullInt64(task.Modified),
		sqlite.TimeToNullInt64(task.DueDate),
		sqlite.TimeToNullInt64(task.StartDate),
		sqlite.TimeToNullInt64(task.Completed),
		sqlite.NullString(task.ParentUID),
		sqlite.NullString(strings.Join(task.Categories, ",")),
		task.UID,
		sm.getBackendName(),
		listID,
	)
	if err != nil {
		return err
	}

	// Update sync metadata
	now := time.Now().Unix()
	remoteModifiedAt := int64(0)
	if !task.Modified.IsZero() {
		remoteModifiedAt = task.Modified.Unix()
	}

	_, err = tx.Exec(`
		UPDATE sync_metadata
		SET last_synced_at = ?, remote_modified_at = ?, locally_modified = 0, locally_deleted = 0
		WHERE task_internal_id = ? AND backend_name = ?
	`, now, remoteModifiedAt, internalID, sm.getBackendName())
	if err != nil {
		return err
	}

	return tx.Commit()
}

// deleteTaskLocally deletes a task from local storage
func (sm *SyncManager) deleteTaskLocally(listID string, taskUID string) error {
	db, err := sm.local.GetDB()
	if err != nil {
		return err
	}

	// Delete task (cascade will delete sync_metadata via internal_id foreign key)
	_, err = db.Exec("DELETE FROM tasks WHERE uid = ? AND backend_name = ? AND list_id = ?", taskUID, sm.getBackendName(), listID)
	return err
}

// FullSync performs a complete synchronization, ignoring CTags
func (sm *SyncManager) FullSync() (*SyncResult, error) {
	// Clear all CTags to force full sync
	db, err := sm.local.GetDB()
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("UPDATE list_sync_metadata SET last_ctag = ''")
	if err != nil {
		return nil, fmt.Errorf("failed to clear CTags: %w", err)
	}

	return sm.Sync()
}

// GetSyncStats returns current sync statistics
func (sm *SyncManager) GetSyncStats() (*SyncStats, error) {
	db, err := sm.local.GetDB()
	if err != nil {
		return nil, err
	}

	stats, err := db.GetStats()
	if err != nil {
		return nil, err
	}

	return &SyncStats{
		LocalTasks:        stats.TaskCount,
		LocalLists:        stats.ListCount,
		PendingOperations: stats.PendingSyncOps,
		LocallyModified:   stats.LocallyModified,
	}, nil
}

// SyncStats contains sync-related statistics
type SyncStats struct {
	LocalTasks        int
	LocalLists        int
	PendingOperations int
	LocallyModified   int
}

// sortTasksByHierarchy sorts tasks so parent tasks come before child tasks.
// This is critical for respecting foreign key constraints during sync.
func sortTasksByHierarchy(tasks []backend.Task) []backend.Task {
	// Build parent-child relationships
	childrenMap := make(map[string][]int) // parentUID -> child indexes
	rootIndexes := []int{}                // tasks with no parent

	for i, task := range tasks {
		if task.ParentUID == "" {
			rootIndexes = append(rootIndexes, i)
		} else {
			childrenMap[task.ParentUID] = append(childrenMap[task.ParentUID], i)
		}
	}

	// Traverse hierarchy depth-first, collecting tasks in order
	sorted := []backend.Task{}
	visited := make(map[int]bool)

	var visit func(index int)
	visit = func(index int) {
		if visited[index] {
			return
		}
		visited[index] = true
		sorted = append(sorted, tasks[index])

		// Visit children
		taskUID := tasks[index].UID
		for _, childIndex := range childrenMap[taskUID] {
			visit(childIndex)
		}
	}

	// Visit all root tasks (and their descendants)
	for _, rootIndex := range rootIndexes {
		visit(rootIndex)
	}

	// Add any orphaned tasks (tasks with parent_uid pointing to non-existent parents)
	for i := range tasks {
		if !visited[i] {
			sorted = append(sorted, tasks[i])
		}
	}

	return sorted
}

// PushOnly executes only the push phase of sync (no pull)
// This is useful for background sync after write operations
func (sm *SyncManager) PushOnly() (*SyncResult, error) {
	startTime := time.Now()
	result := &SyncResult{}

	// Only push local changes
	pushResult, err := sm.push()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("push phase failed: %w", err))
	} else {
		result.PushedTasks = pushResult.PushedTasks
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// updateLocalTaskUID updates a task's UID in the local cache
// This is needed when remote backends (like Todoist) assign their own IDs
func (sm *SyncManager) updateLocalTaskUID(listID string, oldUID string, newUID string) error {
	db, err := sm.local.GetDB()
	if err != nil {
		return err
	}

	// Simply update the UID from "pending-{internal_id}" to the real remote UID
	// The internal_id remains unchanged, so all foreign keys stay valid
	_, err = db.Exec(`
		UPDATE tasks
		SET uid = ?
		WHERE backend_name = ? AND uid = ? AND list_id = ?
	`, newUID, sm.local.Config.Name, oldUID, listID)
	if err != nil {
		return fmt.Errorf("failed to update task UID: %w", err)
	}

	return nil
}

// GetRemote returns the remote backend.TaskManager
func (sm *SyncManager) GetRemote() backend.TaskManager {
	return sm.remote
}

// getBackendName returns the backend name from the local cache backend
func (sm *SyncManager) getBackendName() string {
	return sm.local.Config.Name
}
