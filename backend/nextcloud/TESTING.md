# Nextcloud Integration Testing

## Overview

The Nextcloud integration tests have been designed to test at a higher level through the **operations layer** (`internal/operations/actions.go`) instead of calling backend methods directly, following the same pattern as the Todoist tests.

## Test Architecture Evolution

### Before: Backend-Level Testing

```
Test → TaskManager.AddTask() → Nextcloud CalDAV API
Test → TaskManager.UpdateTask() → Nextcloud CalDAV API
Test → SyncManager.Sync() → Both backends
```

**Problems:**
- Doesn't test the actual user code path
- Misses bugs in the operations layer
- Doesn't test sync integration properly
- Not a true end-to-end test

### After: Operations-Level Testing

```
Test → HandleAddAction() → TaskManager.AddTask() → Nextcloud CalDAV API
Test → HandleCompleteAction() → TaskManager.UpdateTask() → Nextcloud CalDAV API
Test → SyncManager.Sync() → Both backends
```

**Benefits:**
- ✅ Tests the **actual code path** users execute
- ✅ Catches bugs in operations layer (task selection, validation, sync coordination)
- ✅ Tests sync integration (operations trigger background sync)
- ✅ True end-to-end testing from CLI operations to remote CalDAV API

## Test Files

### `integration_test.go`

Contains three main tests:

1. **`TestNextcloudDirectOperations`**
   - Tests Nextcloud backend directly (no sync)
   - Creates temporary test calendar: `GoSyncTasks Direct Test {timestamp}`
   - Verifies basic CRUD operations work
   - Automatically cleans up test calendar
   - Good for testing Nextcloud CalDAV integration

2. **`TestNextcloudWithSyncOperations`** (COMPREHENSIVE TEST)
   - Tests through operations layer: `HandleAddAction`, `HandleCompleteAction`
   - Uses mock cobra commands and sync providers
   - Captures stdout to avoid test clutter
   - **This is the comprehensive test that catches real bugs**

3. **`TestNextcloudUIDUpdateAfterSync`**
   - Focused test on UID update behavior
   - Tests backend-level sync behavior
   - Verifies pending UIDs are replaced with real CalDAV UIDs

### `test_helpers.go`

Test utilities for operations-level testing:

1. **`mockCommand`**
   - Creates cobra.Command with all necessary flags
   - `newMockCommand()` - creates command with default flags
   - `withFlag(name, value)` - sets flag values

2. **`mockSyncProvider`**
   - Implements `SyncCoordinatorProvider` interface
   - Prevents spawning background sync processes in tests
   - Safe for test environment

3. **`outputCapture`**
   - Captures stdout/stderr during test execution
   - Prevents test output clutter
   - `start()` - begin capturing
   - `stop()` - return captured output

## Test Isolation

**Both integration tests** create their own temporary calendars instead of using the user's default calendar:

**Test Calendars:**
- `TestNextcloudDirectOperations` → `GoSyncTasks Direct Test {timestamp}`
- `TestNextcloudWithSyncOperations` → `GoSyncTasks Test {timestamp}`
- `TestNextcloudUIDUpdateAfterSync` → `GoSyncTasks UID Test {timestamp}`

**Benefits:**
- ✅ **No pollution** - Doesn't add test data to your real task calendars
- ✅ **Isolated** - Each test run uses a unique calendar name (timestamped)
- ✅ **Self-cleaning** - Automatically deletes test calendars when done
- ✅ **Safe** - Can run repeatedly without side effects
- ✅ **Parallel-safe** - Multiple test runs won't interfere with each other

**Calendar Operations Are Tested:**
Calendar creation and deletion are now **part of the test assertions**, not just setup/teardown:
- **Creation**: Tests verify the calendar actually exists after calling `CreateTaskList()`
- **Deletion**: Cleanup verifies the calendar is actually removed after calling `DeleteTaskList()`
- This ensures calendar operations work exactly like the CLI (cmd/gosynctasks/list.go)

**Cleanup:**
Both tests use `defer` to ensure cleanup happens even if the test fails. The cleanup includes verification that the calendar was actually deleted. However, if the test process is killed (e.g., Ctrl+C), you may see leftover test calendars in your Nextcloud account. You can safely delete them manually - they're easy to spot with the "GoSyncTasks" prefix and timestamp.

## Running Tests

### All Tests
```bash
go test ./backend/nextcloud
```

