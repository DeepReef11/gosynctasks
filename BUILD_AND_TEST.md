# Build and Test Instructions

Complete instructions for building, testing, and verifying the gosynctasks sync implementation.

## Prerequisites

### Required
- **Go 1.24.3+**: The project requires Go 1.24.3 or later
- **Git**: For version control
- **SQLite**: Bundled via modernc.org/sqlite (no separate installation needed)

### Optional
- **Docker**: For running Nextcloud test server
- **Make**: For using Makefile shortcuts (if available)

## Building

### Quick Build

```bash
# From project root
cd /home/user/gosynctasks

# Build the binary
go build -o gosynctasks ./cmd/gosynctasks

# Verify build
./gosynctasks --version
```

### Install to GOPATH

```bash
# Install to $GOPATH/bin (must be in your PATH)
go install ./cmd/gosynctasks

# Verify installation
gosynctasks --version
```

### Build for Specific Platform

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o gosynctasks-linux-amd64 ./cmd/gosynctasks

# macOS ARM64 (M1/M2)
GOOS=darwin GOARCH=arm64 go build -o gosynctasks-darwin-arm64 ./cmd/gosynctasks

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o gosynctasks-windows-amd64.exe ./cmd/gosynctasks
```

### Optimized Release Build

```bash
# Build with optimizations and stripped symbols
go build -ldflags="-s -w" -o gosynctasks ./cmd/gosynctasks

# Verify size reduction
ls -lh gosynctasks
```

## Testing

### Unit Tests

Run all unit tests:

```bash
# All packages
go test ./...

# With verbose output
go test -v ./...

# Specific package
go test ./backend
go test ./internal/config
```

### Coverage

Generate coverage reports:

```bash
# Run tests with coverage
go test -cover ./...

# Generate HTML coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Open coverage report
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

**Coverage Goals:**
- Backend: >85%
- Internal packages: >80%
- Overall: >80%

### Integration Tests

Run comprehensive end-to-end integration tests:

```bash
# All integration tests
go test ./backend -run Integration -v

# Specific integration test
go test ./backend -run TestBasicSyncWorkflow -v
go test ./backend -run TestOfflineModeWorkflow -v
go test ./backend -run TestConflictResolutionScenarios -v
go test ./backend -run TestLargeDatasetPerformance -v
go test ./backend -run TestErrorRecoveryWithRetry -v
```

**Integration Tests:**
1. `TestBasicSyncWorkflow`: End-to-end sync (remote→local→modify→remote)
2. `TestOfflineModeWorkflow`: Offline operations and queue management
3. `TestConflictResolutionScenarios`: All 4 conflict resolution strategies
4. `TestLargeDatasetPerformance`: 1000+ tasks in <30 seconds
5. `TestErrorRecoveryWithRetry`: Network error handling and retry logic
6. `TestConcurrentSyncOperations`: Race condition detection
7. `TestHierarchicalTaskSync`: Parent-child task sync

### Benchmark Tests

Run performance benchmarks:

```bash
# All benchmarks
go test -bench=. ./backend

# Specific benchmark
go test -bench=BenchmarkSyncPull ./backend
go test -bench=BenchmarkConflictResolution ./backend

# With memory allocations
go test -bench=. -benchmem ./backend

# Run for longer to get more accurate results
go test -bench=. -benchtime=10s ./backend
```

**Available Benchmarks:**
- `BenchmarkSyncPull`: Pull synchronization performance
- `BenchmarkSyncPush`: Push synchronization performance
- `BenchmarkConflictResolution`: Conflict resolution strategies
- `BenchmarkDatabaseOperations`: CRUD operations (Add/Get/Update/Delete)
- `BenchmarkSyncQueue`: Queue management operations
- `BenchmarkHierarchicalTaskSorting`: Task hierarchy sorting

### Test by Package

```bash
# Backend tests
go test -v ./backend

# Config tests
go test -v ./internal/config

# Operations tests
go test -v ./internal/operations

# Views tests
go test -v ./internal/views
```

### Parallel Test Execution

```bash
# Run tests in parallel (faster)
go test -p 4 ./...

# Run with race detector (slower but finds race conditions)
go test -race ./...
```

### Specific Test Patterns

```bash
# Run only SQLite backend tests
go test ./backend -run SQLite -v

# Run only sync manager tests
go test ./backend -run Sync -v

# Run only schema tests
go test ./backend -run Schema -v
```

## Linting

### Run golangci-lint (if installed)

```bash
# Install golangci-lint (if not installed)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run

# Run linter with auto-fix
golangci-lint run --fix

# Run specific linters
golangci-lint run --enable=gofmt,goimports,govet
```

### Format Code

