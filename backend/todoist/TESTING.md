# Todoist Integration Testing

## Overview

The Todoist integration tests have been refactored to test at a higher level through the **operations layer** (`internal/operations/actions.go`) instead of calling backend methods directly.

## Test Architecture Evolution

### Before: Backend-Level Testing

```
Test → TaskManager.AddTask() → Todoist API
Test → TaskManager.UpdateTask() → Todoist API
Test → SyncManager.Sync() → Both backends
```

**Problems:**
- Doesn't test the actual user code path
- Misses bugs in the operations layer
- Doesn't test sync integration properly
- Not a true end-to-end test

### After: Operations-Level Testing

```
Test → HandleAddAction() → TaskManager.AddTask() → Todoist API
Test → HandleCompleteAction() → TaskManager.UpdateTask() → Todoist API
Test → SyncManager.Sync() → Both backends
```

**Benefits:**
- ✅ Tests the **actual code path** users execute
- ✅ Catches bugs in operations layer (task selection, validation, sync coordination)
- ✅ Tests sync integration (operations trigger background sync)
- ✅ True end-to-end testing from CLI operations to remote API

## Test Files

### `integration_test.go`

Contains three main tests:

1. **`TestTodoistDirectOperations`**
   - Tests Todoist backend directly (no sync)
   - Creates temporary test list: `GoSyncTasks Direct Test {timestamp}`
   - Verifies basic CRUD operations work
   - Automatically cleans up test list
   - Good for testing Todoist API integration

2. **`TestTodoistWithSyncOperations`** (REFACTORED)
   - Tests through operations layer: `HandleAddAction`, `HandleCompleteAction`
   - Uses mock cobra commands and sync providers
   - Captures stdout to avoid test clutter
   - **This is the comprehensive test that catches real bugs**

3. **`TestTodoistUIDUpdateAfterSync`**
   - Focused test on UID update behavior
   - Tests backend-level sync behavior
   - Verifies pending UIDs are replaced with real UIDs

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

**Both integration tests** create their own temporary lists instead of using the user's Inbox:

**Test Lists:**
- `TestTodoistDirectOperations` → `GoSyncTasks Direct Test {timestamp}`
- `TestTodoistWithSyncOperations` → `GoSyncTasks Test {timestamp}`

**Benefits:**
- ✅ **No pollution** - Doesn't add test data to your real task lists
- ✅ **Isolated** - Each test run uses a unique list name (timestamped)
- ✅ **Self-cleaning** - Automatically deletes test lists when done
- ✅ **Safe** - Can run repeatedly without side effects
- ✅ **Parallel-safe** - Multiple test runs won't interfere with each other

**List Operations Are Tested:**
List creation and deletion are now **part of the test assertions**, not just setup/teardown:
- **Creation**: Tests verify the list actually exists after calling `CreateTaskList()`
- **Deletion**: Cleanup verifies the list is actually removed after calling `DeleteTaskList()`
- This ensures list operations work exactly like the CLI (cmd/gosynctasks/list.go)

**Cleanup:**
Both tests use `defer` to ensure cleanup happens even if the test fails. The cleanup includes verification that the list was actually deleted. However, if the test process is killed (e.g., Ctrl+C), you may see leftover test lists in your Todoist account. You can safely delete them manually - they're easy to spot with the "GoSyncTasks" prefix and timestamp.

## Running Tests

### All Tests
```bash
go test ./backend/todoist
```

### Specific Test
```bash
# Direct operations (no sync, faster)
go test ./backend/todoist -run TestTodoistDirectOperations

# Operations layer with sync (comprehensive, slower)
go test ./backend/todoist -run TestTodoistWithSyncOperations

# UID update behavior
go test ./backend/todoist -run TestTodoistUIDUpdateAfterSync
```

### With Verbose Output
```bash
go test ./backend/todoist -run TestTodoistWithSyncOperations -v
```

## Prerequisites

Set your Todoist API token:
```bash
export TODOIST_API_TOKEN=your_token_here
```