### Specific Test
```bash
# Direct operations (no sync, faster)
go test ./backend/nextcloud -run TestNextcloudDirectOperations

# Operations layer with sync (comprehensive, slower)
go test ./backend/nextcloud -run TestNextcloudWithSyncOperations

# UID update behavior
go test ./backend/nextcloud -run TestNextcloudUIDUpdateAfterSync
```

### With Verbose Output
```bash
go test ./backend/nextcloud -run TestNextcloudWithSyncOperations -v
```

## Prerequisites

Set your Nextcloud credentials via environment variables:
```bash
export GOSYNCTASKS_NEXTCLOUD_HOST=nextcloud.example.com
export GOSYNCTASKS_NEXTCLOUD_USERNAME=your_username
export GOSYNCTASKS_NEXTCLOUD_PASSWORD=your_password
```

Tests will skip if these are not set.

**Note:** For testing with the Docker test server:
```bash
# Start test server first
./scripts/start-test-server.sh

# Set environment variables
export GOSYNCTASKS_NEXTCLOUD_HOST=localhost:8080
export GOSYNCTASKS_NEXTCLOUD_USERNAME=admin
export GOSYNCTASKS_NEXTCLOUD_PASSWORD=admin

# Run tests
go test ./backend/nextcloud -v
```

## What the Comprehensive Test Does

### Test Flow (TestNextcloudWithSyncOperations)

1. **Setup**
   - Creates temporary SQLite cache database
   - Connects to Nextcloud remote backend
   - Creates sync manager
   - Sets up mock cobra commands and sync provider

2. **Test 0: Create Calendar (tests CreateTaskList like CLI)**
   - Creates unique test calendar on remote: `GoSyncTasks Test {timestamp}`
   - Verifies calendar exists in remote task lists (tests calendar creation works)
   - Creates same calendar in cache for sync operations
   - Verifies calendar exists in cache task lists
   - Sets up deferred cleanup with verification

3. **Test 1: Add Task via Operations Layer**
   - Calls `HandleAddAction()` (what CLI actually calls)
   - Verifies task is added to cache with pending UID
   - **Tests:** Task creation, pending UID generation

4. **Test 2: Sync to Nextcloud**
   - Calls `SyncManager.Sync()` to push to remote
   - Verifies task appears on Nextcloud
   - **Tests:** Push sync, UID replacement

5. **Test 3: Complete Task via Operations Layer**
   - Calls `HandleCompleteAction()` with task summary
   - Finds task by summary (tests task search)
   - Marks task as DONE/COMPLETED
   - **Tests:** Task selection, status update, operations layer logic

6. **Test 4: Sync Completion to Nextcloud**
   - Pushes completion status to remote
   - Verifies task is completed on Nextcloud
   - **Tests:** Update sync

7. **Test 5: Delete Task**
   - Currently uses backend directly (HandleDeleteAction requires confirmation)
   - **TODO:** Add non-interactive mode for HandleDeleteAction

8. **Test 6: Sync Deletion**
   - Pushes deletion to remote
   - Verifies task is removed from Nextcloud
   - **Tests:** Delete sync

9. **Cleanup: Delete Calendar (tests DeleteTaskList like CLI)**
   - Deferred cleanup runs at end
   - Calls `DeleteTaskList()` (exactly how CLI deletes calendars)
   - Verifies calendar is removed from remote
   - **Tests:** Calendar deletion works correctly

## Future Improvements

### 1. Non-Interactive Mode for HandleDeleteAction

Add a flag or config to skip confirmation prompts:

```go
// In operations/actions.go
type ActionOptions struct {
    SkipConfirmation bool
}

func HandleDeleteAction(..., opts *ActionOptions) error {
    if !opts.SkipConfirmation {
        confirmed, _ := utils.PromptConfirmation(...)
        if !confirmed {
            return fmt.Errorf("cancelled")
        }
    }
    // ... rest of delete logic
}
```

Then test can use:
```go
err = operations.HandleDeleteAction(
    cmd, cacheBackend, cfg, &testCalendar,
    testTaskName,
    mockSync,
    &operations.ActionOptions{SkipConfirmation: true},
)
```

### 2. Test HandleUpdateAction

Add test for updating task fields (description, priority, dates):

```go
// Update task description
updateCmd := newMockCommand().
    withFlag("description", "Updated description").
    withFlag("priority", 1)

err = operations.HandleUpdateAction(
    updateCmd.Command,
    cacheBackend,
    cfg,
    &testCalendar,
    testTaskName,
    mockSync,
)
```