```bash
# Format all Go files
go fmt ./...

# Or use gofmt directly
gofmt -w .

# Use goimports for better import handling
go install golang.org/x/tools/cmd/goimports@latest
goimports -w .
```

### Vet Code

```bash
# Run go vet (catches common mistakes)
go vet ./...
```

## Manual Testing

### Initial Setup

1. **Build the binary:**
```bash
go build -o gosynctasks ./cmd/gosynctasks
```

2. **Create test configuration:**
```bash
mkdir -p ./gosynctasks/config
cat > ./gosynctasks/config/config.json << 'EOF'
{
  "backends": {
    "local": {
      "type": "sqlite",
      "enabled": true,
      "db_path": "./gosynctasks/config/test.db"
    },
    "nextcloud": {
      "type": "nextcloud",
      "enabled": true,
      "url": "nextcloud://admin:admin@localhost:8080"
    }
  },
  "sync": {
    "enabled": true,
    "local_backend": "local",
    "remote_backend": "nextcloud",
    "conflict_resolution": "server_wins",
    "auto_sync": false
  },
  "default_backend": "local",
  "ui": "cli"
}
EOF
```

3. **Start Docker test server (optional):**
```bash
./scripts/start-test-server.sh
```

Wait 30-60 seconds for Nextcloud to initialize.

### Testing Sync Workflow

```bash
# Create alias for test instance
alias gst='./gosynctasks --config ./gosynctasks/config'

# Initial sync (pull from Nextcloud)
gst sync

# Check sync status
gst sync status

# Create task locally
gst list create "Test Tasks"
gst "Test Tasks" add "Task 1" -d "First test task" -p 1

# Sync to remote
gst sync

# Verify on Nextcloud web UI (http://localhost:8080)
# Login: admin/admin

# Modify task locally
gst "Test Tasks" update "Task 1" -s DONE

# Sync changes
gst sync

# Check queue
gst sync queue

# Full sync
gst sync --full
```

### Testing Offline Mode

```bash
# Stop Docker (simulate offline)
docker stop nextcloud-test

# Create tasks offline
gst "Test Tasks" add "Offline Task 1"
gst "Test Tasks" add "Offline Task 2"

# Check pending operations
gst sync queue

# Try to sync (should detect offline)
gst sync status

# Restart Docker
docker start nextcloud-test

# Sync when back online
gst sync

# Verify queue is cleared
gst sync queue
```

### Testing Conflict Resolution

```bash
# Create task
gst "Test Tasks" add "Conflict Test"

# Sync
gst sync

# Modify locally
gst "Test Tasks" update "Conflict Test" --summary "Modified Locally"

# Modify remotely (via Nextcloud web UI)
# Change summary to "Modified Remotely"

# Sync (conflict!)
gst sync

# Check result (server_wins by default)
gst "Test Tasks" get
```

### Performance Testing

```bash
# Create many tasks
for i in {1..100}; do
  gst "Test Tasks" add "Task $i"
done

# Time the sync
time gst sync
```

## Debugging

### Enable Verbose Logging

```bash
# Set environment variable (feature not yet implemented)
GOSYNCTASKS_DEBUG=1 ./gosynctasks sync
```

### Inspect SQLite Database

```bash
# Open database
sqlite3 ./gosynctasks/config/test.db

# List tables
.tables

# Check tasks
SELECT id, summary, status, priority FROM tasks LIMIT 10;

# Check sync metadata
SELECT task_uid, locally_modified, last_synced_at FROM sync_metadata LIMIT 10;

# Check sync queue
SELECT * FROM sync_queue;

# Check list sync metadata
SELECT * FROM list_sync_metadata;

# Exit
.quit
```

### Common Issues

**Issue: "Database file not found"**
```bash
# Check database path in config
cat ./gosynctasks/config/config.json | grep db_path

# Create directory if needed
mkdir -p ./gosynctasks/config
```

**Issue: "Sync failed: Authentication failed"**
```bash
# Verify Nextcloud is running
curl http://localhost:8080

# Check credentials in config
# Default test server: admin/admin
```

**Issue: "Foreign key constraint failed"**
```bash
# This is a bug - should not happen
# Hierarchical sorting should prevent this
# Please report with details
```

**Issue: "No module named modernc.org/sqlite"**
```bash
# Download dependencies
go mod download

# Tidy modules
go mod tidy

# Try build again
go build -o gosynctasks ./cmd/gosynctasks
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - uses: actions/setup-go@v4
        with:
          go-version: '1.24.3'

      - name: Download dependencies
        run: go mod download

      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out

      - name: Run benchmarks
        run: go test -bench=. -benchmem ./backend
```

## Test Coverage Report

