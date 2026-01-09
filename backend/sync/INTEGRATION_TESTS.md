# Sync Integration Tests

This directory contains integration tests for the sync functionality. The tests are organized into two files:

## Test Files

### 1. `integration_test.go` - Direct Backend Tests
These tests directly use the sync manager with Nextcloud backend (lower-level tests):
- `TestSyncPushToNextcloud` - Tests pushing local changes to Nextcloud
- `TestSyncPullFromNextcloud` - Tests pulling remote changes from Nextcloud
- `TestSyncBidirectional` - Tests bidirectional sync (push and pull)
- `TestSyncConflictResolution` - Tests conflict resolution strategies
- `TestSyncDeleteTask` - Tests syncing task deletions

### 2. `cli_integration_test.go` - CLI Operations Tests
These tests use the operations layer (same code path as CLI commands):
- `TestSyncWorkflowWithCLIOperations` - End-to-end sync workflow using CLI operations (add, complete, update)
- `TestSyncNoAccidentalDeletionOnDBReset` - **Critical test** that verifies tasks are not accidentally deleted from remote when local DB is reset

## Critical Test: No Accidental Deletion

The `TestSyncNoAccidentalDeletionOnDBReset` test covers an important scenario:

**Scenario**: User has tasks synced to Nextcloud, then deletes their local SQLite database (fresh start)

**Expected Behavior**:
- First sync after DB deletion should PULL tasks from remote
- Tasks should NOT be deleted from remote
- User should be able to continue working with all existing tasks

**Test Phases**:
1. Create list and add 2 tasks locally
2. Sync to push tasks to remote
3. Delete local SQLite DB and recreate (simulating fresh start)
4. **Critical**: Sync again - should pull tasks from remote, not delete them
5. Add new task and verify sync still works correctly

## Running the Tests

### Prerequisites

The integration tests require a Nextcloud test server. You can start one using Docker:

```bash
# From the project root
./scripts/start-test-server.sh
```

This will start a Nextcloud server and set the required environment variables.

### Environment Variables

The tests need these environment variables:
- `GOSYNCTASKS_NEXTCLOUD_HOST` - Nextcloud server host (e.g., `localhost:8080`)
- `GOSYNCTASKS_NEXTCLOUD_USERNAME` - Nextcloud username (e.g., `admin`)
- `GOSYNCTASKS_NEXTCLOUD_PASSWORD` - Nextcloud password (e.g., `admin`)

Or set `SKIP_INTEGRATION=1` to explicitly skip all integration tests.

### Running Tests

```bash
# Run all sync tests (unit + integration)
go test ./backend/sync -v

# Run only CLI integration tests
go test ./backend/sync -v -run="TestSyncWorkflowWithCLIOperations|TestSyncNoAccidentalDeletionOnDBReset"

# Run only direct backend integration tests
go test ./backend/sync -v -run="TestSyncPushToNextcloud|TestSyncPullFromNextcloud|TestSyncBidirectional"

# Skip integration tests
SKIP_INTEGRATION=1 go test ./backend/sync -v
```

## Test Behavior

When Nextcloud is not available:
- Tests automatically skip with a helpful message
- No test failures occur
- Unit tests still run normally

When Nextcloud is available:
- All integration tests run against the real Nextcloud server
- Tests create and cleanup their own test lists
- Tests are isolated and can run in parallel

## Test Helpers

The `cli_integration_test.go` file includes test helpers:
- `mockCommand` - Creates cobra commands with flags for testing
- `mockSyncProvider` - Prevents background sync processes during tests
- `outputCapture` - Captures stdout/stderr during test execution
- `createTestNextcloudBackend` - Creates a Nextcloud backend for testing
- `cleanupTestList` - Cleans up tasks from test lists

These helpers ensure tests use the same code paths as the actual CLI commands.

## Recent Fixes

### Integer Flag Conversion (v1.0.1)
Fixed bug in `withFlag()` helper where integer values were incorrectly converted using `string(rune(v))` instead of `fmt.Sprintf("%d", v)`. This caused priority=9 to be converted to a tab character instead of "9".

### Resilient Task Verification
Updated tests to verify specific test tasks by name instead of expecting exact task counts. This makes tests more resilient to:
- Existing tasks in the test list from previous runs
- Multiple operations being queued and processed together
- Variations in sync behavior

Tests now log:
- Pending operations before sync
- Actual vs expected values in failures
- Total task counts alongside specific checks
- Phase-by-phase progress