### 3. Test Error Paths

Add tests for:
- Invalid task summary
- Task not found
- Sync failures
- Network errors
- Conflict resolution
- CalDAV PROPFIND/REPORT errors

### 4. Integration with Background Sync

Currently mocks sync provider. Could test:
- Actual background sync triggering
- Auto-sync behavior
- Staleness detection

### 5. Test Subtask Operations

Add tests for:
- Creating subtasks with `-P "Parent"` flag
- Creating nested hierarchies with path syntax
- Hierarchical sync behavior
- Parent-child relationship preservation

## Debugging Tests

### View Test Output

```bash
# Run with verbose output
go test ./backend/nextcloud -run TestNextcloudWithSyncOperations -v

# See what's happening in each step
# The test logs show:
# - Test 0: Creating test calendar...
# - Test 1: Adding task via operations layer...
# - Test 2: Syncing to Nextcloud...
# - etc.
```

### Check Nextcloud State

During test development, you can check Nextcloud manually:
1. Run test up to sync point
2. Add `time.Sleep(10 * time.Second)` to pause
3. Check Nextcloud web UI
4. Verify task/calendar state

### Common Issues

**"Task not found after HandleCompleteAction"**
- Check if task search is working correctly
- Verify task summary matches exactly
- Check if task was filtered out

**"UID still pending after sync"**
- This is the bug we're testing for!
- Check `SyncManager.Sync()` UID update logic
- Verify `UpdateUIDMapping()` is called

**"Test times out"**
- Nextcloud API might be slow
- Increase sleep times between operations
- Check network connectivity

**"Calendar still exists after deletion"**
- CalDAV deletion might be delayed
- Verify cleanup defer is running
- Check for errors in DeleteTaskList

**"CalDAV authentication failed"**
- Verify GOSYNCTASKS_NEXTCLOUD_HOST, GOSYNCTASKS_NEXTCLOUD_USERNAME, GOSYNCTASKS_NEXTCLOUD_PASSWORD are set
- Check if test server is running (`docker ps`)
- Try accessing Nextcloud web UI manually

## Why This Matters

The refactored test catches the **real bug** in the codebase:

**Bug:** When tasks are added to the cache, they get a `pending-{uuid}` UID. After sync pushes them to Nextcloud, the UID should be updated to the Nextcloud CalDAV UID. But the operations layer (HandleCompleteAction, HandleDeleteAction) was using the cached task objects, which still had the pending UID.

**How operations-level test catches it:**
1. Add task via HandleAddAction → gets pending UID
2. Sync → should update UID in cache to Nextcloud CalDAV UID
3. Complete via HandleCompleteAction → searches by summary, finds task with (hopefully) real UID
4. If UID wasn't updated, the complete operation uses wrong UID
5. Sync tries to push update with wrong UID → fails

Testing at the backend level wouldn't catch this because:
- Backend tests pass the UID directly
- They don't go through task search by summary
- They don't test the operations layer's task selection logic

## Nextcloud-Specific Details

### CalDAV Protocol

Nextcloud uses CalDAV protocol for task synchronization:
- **PROPFIND**: Discover calendars and tasks
- **REPORT**: Query tasks with filters
- **PUT**: Create/update tasks
- **DELETE**: Remove tasks

### Task Format

Tasks are stored as iCalendar VTODO format:
- **UID**: Unique identifier (CalDAV generates these)
- **SUMMARY**: Task title
- **STATUS**: NEEDS-ACTION, COMPLETED, IN-PROCESS, CANCELLED
- **PRIORITY**: 0-9 (CalDAV standard)
- **DTSTART**: Start date
- **DUE**: Due date
- **CATEGORIES**: Tags

### Status Translation

The backend translates between internal and CalDAV statuses:
- Internal `TODO` ↔ CalDAV `NEEDS-ACTION`
- Internal `DONE` ↔ CalDAV `COMPLETED`
- Internal `PROCESSING` ↔ CalDAV `IN-PROCESS`
- Internal `CANCELLED` ↔ CalDAV `CANCELLED`

## Conclusion

The refactored test provides:
- **Higher confidence** - tests actual user code paths
- **Better bug detection** - catches integration issues
- **More realistic** - mimics real CLI usage
- **Future-proof** - will catch regressions in operations layer
- **CalDAV-aware** - tests protocol-specific behavior

This is how integration tests should be written - test through the highest reasonable level that still allows for automation.
