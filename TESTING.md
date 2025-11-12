# gosynctasks Testing Guide

Comprehensive testing guide covering Docker test environment setup, unit tests, integration tests, and manual testing.

## Table of Contents

- [Docker Test Environment](#docker-test-environment)
- [Quick Start](#quick-start)
- [Unit Testing](#unit-testing)
- [Integration Testing](#integration-testing)
- [Manual Testing](#manual-testing)
- [Test Coverage](#test-coverage)
- [Writing Tests](#writing-tests)

## Docker Test Environment

### Nextcloud Test Server (Docker)

#### TL;DR - Quick Commands

```bash
# Start server
./scripts/start-test-server.sh

# Update your config with: nextcloud://admin:admin123@localhost:8080

# Test it
gosynctasks                    # List all task lists
gosynctasks MyList add "Test"  # Create a task

# Stop server (keeps data)
./scripts/stop-test-server.sh
```

#### Prerequisites

- Docker installed and running
- Docker Compose (or Docker with compose plugin)
- Port 8080 available on your machine

#### Quick Start

1. **Start the test server:**
   ```bash
   ./scripts/start-test-server.sh
   ```

   This will:
   - Start Nextcloud, PostgreSQL, and Redis containers
   - Wait for all services to be healthy
   - Install the Tasks and Calendar apps automatically
   - Display connection details when ready

2. **Configure gosynctasks:**

   The script will show you the exact configuration to add. Update your `~/.config/gosynctasks/config.json`:

   ```json
   {
     "connector": {
       "url": "nextcloud://admin:admin123@localhost:8080",
       "insecure_skip_verify": false
     },
     "ui": "cli"
   }
   ```

   **Note:** The URL format is `nextcloud://username:password@host:port`

3. **Access Nextcloud web interface (optional):**
   - URL: http://localhost:8080
   - Username: `admin`
   - Password: `admin123`

#### Testing Workflow with Docker

1. **Create a task list in Nextcloud:**
   - Log in to http://localhost:8080
   - Go to Tasks app (left sidebar)
   - Create a new list (e.g., "TestTasks")

2. **Test with gosynctasks:**
   ```bash
   # List all task lists
   gosynctasks

   # View tasks in a list
   gosynctasks TestTasks

   # Add a task
   gosynctasks TestTasks add "Test task 1"

   # Add task with details
   gosynctasks TestTasks add "Important task" -d "This is a test" -p 1

   # Update a task
   gosynctasks TestTasks update "Test task" -s DONE

   # Complete a task
   gosynctasks TestTasks complete "Important task"
   ```

3. **Verify in Nextcloud:**
   - Refresh the Tasks app in browser
   - Changes should appear immediately

#### Server Management

**Stop the server:**
```bash
./scripts/stop-test-server.sh
```

You'll be prompted whether to keep or delete data:
- **Option 1**: Stop server, keep data (default) - You can restart later with your tasks intact
- **Option 2**: Stop server and delete ALL data - Clean slate for fresh testing

**View logs:**
```bash
docker compose logs -f nextcloud
docker compose logs -f db
docker compose logs -f redis
```

**Restart the server (with existing data):**
```bash
./scripts/start-test-server.sh
```

Your tasks and configuration will be preserved.

#### Troubleshooting Docker Setup

**Server won't start:**
```bash
# Check if port 8080 is already in use
lsof -i :8080

# If port is in use, edit docker-compose.yml and change the port
```

**Tasks app not installed:**
```bash
docker compose exec nextcloud php occ app:install tasks
docker compose exec nextcloud php occ app:enable tasks
```

**Authentication failures:**
```bash
# Reset admin password
docker compose exec nextcloud php occ user:resetpassword admin
```

**Connection refused errors:**
```bash
# Make sure server is running
docker compose ps

# Check if Nextcloud is healthy
docker compose exec nextcloud curl -f http://localhost/status.php
```

**Clear cache and start fresh:**
```bash
./scripts/stop-test-server.sh
# Select option 2 to delete all data
./scripts/start-test-server.sh
```

#### Security Warning

**This is a TEST server only!**

- Default credentials are publicly known (`admin:admin123`)
- No HTTPS/TLS encryption
- Should NOT be exposed to the internet
- Should NOT be used for production data

## Quick Start

```bash
# Run all unit tests
go test ./...

# Run with coverage
go test ./... -cover

# Run with verbose output
go test ./... -v

# Run specific package tests
go test ./backend
go test ./internal/views/builder

# Run with race detection
go test ./... -race
```

## Unit Testing

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./backend
go test ./internal/views/builder
go test ./internal/config

# Specific test function
go test ./backend -run TestParseVTODO
go test ./backend -run TestParseVTODO/valid_task

# Tests matching pattern
go test ./... -run ".*Validation.*"
```

### Test Coverage by Package

| Package                         | Coverage | Status |
|---------------------------------|----------|--------|
| `backend`                       | 65.2%    | ✅ Good |
| `internal/config`               | 53.7%    | ⚠️ Fair |
| `internal/views`                | 50.4%    | ⚠️ Fair |
| `internal/views/builder`        | 44.5%    | ⚠️ Fair |
| `internal/views/formatters`     | 23.6%    | ⚠️ Needs improvement |
| `internal/operations`           | 4.0%     | ❌ Low |
| `cmd/gosynctasks`               | 3.3%     | ❌ Low |

### Generating Coverage Reports

```bash
# Generate coverage profile
go test ./... -coverprofile=coverage.out

# View coverage in browser
go tool cover -html=coverage.out

# Show coverage by function
go tool cover -func=coverage.out

# Coverage for specific package
go test ./backend -coverprofile=backend.out
go tool cover -html=backend.out
```

### Key Test Files

```
backend/
├── taskManager_test.go          # Status translation, validation
├── parseVTODOs_test.go          # iCalendar parsing
├── nextcloudBackend_test.go     # Nextcloud backend integration
├── gitBackend_test.go           # Git backend integration
├── formatWithView_test.go       # Task formatting, start dates
└── selector_test.go             # Backend selection

internal/views/builder/
├── types_test.go                # ViewBuilder, field operations (10 tests)
├── model_test.go                # State machine, UI (13 tests)
└── validation_test.go           # Input validation (10 tests)

internal/views/
├── fields_test.go               # Field registry, format validation
├── loader_test.go               # View loading from YAML
├── renderer_test.go             # Task rendering with views
└── resolver_test.go             # View resolution logic

internal/config/
└── config_test.go               # Config loading, validation

internal/operations/
└── dates_test.go                # Date parsing and formatting

cmd/gosynctasks/
└── view_test.go                 # CLI view commands
```

## Integration Testing

### Setup for Integration Tests

```bash
# Build the binary
go build -o gosynctasks ./cmd/gosynctasks

# Create test alias for convenience
alias gst="go run ./cmd/gosynctasks --config ./gosynctasks/config"
```

### Backend Integration Tests

#### Nextcloud Backend

The Nextcloud backend tests use mock HTTP servers to simulate CalDAV responses:

```bash
# Run Nextcloud backend tests
go test ./backend -run TestNextcloud -v

# Test specific scenarios
go test ./backend -run TestNextcloudBackend_GetTasks
go test ./backend -run TestNextcloudBackend_Authentication
```

These tests verify:
- CalDAV XML query generation
- iCalendar VTODO parsing
- HTTP request/response handling
- Authentication handling
- Error cases (401, 404, network errors)

#### Git Backend

```bash
# Run Git backend tests
go test ./backend -run TestGit -v
```

Tests verify:
- Markdown task parsing
- Git repository detection
- File reading/writing
- Task metadata extraction

### File Backend

```bash
# Run File backend tests
go test ./backend -run TestFile -v
```

## Manual Testing

### Prerequisites

1. Configure test environment:
```bash
# Use test config
export GST_CONFIG=./gosynctasks/config

# Or use explicit config flag
--config ./gosynctasks/config
```

2. Ensure you have access to a test task list (named "Test" in examples below)

### Essential Test Workflow

#### 1. Add Tasks

```bash
gst Test add "Buy groceries" -d "Get milk and eggs" -p 5
gst Test add "Write report" -p 1
gst Test add "Call dentist" -p 3
```

**Verify:**
- Tasks created successfully
- Different priorities assigned
- Descriptions stored

#### 2. List Tasks

```bash
# Basic view
gst Test

# All metadata
gst Test -v all
```

**Verify:**
- Priority sorting (1=highest priority first)
- Colors:
  - Priority 1-4: Red
  - Priority 5: Yellow
  - Priority 6-9: Blue
- Status symbols:
  - ○ for TODO
  - ● for PROCESSING
  - ✓ for DONE
  - ✗ for CANCELLED
- Description display (truncated to 70 chars)

#### 3. Update Task

```bash
gst Test update "Call dentist" -p 1 -d "Schedule cleaning appointment"
```

**Verify:**
- Priority changed
- Description updated
- Modified timestamp updated

#### 4. Complete Task

```bash
# Exact match
gst Test complete "Buy groceries"

# Partial match (requires confirmation)
echo "y" | gst Test complete "groceries"
```

**Verify:**
- Partial match shows confirmation prompt
- Status changes to DONE (✓ green)
- Completion timestamp set

#### 5. Filter by Status

```bash
# Show only TODO tasks
gst Test -s TODO

# Show only completed tasks
gst Test -s DONE

# Multiple statuses
gst Test -s TODO,PROCESSING
```

**Verify:**
- Filtering works correctly
- Only requested statuses shown

#### 6. Change Task Status

```bash
# Mark as processing
gst Test update "Write report" -s PROCESSING

# Mark as cancelled
gst Test update "Write report" -s CANCELLED

# Back to TODO
gst Test update "Write report" -s TODO
```

**Verify:**
- Status changes reflected
- Symbols update correctly
- Completion timestamp cleared when moving from DONE

#### 7. Delete Tasks

```bash
echo "y" | gst Test delete "Write report"
echo "y" | gst Test delete "Call dentist"
echo "y" | gst Test delete "Buy groceries"
```

**Verify:**
- Confirmation prompt shown
- Tasks removed from list
- List empty after deleting all

### Advanced Features Testing

#### Priority-Based Workflow

```bash
# Add tasks with various priorities
gst Test add "Critical bug" -p 1
gst Test add "Important feature" -p 2
gst Test add "Nice to have" -p 7
gst Test add "Low priority" -p 9

# List and verify sorting
gst Test
```

**Verify:**
- Tasks sorted by priority (1 first)
- Color coding matches priority
- 0 priority (undefined) goes last

#### Date-Based Tasks

```bash
# Add task with dates
gst Test add "Project deadline" -p 1 --due 2025-12-31
gst Test add "Meeting" --start 2025-11-15

# View with dates
gst Test -v all
```

**Verify:**
- Due dates displayed
- Start dates displayed with color coding:
  - Cyan: Past/present start date
  - Yellow: Within 3 days
  - Gray: Beyond 3 days
- Date format correct (YYYY-MM-DD)

#### Search and Partial Matching

```bash
# Add similar tasks
gst Test add "Review PR #123"
gst Test add "Review PR #456"
gst Test add "Review documentation"

# Search with partial match
gst Test update "PR #123" -p 1
```

**Verify:**
- Exact matches proceed immediately
- Partial matches show selection menu
- Multiple matches show all options
- Can select by number or cancel

#### Interactive List Selection

```bash
# Run without list name
gst
```

**Verify:**
- Shows all available lists
- Displays task count for each
- Shows descriptions
- Allows selection by number
- Can cancel with 0
- Terminal width adaptation (40-100 chars)

### Custom Views Testing

#### Interactive View Builder

```bash
# Launch interactive builder
gst view create my-urgent-tasks
```

**Interactive flow to test:**

1. **Welcome Screen**
   - Press Enter to continue
   - Ctrl+C to cancel

2. **Basic Info**
   - Enter description (optional)
   - Press Enter

3. **Field Selection**
   - ↑/↓ to navigate
   - Space to toggle selection
   - At least one field required
   - Enter to continue

4. **Field Ordering**
   - ↑/↓ to navigate
   - Ctrl+↑/↓ to move fields
   - Enter to continue

5. **Field Configuration**
   - ↑/↓ to navigate fields
   - Tab/Shift+Tab to switch fields
   - Space to toggle color
   - Enter to continue

6. **Display Options**
   - ↑/↓ to navigate
   - Space to toggle options
   - Enter to continue

7. **Confirmation**
   - Review settings
   - Y to save, N to cancel

**Verify:**
- State transitions work smoothly
- Validation prevents invalid configs
- Error messages display in red
- Keyboard shortcuts work
- View saves correctly

#### Using Custom Views

```bash
# List with custom view
gst Test -v my-urgent-tasks

# Verify custom configuration applied
```

### Error Handling Testing

#### Connection Errors

```bash
# Test with invalid config
gst --config /tmp/nonexistent.json Test
```

**Verify:** Clear error message about missing config

#### Authentication Errors

```bash
# Test with wrong credentials (edit config temporarily)
gst Test
```

**Verify:**
- Clear "Authentication failed" message
- Helpful guidance to check URL/credentials

#### Invalid Operations

```bash
# Try to update non-existent task
gst Test update "NonExistent" -p 1

# Try invalid status
gst Test update "SomeTask" -s INVALID
```

**Verify:**
- Clear error messages
- Suggestions for valid options

### Backend-Specific Tests

#### File Backend

```bash
# Test file backend operations
gst FileList add "Local task"
gst FileList
```

**Verify:**
- File backend status translation (TODO, DONE, PROCESSING, CANCELLED)
- Display names correct

#### Git Backend

```bash
# Test git backend (if configured)
gst GitTasks add "Git-tracked task"
gst GitTasks
```

**Verify:**
- Markdown file updates
- Git repository detection
- Task metadata in markdown format

## Test Coverage

### Measuring Coverage

```bash
# Overall coverage
go test ./... -cover

# Detailed coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Coverage Goals

- **Critical paths** (auth, parsing, storage): 80%+
- **Business logic** (task operations): 60%+
- **UI/formatting**: 40%+
- **CLI integration**: 20%+

### Why Some Areas Have Lower Coverage

- **CLI commands** (`cmd/gosynctasks`): Mostly integration code, requires end-to-end testing
- **Operations** (`internal/operations`): Glue code between packages
- **Formatters** (`internal/views/formatters`): Visual output, hard to test programmatically

## Writing Tests

### Test File Structure

```go
package mypackage

import "testing"

func TestMyFunction(t *testing.T) {
    // Arrange
    input := "test input"
    expected := "expected output"

    // Act
    result := MyFunction(input)

    // Assert
    if result != expected {
        t.Errorf("MyFunction(%q) = %q, want %q", input, result, expected)
    }
}
```

### Table-Driven Tests

Preferred pattern for multiple test cases:

```go
func TestStatusTranslation(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "TODO to NEEDS-ACTION",
            input:    "TODO",
            expected: "NEEDS-ACTION",
            wantErr:  false,
        },
        {
            name:     "invalid status",
            input:    "INVALID",
            expected: "",
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := TranslateStatus(tt.input)

            if (err != nil) != tt.wantErr {
                t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
            }
            if result != tt.expected {
                t.Errorf("got %q, want %q", result, tt.expected)
            }
        })
    }
}
```

### Testing with Mock Servers

For backend tests:

```go
func TestBackendMethod(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(mockResponse))
    }))
    defer server.Close()

    backend := NewBackend(server.URL)
    // Test backend methods...
}
```

## Best Practices

### Before Committing

```bash
# Run all tests
go test ./...

# Check for race conditions
go test ./... -race

# Verify coverage hasn't dropped
go test ./... -cover

# Format code
go fmt ./...
```

### When Adding Features

1. Write tests first (TDD)
2. Ensure tests fail initially
3. Implement feature
4. Verify tests pass
5. Check coverage

### When Fixing Bugs

1. Write test that reproduces the bug
2. Verify test fails
3. Fix the bug
4. Verify test passes
5. Ensure no regressions

## Troubleshooting

### Tests Fail Locally

```bash
# Clean test cache
go clean -testcache

# Run with verbose output
go test ./... -v

# Run specific failing test
go test ./backend -run TestFailingTest -v
```

### Flaky Tests

- Use fixed times instead of `time.Now()` in tests
- Avoid depending on external services
- Run with `-race` to detect race conditions
- Make tests independent (no shared state)

### Slow Tests

```bash
# Find slow tests
go test ./... -v 2>&1 | grep -E "PASS:.*[0-9]+\.[0-9]+s"

# Run with timeout
go test ./... -timeout 30s
```

## Continuous Integration

Tests run automatically on GitHub Actions (if configured).

### Local CI Simulation

```bash
# Run full CI test suite
go test ./... -race -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
go vet ./...
```

## Test Results Summary

### Unit Tests
- ✅ 33 tests in builder package (44.5% coverage)
- ✅ Backend tests with mock servers (65.2% coverage)
- ✅ Config validation tests (53.7% coverage)
- ✅ Views and formatters tests (50.4% coverage)

### Integration Tests
- ✅ Nextcloud CalDAV backend
- ✅ Git backend markdown parsing
- ✅ File backend status translation
- ✅ Multi-backend selection

### Manual Tests
- ✅ Add, update, complete, delete operations
- ✅ Priority sorting and coloring
- ✅ Partial match search with confirmation
- ✅ Status filtering
- ✅ Terminal width adaptation
- ✅ Interactive view builder
- ✅ Custom view usage

## Resources

- [Go Testing Documentation](https://pkg.go.dev/testing)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Testing Best Practices](https://github.com/golang/go/wiki/TestComments)
