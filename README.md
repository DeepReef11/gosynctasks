# gosynctasks

[![CI](https://github.com/DeepReef11/gosynctasks/actions/workflows/ci.yml/badge.svg)](https://github.com/DeepReef11/gosynctasks/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/DeepReef11/gosynctasks/branch/main/graph/badge.svg)](https://codecov.io/gh/DeepReef11/gosynctasks)
[![Go Report Card](https://goreportcard.com/badge/github.com/DeepReef11/gosynctasks)](https://goreportcard.com/report/github.com/DeepReef11/gosynctasks)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A flexible, multi-backend task synchronization tool written in Go. Manage your tasks across different storage backends including Nextcloud CalDAV, Git repositories with Markdown files, and local file storage.

## Features

- **Multi-Backend Support**: Work with multiple task storage backends simultaneously
- **Offline Sync**: Local SQLite cache with bidirectional synchronization to remote backends
- **Git/Markdown Backend**: Manage tasks directly in markdown files within git repositories
- **Nextcloud CalDAV**: Full CRUD support for Nextcloud Tasks
- **Auto-Detection**: Automatically detect and use the appropriate backend based on context
- **Flexible CLI**: Intuitive command-line interface with completion support
- **Hierarchical Tasks**: Support for subtasks and parent-child relationships
- **Custom Views**: Create custom task views with filtering, sorting, and custom formatting
- **Task Filtering**: Filter by status, priority, tags, and dates
- **Interactive Mode**: User-friendly interactive list and task selection
- **Conflict Resolution**: Four strategies for handling sync conflicts (server_wins, local_wins, merge, keep_both)
- **Offline Queue**: Work offline and automatically sync when reconnected

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/DeepReef11/gosynctasks.git
cd gosynctasks

# Build the binary
go build -o gosynctasks ./cmd/gosynctasks

# Install (optional)
go install ./cmd/gosynctasks
```

### Configuration

On first run, gosynctasks will help you create a configuration file at `~/.config/gosynctasks/config.json`.

#### Multi-Backend Configuration

```json
{
  "backends": {
    "nextcloud": {
      "type": "nextcloud",
      "enabled": true,
      "url": "nextcloud://username:password@your-server.com",
      "insecure_skip_verify": false
    },
    "git": {
      "type": "git",
      "enabled": true,
      "file": "TODO.md",
      "auto_detect": true,
      "auto_commit": false
    }
  },
  "default_backend": "nextcloud",
  "auto_detect_backend": true,
  "backend_priority": ["git", "nextcloud"],
  "ui": "cli"
}
```

#### Legacy Configuration (Auto-Migrated)

Old single-backend configurations are automatically migrated:

```json
{
  "connector": {
    "url": "nextcloud://username:password@your-server.com"
  },
  "ui": "cli"
}
```

The migration creates a backup at `config.json.backup` and converts to the new multi-backend format.

## Backends

### Git Backend

The Git backend allows you to manage tasks directly in markdown files within git repositories. Perfect for keeping tasks alongside your code!

**Setup:**

1. Create a `TODO.md` file in your git repository
2. Add the marker: `<!-- gosynctasks:enabled -->`
3. Configure the backend in your config

**Markdown Format:**

```markdown
<!-- gosynctasks:enabled -->

## Work Tasks
- [ ] Review PR #123 @priority:1 @due:2025-01-20
- [x] Deploy to staging @completed:2025-01-10
- [>] Write documentation @priority:2

## Personal Tasks
- [ ] Buy groceries @priority:5
```

**Status Mapping:**
- `[ ]` → TODO (NEEDS-ACTION)
- `[x]` → DONE (COMPLETED)
- `[>]` → PROCESSING (IN-PROCESS)
- `[-]` → CANCELLED

**Metadata Tags:**
- `@priority:N` - Priority 1-9 (1=highest)
- `@due:YYYY-MM-DD` - Due date
- `@created:YYYY-MM-DD` - Creation date
- `@completed:YYYY-MM-DD` - Completion date
- `@uid:string` - Unique identifier (auto-generated)

**Features:**
- ✅ Auto-detection in git repositories
- ✅ Preserves markdown formatting
- ✅ Full CRUD operations
- ✅ Optional auto-commit
- ✅ Unicode and emoji support
- ✅ Works with any markdown renderer (GitHub, GitLab, etc.)

### Nextcloud Backend

Full CalDAV support for Nextcloud Tasks with complete CRUD operations.

**Features:**
- ✅ List all task lists
- ✅ Get, create, update, delete tasks
- ✅ Status filtering
- ✅ Priority-based coloring
- ✅ Hierarchical tasks (subtasks)
- ✅ Due dates and metadata

**URL Format:**
```
nextcloud://username:password@server.com
nextcloud://username:password@server.com:8080/nextcloud
```

### SQLite Backend (Offline Sync)

Local SQLite database for offline synchronization with remote backends.

**Features:**
- ✅ Offline mode - work without network connectivity
- ✅ Bidirectional sync with Nextcloud
- ✅ **Auto-sync with background daemon** - instant operations, sync happens after
- ✅ Conflict resolution (4 strategies)
- ✅ Operation queuing and retry logic
- ✅ Efficient sync with CTags/ETags
- ✅ Supports 1000+ tasks with <30s sync time

**Configuration:**
```json
{
  "backends": {
    "sqlite": {
      "type": "sqlite",
      "enabled": true,
      "db_path": ""  // Empty = use XDG default (~/.local/share/gosynctasks/tasks.db)
    },
    "nextcloud": {
      "type": "nextcloud",
      "enabled": true,
      "url": "nextcloud://user:pass@server.com"
    }
  },
  "sync": {
    "enabled": true,
    "local_backend": "sqlite",
    "remote_backend": "nextcloud",
    "conflict_resolution": "server_wins",
    "auto_sync": true,       // Enable background daemon sync
    "sync_interval": 5       // Minutes before data considered stale
  },
  "backend_priority": ["nextcloud"]  // Local backend auto-selected when sync enabled
}
```

**Note:** When `sync.enabled = true`, the CLI automatically uses `sqlite` for all operations and the `backend_priority` only applies when sync is disabled.

### File Backend

Local file-based storage (work in progress).

## Synchronization

gosynctasks includes a powerful offline synchronization system for working with remote backends (Nextcloud) while maintaining a local SQLite cache.

### Quick Start with Sync

```bash
# Configure sync in config.json (see SQLite Backend above)

# Perform initial sync
gosynctasks sync

# Check sync status
gosynctasks sync status

# Work offline - changes are queued
gosynctasks Work add "Task created offline"
gosynctasks Work update "Some task" -s DONE

# When back online, sync changes
gosynctasks sync
```

### Sync Features

**Offline Mode:**
- Work seamlessly without network connectivity
- Changes automatically queued for sync
- Visual indicators when offline

**Bidirectional Sync:**
- Pull: Download changes from remote → local
- Push: Upload local changes → remote
- CTag-based change detection (efficient)

**Conflict Resolution:**
Choose how to handle conflicts when the same task is modified both locally and remotely:

- `server_wins` (default): Remote changes override local (safest)
- `local_wins`: Local changes override remote
- `merge`: Intelligent merge of non-conflicting fields
- `keep_both`: Keep both versions (creates copy)

**Error Handling:**
- Automatic retry with exponential backoff
- Failed operations tracked in queue
- Manual retry and clear options

### Sync Commands

```bash
# Synchronize with remote
gosynctasks sync

# Force full re-sync (ignore CTags)
gosynctasks sync --full

# Check sync status
gosynctasks sync status

# View pending operations
gosynctasks sync queue

# Clear failed operations
gosynctasks sync queue clear --failed

# Retry failed operations
gosynctasks sync queue retry
```

### Sync Configuration

```json
{
  "sync": {
    "enabled": true,
    "local_backend": "local",
    "remote_backend": "nextcloud",
    "conflict_resolution": "server_wins",
    "auto_sync": false,
    "sync_interval": 300
  }
}
```

**Options:**
- `enabled`: Enable/disable sync
- `local_backend`: Name of SQLite backend
- `remote_backend`: Name of remote backend (Nextcloud)
- `conflict_resolution`: Strategy for conflicts (server_wins, local_wins, merge, keep_both)
- `auto_sync`: **NEW!** Enable background daemon sync - operations return instantly
- `sync_interval`: Minutes before data considered stale (for pull operations)

### Auto-Sync Behavior

When `auto_sync: true`:
- ✅ **Instant operations** - CLI returns immediately after writing to SQLite
- ✅ **Background daemon** - Detached process runs `gosynctasks sync --quiet`
- ✅ **Queue-based** - Operations persisted in `sync_queue` table
- ✅ **Reliable** - Failed syncs retry on next operation
- ✅ **Offline-friendly** - Queue builds up, syncs when online

**For detailed sync documentation, see [SYNC_GUIDE.md](SYNC_GUIDE.md)**

## Usage

### Backend Selection

```bash
# List configured backends
gosynctasks --list-backends

# Detect backend in current directory
gosynctasks --detect-backend

# Use specific backend
gosynctasks --backend git MyList get
gosynctasks --backend nextcloud WorkTasks add "New task"
```

**Selection Priority:**
1. CLI flag: `--backend <name>`
2. Auto-detection (if `auto_detect_backend: true`)
3. Default backend from config
4. First enabled backend

### Task Management

```bash
# List tasks
gosynctasks                              # Interactive list selection
gosynctasks MyList                       # Show tasks from "MyList"
gosynctasks MyList get                   # Explicit get action

# Filter tasks
gosynctasks MyList -s TODO,DONE          # Filter by status
gosynctasks MyList -s T,D,P              # Using abbreviations

# Add tasks
gosynctasks MyList add "Task summary"
gosynctasks MyList add "Task" -d "Description" -p 1 -S done

# Add subtasks
gosynctasks MyList add "Subtask" -P "Parent Task"
gosynctasks MyList add "parent/child/grandchild"  # Auto-creates hierarchy

# Update tasks
gosynctasks MyList update "task name" -s DONE
gosynctasks MyList update "partial" -p 5

# Complete tasks (shortcut)
gosynctasks MyList complete "task name"
```

### Custom Views

```bash
# List all views
gosynctasks view list

# Show view configuration
gosynctasks view show myview

# Create new view
gosynctasks view create myview

# Use view
gosynctasks MyList -v myview
gosynctasks MyList -v all
```

### List Management

```bash
# Create list
gosynctasks list create "New List"
gosynctasks list create "Project X" -d "Description" --color "#ff0000"

# List info
gosynctasks list info MyList
gosynctasks list                         # List all lists

# Rename list
gosynctasks list rename "Old Name" "New Name"

# Delete list
gosynctasks list delete "List Name"

# Trash management
gosynctasks list trash                   # Show deleted lists
gosynctasks list trash restore "List"    # Restore from trash
gosynctasks list trash empty "List"      # Permanently delete
```

### Shell Completion

```bash
# Zsh
eval "$(gosynctasks completion zsh)"

# Bash
eval "$(gosynctasks completion bash)"

# Fish
gosynctasks completion fish | source

# PowerShell
gosynctasks completion powershell | Out-String | Invoke-Expression
```

## Development

### Quick Commands (Makefile)

```bash
# Show all available commands
make help

# Build
make build                       # Single binary
make build-all                   # All platforms

# Testing
make test                        # Unit tests
make test-integration            # Mock backend integration tests
make test-integration-nextcloud  # Real Nextcloud sync tests
make test-all                    # All tests
make test-coverage               # Generate coverage report

# Code Quality
make lint                        # Run linter
make fmt                         # Format code
make vet                         # Run go vet
make security                    # Security scan

# Docker Test Server
make docker-up                   # Start Nextcloud server
make docker-down                 # Stop server
make docker-logs                 # View logs
make docker-status               # Check status

# CI/CD
make ci                          # Run full CI suite locally
make clean                       # Clean build artifacts
make clean-all                   # Clean everything including Docker
```

### Building from Source

```bash
# Clone repository
git clone https://github.com/DeepReef11/gosynctasks.git
cd gosynctasks

# Build binary
make build
# or
go build -o gosynctasks ./cmd/gosynctasks

# Install to $GOPATH/bin
make install
# or
go install ./cmd/gosynctasks

# Build for all platforms
make build-all
```

### Testing

gosynctasks has comprehensive test coverage with three test types:

#### 1. Unit Tests
```bash
# Run all unit tests
make test
# or
go test ./...

# With coverage
make test-coverage
# or
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Specific package
go test ./backend -v
go test ./internal/config -v
```

#### 2. Mock Backend Integration Tests
```bash
# Fast integration tests (no external dependencies)
make test-integration
# or
go test -v -timeout 10m \
  ./backend/integration_test.go \
  ./backend/mockBackend.go \
  ./backend/syncManager.go \
  ./backend/taskManager.go
```

#### 3. Nextcloud Sync Integration Tests
```bash
# Real CalDAV server tests
make test-integration-nextcloud
# or manually:
make docker-up                    # Start Nextcloud
./scripts/wait-for-nextcloud.sh  # Wait for ready
go test -v -timeout 15m -tags=integration ./backend/sync_integration_test.go
make docker-down                  # Cleanup
```

**What's tested:**
- ✅ Push sync (local → Nextcloud)
- ✅ Pull sync (Nextcloud → local)
- ✅ Bidirectional sync
- ✅ Conflict resolution (ServerWins, LocalWins)
- ✅ Task deletion sync
- ✅ Real CalDAV protocol
- ✅ iCalendar format validation

See [TESTING.md](TESTING.md) for comprehensive testing guide including:
- Manual feature testing workflows
- Git backend testing
- Multi-backend testing
- Docker test environment setup
- CI/CD pipeline details

### Project Structure

```
gosynctasks/
├── cmd/gosynctasks/        # CLI application
│   └── main.go             # Entry point and command definitions
├── backend/                # Backend implementations
│   ├── taskManager.go      # TaskManager interface
│   ├── selector.go         # Backend selection logic
│   ├── nextcloudBackend.go # Nextcloud CalDAV backend
│   ├── gitBackend.go       # Git/Markdown backend
│   ├── markdownParser.go   # Markdown parsing
│   └── markdownWriter.go   # Markdown writing
├── internal/
│   ├── config/             # Configuration management
│   ├── operations/         # Task operations
│   └── views/              # Custom views system
├── scripts/                # Utility scripts
│   └── start-test-server.sh # Docker test environment
└── gosynctasks/
    └── config/             # Test configurations
```

## Migration Guide

### From Single Backend to Multi-Backend

Your existing configuration will be automatically migrated when you first run the new version:

**Before (old format):**
```json
{
  "connector": {
    "url": "nextcloud://user:pass@server.com"
  },
  "ui": "cli"
}
```

**After (new format):**
```json
{
  "backends": {
    "nextcloud": {
      "type": "nextcloud",
      "enabled": true,
      "url": "nextcloud://user:pass@server.com",
      "insecure_skip_verify": false
    }
  },
  "default_backend": "nextcloud",
  "auto_detect_backend": false,
  "backend_priority": ["nextcloud"],
  "ui": "cli"
}
```

The old config is backed up to `config.json.backup`. You can now add additional backends:

```json
{
  "backends": {
    "nextcloud": {
      "type": "nextcloud",
      "enabled": true,
      "url": "nextcloud://user:pass@server.com"
    },
    "git": {
      "type": "git",
      "enabled": true,
      "file": "TODO.md",
      "auto_detect": true
    }
  },
  "default_backend": "nextcloud",
  "auto_detect_backend": true,
  "backend_priority": ["git", "nextcloud"],
  "ui": "cli"
}
```

### Adding Git Backend to Existing Config

1. Add git backend to your config:
```json
{
  "backends": {
    "git": {
      "type": "git",
      "enabled": true,
      "file": "TODO.md",
      "auto_detect": true,
      "auto_commit": false
    }
  }
}
```

2. Enable auto-detection:
```json
{
  "auto_detect_backend": true,
  "backend_priority": ["git", "nextcloud"]
}
```

3. Create `TODO.md` in your git repository:
```markdown
<!-- gosynctasks:enabled -->

## Tasks
- [ ] Your first task
```

4. Now when you run `gosynctasks` from within your git repository, it will automatically use the git backend!

## Configuration Examples

### Example 1: Git-First with Nextcloud Fallback

```json
{
  "backends": {
    "git": {
      "type": "git",
      "enabled": true,
      "file": "TODO.md",
      "auto_detect": true,
      "auto_commit": false
    },
    "nextcloud": {
      "type": "nextcloud",
      "enabled": true,
      "url": "nextcloud://user:pass@server.com"
    }
  },
  "default_backend": "nextcloud",
  "auto_detect_backend": true,
  "backend_priority": ["git", "nextcloud"],
  "ui": "cli"
}
```

**Behavior:** When in a git repo with TODO.md, uses git backend. Otherwise uses Nextcloud.

### Example 2: Multiple Git Backends

```json
{
  "backends": {
    "work": {
      "type": "git",
      "enabled": true,
      "file": "/home/user/work/TODO.md",
      "auto_detect": false
    },
    "personal": {
      "type": "git",
      "enabled": true,
      "file": "/home/user/personal/TODO.md",
      "auto_detect": false
    }
  },
  "default_backend": "work",
  "auto_detect_backend": false,
  "ui": "cli"
}
```

**Usage:**
```bash
gosynctasks --backend work MyTasks get
gosynctasks --backend personal Shopping add "Milk"
```

### Example 3: Nextcloud Only

```json
{
  "backends": {
    "nextcloud": {
      "type": "nextcloud",
      "enabled": true,
      "url": "nextcloud://user:pass@server.com:443/nextcloud",
      "insecure_skip_verify": false
    }
  },
  "default_backend": "nextcloud",
  "auto_detect_backend": false,
  "ui": "cli"
}
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](.github/CONTRIBUTING.md) for detailed guidelines.

### Development Workflow

1. **Fork and clone** the repository
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Make your changes** with clear commit messages
4. **Run tests locally**: `make ci`
5. **Push and create a PR**

### CI/CD Pipeline

Every push and pull request triggers automated checks:

| Job | Description | Time |
|-----|-------------|------|
| **Lint** | Code quality (golangci-lint) | ~2 min |
| **Test** | Unit tests with coverage | ~2 min |
| **Integration** | Mock + Nextcloud sync tests | ~8 min |
| **Security** | Vulnerability scan (govulncheck) | ~1 min |
| **Build** | Multi-platform binaries | ~4 min |

**Branch Protection:**
- ✅ All CI checks must pass
- ✅ Code review required
- ✅ Branch must be up to date

### Running CI Locally

```bash
# Full CI suite
make ci

# Individual checks
make lint                        # Code quality
make test                        # Unit tests
make test-integration            # Integration tests
make test-integration-nextcloud  # Nextcloud tests
make security                    # Security scan
make build                       # Build check
```

### Testing Requirements

Before submitting a PR:

```bash
# Run all tests
make test-all

# Check for race conditions
go test -race ./...

# Ensure coverage doesn't drop
make test-coverage

# Format code
make fmt

# Run linter
make lint
```

### Code Style

- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use meaningful variable names
- Add comments for exported functions
- Write tests for new features
- Update documentation

### Commit Convention

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new backend support
fix: resolve sync conflict bug
docs: update README with examples
test: add integration tests for sync
refactor: simplify backend selection
chore: update dependencies
```

### Creating a Release

Releases are automated via GitHub Actions:

```bash
# Create and push a version tag
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3

# GitHub Actions will:
# - Run all CI checks
# - Build multi-platform binaries
# - Create GitHub release
# - Attach binaries as artifacts
```

### Getting Help

- 📖 Read [CLAUDE.md](CLAUDE.md) for architecture details
- 🧪 See [TESTING.md](TESTING.md) for testing guide
- 🔄 Check [SYNC_GUIDE.md](SYNC_GUIDE.md) for sync documentation
- 💬 Open an issue for questions
- 🐛 Report bugs with reproduction steps

## Roadmap

### Completed ✅
- Multi-backend support (Nextcloud, Git, SQLite)
- Git/Markdown backend with auto-detection
- Backend auto-detection and priority selection
- Automatic config migration
- **Comprehensive CI/CD pipeline** with automated releases
- **Nextcloud sync integration tests** with real CalDAV server
- SQLite sync layer with offline mode
- Conflict resolution (4 strategies: server_wins, local_wins, merge, keep_both)
- Bidirectional synchronization with etag handling
- **Auto-sync with background daemon** - instant operations
- Path expansion (`$HOME`, `~`) in config files
- Custom views with plugin formatters
- Hierarchical tasks (subtasks)
- Interactive list and task selection

### In Progress 🚧
- File backend implementation
- Enhanced documentation

### Planned 📋
- GitHub/GitLab Issues backends
- Jira integration
- Trello/Notion backends
- Cross-backend task migration
- Mobile companion app
- Web UI dashboard
- Sync hooks and plugins
- Real-time sync with webhooks

## Support

For issues, questions, or contributions:
- Open an issue on GitHub
- See [CLAUDE.md](CLAUDE.md) for development guidelines
- Check [TESTING.md](TESTING.md) for testing procedures
- Read [SYNC_GUIDE.md](SYNC_GUIDE.md) for detailed synchronization documentation

---

Built with ❤️ using Go and the Cobra CLI framework.