Tests will skip if token is not set.

## What the Comprehensive Test Does

### Test Flow (TestTodoistWithSyncOperations)

1. **Setup**
   - Creates temporary SQLite cache database
   - Connects to Todoist remote backend
   - Creates sync manager
   - Sets up mock cobra commands and sync provider

2. **Test 0: Create List (tests CreateTaskList like CLI)**
   - Creates unique test list on remote: `GoSyncTasks Test {timestamp}`
   - Verifies list exists in remote task lists (tests list creation works)
   - Creates same list in cache for sync operations
   - Verifies list exists in cache task lists
   - Sets up deferred cleanup with verification

3. **Test 1: Add Task via Operations Layer**
   - Calls `HandleAddAction()` (what CLI actually calls)
   - Verifies task is added to cache with pending UID
   - **Tests:** Task creation, pending UID generation

4. **Test 2: Sync to Todoist**
   - Calls `SyncManager.Sync()` to push to remote
   - Verifies task appears on Todoist
   - **Tests:** Push sync, UID replacement

5. **Test 3: Complete Task via Operations Layer**
   - Calls `HandleCompleteAction()` with task summary
   - Finds task by summary (tests task search)
   - Marks task as DONE
   - **Tests:** Task selection, status update, operations layer logic

6. **Test 4: Sync Completion to Todoist**
   - Pushes completion status to remote
   - Verifies task is completed on Todoist
   - **Tests:** Update sync

7. **Test 5: Delete Task**
   - Currently uses backend directly (HandleDeleteAction requires confirmation)
   - **TODO:** Add non-interactive mode for HandleDeleteAction

8. **Test 6: Sync Deletion**
   - Pushes deletion to remote
   - Verifies task is removed from Todoist
   - **Tests:** Delete sync

9. **Cleanup: Delete List (tests DeleteTaskList like CLI)**
   - Deferred cleanup runs at end
   - Calls `DeleteTaskList()` (exactly how CLI deletes lists)
   - Verifies list is removed from remote
   - **Tests:** List deletion works correctly

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
    cmd, cacheBackend, cfg, &inboxList,
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
    &inboxList,
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

### 4. Integration with Background Sync

Currently mocks sync provider. Could test:
- Actual background sync triggering
- Auto-sync behavior
- Staleness detection

## Debugging Tests

### View Test Output

```bash
# Run with verbose output
go test ./backend/todoist -run TestTodoistWithSyncOperations -v

# See what's happening in each step
# The test logs show:
# - Test 1: Adding task via operations layer...
# - Test 2: Syncing to Todoist...
# - etc.
```

### Check Todoist State

During test development, you can check Todoist manually:
1. Run test up to sync point
2. Add `time.Sleep(10 * time.Second)` to pause
3. Check Todoist web UI
4. Verify task state

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
- Todoist API might be slow
- Increase sleep times between operations
- Check network connectivity

## Why This Matters

The refactored test caught the **real bug** in the codebase:

**Bug:** When tasks are added to the cache, they get a `pending-{uuid}` UID. After sync pushes them to Todoist, the UID should be updated to the Todoist UID. But the operations layer (HandleCompleteAction, HandleDeleteAction) was using the cached task objects, which still had the pending UID.

**How operations-level test catches it:**
1. Add task via HandleAddAction → gets pending UID
2. Sync → should update UID in cache
3. Complete via HandleCompleteAction → searches by summary, finds task with (hopefully) real UID
4. If UID wasn't updated, the complete operation uses wrong UID
5. Sync tries to push update with wrong UID → fails

Testing at the backend level wouldn't catch this because:
- Backend tests pass the UID directly
- They don't go through task search by summary
- They don't test the operations layer's task selection logic

## Conclusion

The refactored test provides:
- **Higher confidence** - tests actual user code paths
- **Better bug detection** - catches integration issues
- **More realistic** - mimics real CLI usage
- **Future-proof** - will catch regressions in operations layer

This is how integration tests should be written - test through the highest reasonable level that still allows for automation.
