# Pending UID Replacement Fix

## Problem

When syncing tasks from SQLite cache to Nextcloud, tasks with pending UIDs (e.g., `pending-1`, `pending-123`) were not being updated to proper CalDAV UIDs after successful creation on the remote server.

### Root Cause

1. **SQLite cache creates pending UIDs**: When tasks are created in the cache, they get temporary UIDs like `pending-{internal_id}` until synced to the remote
2. **Nextcloud backend accepted pending UIDs as-is**: The `AddTask` method used whatever UID was provided, including pending UIDs
3. **UID update never triggered**: Since `AddTask` returned the same UID it received, the sync manager's condition `if remoteUID != task.UID` was never true
4. **Result**: Tasks kept their `pending-*` UIDs even after successful sync

## Solution

Modified `backend/nextcloud/backend.go` to detect and replace pending UIDs:

```go
func (nB *NextcloudBackend) AddTask(listID string, task backend.Task) (string, error) {
    // Set defaults
    if task.UID == "" || strings.HasPrefix(task.UID, "pending-") {
        // Generate a new UID if empty or if it's a pending UID from cache
        task.UID = fmt.Sprintf("task-%d", time.Now().Unix())
    }
    // ... rest of function
}
```

### How It Works

1. **Detect pending UIDs**: Check if UID starts with `"pending-"`
2. **Generate proper CalDAV UID**: Replace with `task-{timestamp}` format
3. **Return new UID**: The sync manager receives a different UID than what was sent
4. **Trigger UID update**: Sync manager calls `updateLocalTaskUID()` to update cache

## Testing

### Unit Test

A new unit test verifies the fix:

```bash
go test ./backend/nextcloud -v -run TestNextcloudBackend_AddTask_PendingUIDReplacement
```

This test verifies:
- Pending UIDs (`pending-123`) are replaced with proper UIDs (`task-*`)
- Empty UIDs are replaced
- Normal custom UIDs are preserved

### Integration Test

To run the full integration test with a real Nextcloud server:

```bash
# Set environment variables for your test server
export GOSYNCTASKS_NEXTCLOUD_HOST="http://localhost:8080"
export GOSYNCTASKS_NEXTCLOUD_USERNAME="admin"
export GOSYNCTASKS_NEXTCLOUD_PASSWORD="admin"

# Run the integration test
go test ./backend/nextcloud -v -run TestNextcloudWithSyncOperations
```

Expected output should show:
```
Initial sync: pushed 1 tasks
Task UID after sync: task-1234567890  # (not pending-1)
```

## Impact

- **Fixes**: UID mismatch bug in sync operations
- **Preserves**: Normal UID behavior for non-pending tasks
- **Compatible**: Works with existing sync architecture
- **No breaking changes**: Only affects tasks with pending UIDs

## Related Files

- `backend/nextcloud/backend.go`: Main fix
- `backend/nextcloud/backend_test.go`: Unit test
- `backend/nextcloud/integration_test.go`: Integration test
- `backend/sync/manager.go`: Sync logic that calls `updateLocalTaskUID()`
- `backend/sqlite/backend.go`: Creates pending UIDs
