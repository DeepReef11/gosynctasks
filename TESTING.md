# Testing Guide

Quick reference for testing gosynctasks with Docker test environment, unit tests, and integration tests.

## Quick Start

```bash
# Run all unit tests
go test ./...

# Run with coverage
go test ./... -cover

# Run integration tests (requires Docker)
make test-integration-nextcloud

# Full CI suite
make ci
```

## Docker Test Environment

### Start Nextcloud Test Server

```bash
# Start server (auto-installs Tasks app)
./scripts/start-test-server.sh

# Stop server
docker compose down

# Stop and delete data
docker compose down -v
```

Server runs at: `http://localhost:8080` 
Credentials: `admin` / `admin123`

### Test Configuration

Pre-configured test config at `./gosynctasks/config/config.yaml`:

```yaml
backends:
  nextcloud:
    type: nextcloud
    enabled: true
    url: nextcloud://admin:admin123@localhost:8080/
    allow_http: true
    suppress_http_warning: true
    insecure_skip_verify: true

ui: cli
```

### Using Test Config

```bash
# Build binary
go build -o gosynctasks ./cmd/gosynctasks

# Use test config
./gosynctasks --config ./gosynctasks/config MyList add "Test task"

# Or create alias
alias gosynctest="./gosynctasks --config ./gosynctasks/config"
gosynctest MyList get
```

## Unit Tests

### Run Tests

```bash
# All tests
go test ./...

# Specific package
go test ./backend
go test ./internal/views

# Specific test
go test ./backend -run TestParseVTODO

# With race detection
go test ./... -race

# With verbose output
go test ./... -v
```

### Coverage

```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out

# View in browser
go tool cover -html=coverage.out

# Show coverage by function
go tool cover -func=coverage.out
```

## Integration Tests

### Makefile Commands

```bash
# Unit tests only
make test

# Mock integration tests (fast, no Docker)
make test-integration

# Nextcloud sync tests (requires Docker)
make test-integration-nextcloud

# All tests
make test-all

# Lint
make lint

# Security scan
make security

# Full CI suite
make ci
```

### Nextcloud Sync Integration Tests

Tests real CalDAV synchronization with Docker Nextcloud server.

**Location:** `backend/sync_integration_test.go`

**Tests:**
- Push sync (local → Nextcloud)
- Pull sync (Nextcloud → local)
- Bidirectional sync
- Conflict resolution (ServerWins, LocalWins)
- Task deletion

**Run manually:**
```bash
# Start Nextcloud
./scripts/start-test-server.sh

# Wait for ready
./scripts/wait-for-nextcloud.sh

# Run tests
NEXTCLOUD_TEST_URL="nextcloud://admin:admin123@localhost:8080/" \
  go test -v -timeout 15m -tags=integration ./backend -run "TestSync"

# Cleanup
docker compose down -v
```

**Environment Variables:**
- `NEXTCLOUD_TEST_URL` - Custom Nextcloud URL
- `SKIP_INTEGRATION=1` - Skip integration tests

## Manual Testing

### Essential Test Workflow

```bash
# Use test config
alias gosynctest="./gosynctasks --config ./gosynctasks/config"

# Create test list
gosynctest list create Test

# Add tasks
gosynctest Test add "Task 1" -p 1
gosynctest Test add "Task 2" -d "Description" -p 5

# List tasks
gosynctest Test
gosynctest Test -v all

# Update task
gosynctest Test update "Task 1" -s DONE

# Complete task
gosynctest Test complete "Task 2"

# Delete task
gosynctest Test delete "Task 1"
```

### Test Subtasks

```bash
# Add parent
gosynctest Test add "Feature X" -p 1

# Add subtask
gosynctest Test add "Write code" -P "Feature X"

# Path-based creation (auto-creates hierarchy)
gosynctest Test add "Epic/Story/Task"

# Literal mode (disable path parsing)
gosynctest Test add -l "URL: http://example.com"
```

### Test Custom Views

```bash
# Create view interactively
gosynctest view create my-view

# List with view
gosynctest Test -v my-view

# List views
gosynctest view list
```

### Test Git Backend

```bash
# Create test repo
mkdir -p /tmp/test-git
cd /tmp/test-git
git init

# Create TODO.md
cat > TODO.md << 'EOF'
<!-- gosynctasks:enabled -->

## Tasks
EOF

git add TODO.md && git commit -m "Init"

# Configure backend
# Edit ~/.config/gosynctasks/config.yaml:
backends:
  git:
    type: git
    enabled: true
    file: TODO.md
    auto_detect: true

auto_detect_backend: true
backend_priority: [git]

# Test
gosynctasks --detect-backend  # Should output: git
gosynctasks Tasks add "Test task"
cat TODO.md  # Verify markdown format

# Cleanup
rm -rf /tmp/test-git
```

## Troubleshooting

### Docker Issues

```bash
# Port 8080 in use
lsof -i :8080

# Check container status
docker compose ps

# View logs
docker compose logs -f nextcloud

# Reset database
docker compose down -v
./scripts/start-test-server.sh
```

### Test Issues

```bash
# Clean test cache
go clean -testcache

# Run with verbose output
go test ./... -v

# Run specific failing test
go test ./backend -run TestFailingTest -v

# Check for race conditions
go test ./... -race
```

### Connection Errors

```bash
# Verify server is running
curl http://localhost:8080/status.php

# Check Docker network
docker network ls
docker network inspect gosynctasks_default

# Restart containers
docker compose restart
```

## CI/CD Pipeline

GitHub Actions runs these jobs on every push:

| Job | Description | Time |
|-----|-------------|------|
| Lint | golangci-lint | ~2min |
| Test | Unit tests with coverage | ~2min |
| Integration | Mock + Nextcloud sync | ~8min |
| Security | govulncheck | ~1min |
| Build | Multi-platform binaries | ~4min |

### Run CI Locally

```bash
# Full CI suite
make ci

# Individual checks
make lint
make test
make test-integration-nextcloud
make security
make build
```

## Test Coverage Goals

- **Critical paths** (auth, parsing, storage): 80%+
- **Business logic** (task operations): 60%+
- **UI/formatting**: 40%+
- **CLI integration**: 20%+

## Writing Tests

### Table-Driven Tests

```go
func TestStatusTranslation(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"TODO to NEEDS-ACTION", "TODO", "NEEDS-ACTION"},
        {"DONE to COMPLETED", "DONE", "COMPLETED"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := TranslateStatus(tt.input)
            if result != tt.expected {
                t.Errorf("got %q, want %q", result, tt.expected)
            }
        })
    }
}
```

### Mock HTTP Servers

```go
func TestBackend(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(mockResponse))
    }))
    defer server.Close()

    backend := NewBackend(server.URL)
    // Test methods...
}
```

## Resources

- [Go Testing Documentation](https://pkg.go.dev/testing)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [README.md](README.md) - General usage
- [SYNC_GUIDE.md](SYNC_GUIDE.md) - Sync documentation