After running tests with coverage:

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage by package
go tool cover -func=coverage.out

# Expected output:
# gosynctasks/backend/database.go:        InitDatabase            100.0%
# gosynctasks/backend/schema.go:          AllTableSchemas         100.0%
# gosynctasks/backend/sqliteBackend.go:   NewSQLiteBackend        95.2%
# gosynctasks/backend/syncManager.go:     Sync                    92.3%
# ...
# total:                                  (statements)            83.5%
```

**Target Coverage:**
- `backend/`: >85%
- `internal/config/`: >80%
- `internal/operations/`: >80%
- `internal/views/`: >80%
- **Overall: >80%**

## Environment Setup for Development

### Recommended Tools

```bash
# Install development tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install gotest.tools/gotestsum@latest

# Use gotestsum for better test output
gotestsum --format testname ./...
```

### Editor Configuration

**VS Code (.vscode/settings.json):**
```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "go.formatTool": "goimports",
  "go.testFlags": ["-v", "-race"],
  "go.coverOnSave": true
}
```

**GoLand/IntelliJ:**
- Enable "Go Modules" in Settings
- Set "Go Fmt" to run on save
- Enable race detector in test configurations

## Benchmarking

### Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. ./backend

# Run with CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./backend

# Run with memory profiling
go test -bench=. -memprofile=mem.prof ./backend

# Analyze CPU profile
go tool pprof cpu.prof
# Commands in pprof: top, list, web

# Analyze memory profile
go tool pprof mem.prof
```

### Benchmark Comparison

```bash
# Run benchmarks and save baseline
go test -bench=. ./backend | tee baseline.txt

# After optimizations, compare
go test -bench=. ./backend | tee optimized.txt

# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Compare results
benchstat baseline.txt optimized.txt
```

### Expected Benchmark Results

```
BenchmarkSyncPull/tasks=10-8              100   12345678 ns/op   1234567 B/op   12345 allocs/op
BenchmarkSyncPull/tasks=100-8              10   123456789 ns/op  12345678 B/op  123456 allocs/op
BenchmarkSyncPull/tasks=1000-8              1   1234567890 ns/op 123456789 B/op 1234567 allocs/op
```

## Docker Test Environment

### Starting Test Server

```bash
# Start Nextcloud in Docker
./scripts/start-test-server.sh

# Verify it's running
docker ps | grep nextcloud

# Check logs
docker logs nextcloud-test

# Access web UI
open http://localhost:8080  # macOS
xdg-open http://localhost:8080  # Linux
```

### Stopping Test Server

```bash
# Stop container
docker stop nextcloud-test

# Remove container
docker rm nextcloud-test

# Remove volume (clean slate)
docker volume rm nextcloud_data
```

## Release Build

### Creating a Release

```bash
# Version number
VERSION=1.0.0

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=$VERSION" -o gosynctasks-linux-amd64 ./cmd/gosynctasks
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version=$VERSION" -o gosynctasks-darwin-amd64 ./cmd/gosynctasks
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version=$VERSION" -o gosynctasks-darwin-arm64 ./cmd/gosynctasks
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.version=$VERSION" -o gosynctasks-windows-amd64.exe ./cmd/gosynctasks

# Create checksums
sha256sum gosynctasks-* > checksums.txt

# Create tarball
tar czf gosynctasks-$VERSION-linux-amd64.tar.gz gosynctasks-linux-amd64
tar czf gosynctasks-$VERSION-darwin-amd64.tar.gz gosynctasks-darwin-amd64
tar czf gosynctasks-$VERSION-darwin-arm64.tar.gz gosynctasks-darwin-arm64
zip gosynctasks-$VERSION-windows-amd64.zip gosynctasks-windows-amd64.exe
```

## Summary

**Quick Start:**
```bash
# Build
go build -o gosynctasks ./cmd/gosynctasks

# Test
go test ./...

# Run
./gosynctasks --help
```

**Full Verification:**
```bash
# 1. Build
go build -o gosynctasks ./cmd/gosynctasks

# 2. Unit tests
go test -v ./...

# 3. Integration tests
go test -v ./backend -run Integration

# 4. Benchmarks
go test -bench=. ./backend

# 5. Coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# 6. Lint
golangci-lint run

# 7. Manual test
./gosynctasks --help
```

## Support

For issues with building or testing:
- Check [README.md](README.md) for general usage
- See [CLAUDE.md](CLAUDE.md) for architecture details
- Read [SYNC_GUIDE.md](SYNC_GUIDE.md) for sync documentation
- Report bugs on GitHub Issues

---

**Environment Note:** This document assumes a development environment with network access. Some operations (like `go mod download`) may fail in restricted environments.
