# Todoist Integration Test Improvements

## Summary

Applied the same improvements from the Nextcloud integration test to ensure consistent testing across all backends.

## Changes Made

### 1. Added Config Setup with Sync Enabled

**Before:**
```go
cfg := &config.Config{
    DateFormat: "2006-01-02",
}
```

**After:**
```go
cfg := &config.Config{
    DateFormat: "2006-01-02",
    Sync: &config.SyncConfig{
        Enabled:            true,
        AutoSync:           true,
        LocalBackend:       "sqlite",
        ConflictResolution: "server_wins",
    },
    Backends: map[string]backend.BackendConfig{
        "todoist": {
            Name:     "todoist",
            Type:     "todoist",
            Enabled:  true,
            APIToken: apiToken,
        },
    },
}
// Set this as the global config so triggerPushSync sees it
config.SetConfigForTest(cfg)
```

**Why:** The global config singleton must be set for `triggerPushSync` in operations to see that sync is enabled.

### 2. Fixed List ID Mismatch

**Before:**
```go
// Remote list ID: from remoteBackend.CreateTaskList()
// Cache list ID: from cacheBackend.CreateTaskList() - DIFFERENT!
```

**After:**
```go
// Insert list directly into cache database with the remote list ID
// This ensures sync expects cache and remote to use the same list IDs
db, err := cacheBackend.GetDB()
_, err = db.Exec(`
    INSERT INTO list_sync_metadata (list_id, backend_name, list_name, list_color, created_at, modified_at)
    VALUES (?, ?, ?, ?, ?, ?)
`, testListID, "todoist", testListName, "", time.Now().Unix(), time.Now().Unix())

cacheListID := testListID  // Same as remote!
```

**Why:** Sync operations require matching list IDs between cache and remote. If they differ, sync operations won't find the correct list.

### 3. Added Debug Output for Pending Operations

**Added before sync:**
```go
// Check pending operations before sync (for debugging)
pendingOps, err := cacheBackend.GetPendingSyncOperations()
if err != nil {
    t.Fatalf("Failed to get pending ops: %v", err)
}
t.Logf("Pending operations before sync: %d", len(pendingOps))
for _, op := range pendingOps {
    t.Logf("  - Operation: %s, TaskUID: %s, ListID: %s", op.Operation, op.TaskUID, op.ListID)
}
```

**Why:** Helps diagnose sync issues by showing what operations are queued before sync executes.

## Todoist vs Nextcloud: Pending UID Handling

### Important Difference

**Todoist does NOT have the pending UID issue that Nextcloud had:**

- **Nextcloud:** Was accepting pending UIDs as-is for CalDAV UIDs → Fixed by detecting and replacing
- **Todoist:** API assigns its own IDs, ignoring any UID sent → Already working correctly

### Why Todoist Works

In `backend/todoist/backend.go`:
```go
func (tb *TodoistBackend) AddTask(listID string, task backend.Task) (string, error) {
    req := toCreateTaskRequest(task, listID)  // UID not included in request

    createdTask, err := tb.apiClient.CreateTask(req)
    if err != nil {
        return "", fmt.Errorf("failed to create task: %w", err)
    }

    // Return the Todoist-assigned task ID (not the pending UID)
    return createdTask.ID, nil
}
```

The `toCreateTaskRequest` function (in `mapper.go`) doesn't include the task UID:
```go
func toCreateTaskRequest(task backend.Task, projectID string) CreateTaskRequest {
    req := CreateTaskRequest{
        Content:     task.Summary,      // ✓ Used
        Description: task.Description,  // ✓ Used
        ProjectID:   projectID,         // ✓ Used
        ParentID:    task.ParentUID,    // ✓ Used
        Labels:      task.Categories,   // ✓ Used
        // task.UID is NOT sent to API    ✗ Ignored
    }
    // ... set priority and due date
    return req
}
```

## Testing

### Run Integration Test

Requires Todoist API token:

```bash
export TODOIST_API_TOKEN="your-api-token"
go test ./backend/todoist -v -run TestTodoistWithSyncOperations
```

### Expected Output

```
Test 0a: Creating test list on remote via CreateTaskList: GoSyncTasks Test 20260105...
✓ Remote list created with ID: 1234567890
✓ Remote list verified
Test 0b: Creating test list in cache with remote ID: 1234567890
✓ Cache list created with remote ID: 1234567890
Test 1: Adding task via operations layer...
Task added with UID: pending-1
Test 2: Syncing to Todoist...
Pending operations before sync: 1
  - Operation: create, TaskUID: pending-1, ListID: 1234567890
Initial sync: pushed 1 tasks
Task UID after sync: 8234567890  ✅ (Todoist-assigned ID, not pending-1)
```

## Impact

- **Consistency:** Same test structure across Nextcloud and Todoist
- **Better debugging:** Pending operations are now visible in test output
- **Correct behavior:** List IDs match between cache and remote
- **No breaking changes:** Only affects test setup, not production code

## Related Files

- `backend/todoist/integration_test.go`: Updated test
- `backend/todoist/backend.go`: AddTask implementation (unchanged - already correct)
- `backend/todoist/mapper.go`: toCreateTaskRequest (unchanged - doesn't use UID)
- `internal/config/config.go`: SetConfigForTest helper
