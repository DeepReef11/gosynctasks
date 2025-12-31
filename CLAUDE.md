# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project Overview

gosynctasks is a Go-based task synchronization CLI that interfaces with multiple backends (Nextcloud CalDAV, SQLite) to manage tasks with offline sync support. Built with Cobra framework.

**Testing Requirements:**
- After code changes: rebuild and provide test commands
- Docker test server required for integration tests: `./scripts/start-test-server.sh`
- Test config: `./gosynctasks/config/config.yaml` (pre-configured for localhost:8080)

## Quick Reference

### Build & Test
```bash
go build -o gosynctasks ./cmd/gosynctasks     # Build
go test ./...                                  # Run all tests
./scripts/start-test-server.sh                # Start Docker test server

# Test function (use with Docker test server)
gst() { go run ./cmd/gosynctasks --config ./gosynctasks/config "$@"; }
```

### Common Commands
```bash
gosynctasks                             # Interactive list selection
gosynctasks MyList                      # Show tasks
gosynctasks MyList add "Task"           # Add task (aliases: a)
gosynctasks MyList add "Task" -P "Parent"  # Add subtask
gosynctasks MyList add "parent/child"   # Auto-create hierarchy
gosynctasks MyList update "task" -s DONE   # Update task (aliases: u)
gosynctasks MyList complete "task"      # Complete task (aliases: c)
gosynctasks MyList -v all               # Show all metadata

# List management
gosynctasks list create "Name"          # Create list
gosynctasks list trash                  # View deleted lists
gosynctasks list trash restore "Name"   # Restore deleted list

# Custom views
gosynctasks view list                   # List views
gosynctasks view create myview          # Create view

# Sync (SQLite + Nextcloud)
gosynctasks sync                        # Bidirectional sync
gosynctasks sync status                 # Show sync status
gosynctasks sync queue                  # View pending operations

# Credential Management
gosynctasks credentials set <backend> <user> --prompt  # Store in keyring (secure)
gosynctasks credentials get <backend> <user>           # Check credential source
gosynctasks credentials delete <backend> <user>        # Remove from keyring
```

### Credential Management

**Priority Order:** Keyring > Environment Variables > Config URL

**Credential Storage Options:**

1. **System Keyring (RECOMMENDED)**
   - Most secure - credentials stored in OS keyring
   - macOS: Keychain, Windows: Credential Manager, Linux: Secret Service
   ```bash
   gosynctasks credentials set nextcloud myuser --prompt
   ```
   - Config example:
   ```yaml
   nextcloud:
     type: nextcloud
     enabled: true
     host: "nextcloud.example.com"
     username: "myuser"  # Password retrieved from keyring
   ```

2. **Environment Variables (Good for CI/CD)**
   - Set credentials via environment
   ```bash
   export GOSYNCTASKS_NEXTCLOUD_USERNAME=myuser
   export GOSYNCTASKS_NEXTCLOUD_PASSWORD=secret
   export GOSYNCTASKS_NEXTCLOUD_HOST=nextcloud.example.com
   ```
   - Config can be minimal:
   ```yaml
   nextcloud:
     type: nextcloud
     enabled: true
     username: "myuser"  # Optional hint
   ```

3. **Config URL (LEGACY - Not Recommended)**
   - Plain text credentials in config file
   ```yaml
   nextcloud:
     type: nextcloud
     enabled: true
     url: "nextcloud://username:password@nextcloud.example.com"
   ```
   - Backward compatible but less secure

**Migration Path:**
```bash
# 1. Store credentials in keyring
gosynctasks credentials set nextcloud myuser --prompt

# 2. Update config to use keyring
# Change from: url: "nextcloud://user:pass@host"
# Change to:
#   host: "nextcloud.example.com"
#   username: "myuser"

# 3. Verify it works
gosynctasks credentials get nextcloud myuser
gosynctasks nextcloud  # Test connection
```

### Status & Abbreviations
- Internal: TODO, DONE, PROCESSING, CANCELLED
- CalDAV: NEEDS-ACTION, COMPLETED, IN-PROCESS, CANCELLED
- CLI abbreviations: T/D/P/C

## Architecture Overview

### Backend System
**Pluggable backend pattern** - All backends implement `backend.TaskManager` interface:
- Required methods: `GetTaskLists()`, `GetTasks()`, `FindTasksBySummary()`, `AddTask()`, `UpdateTask()`, `SortTasks()`, `GetPriorityColor()`
- Backend selection via URL scheme in config (`nextcloud://`, `sqlite://`, `file://`)
- Factory: `backend.ConnectorConfig` creates TaskManager from URL
- **Selection priority**: Explicit flag → Sync local backend (if enabled) → Auto-detect → Default → First enabled

**Implementations:**
- **NextcloudBackend**: CalDAV protocol (PROPFIND/REPORT/PUT), iCalendar VTODO parsing
- **SQLiteBackend**: Local database with sync support (see Sync section)
- **FileBackend**: Placeholder (not functional)

### Key Components

**Configuration** (`internal/config/config.go`):
- Location: `$XDG_CONFIG_HOME/gosynctasks/config.yaml`
- Singleton pattern with `sync.Once`
- First-run setup from embedded sample

**CLI Structure** (`cmd/gosynctasks/`):
- Framework: Cobra
- Argument order: `gosynctasks [list-name] [action] [task-summary]`
- Actions: `get` (default), `add`, `update`, `complete`
- Task list cache: `$XDG_CACHE_HOME/gosynctasks/lists.json`
- Interactive list selection with dynamic terminal width detection

**Data Models** (`backend/taskManager.go`):
- **Task**: Follows iCalendar VTODO (UID, Summary, Description, Status, Priority 0-9, Timestamps, Categories, ParentUID for subtasks)
- **TaskList**: Represents calendar/list (ID, Name, Description, CTags, Color)

**Subtask Support** (`internal/operations/subtasks.go`):
- Hierarchical tasks via `-P "Parent"` or `-P "Parent/Child"` flags
- Path-based creation: `add "parent/child/grandchild"` auto-creates hierarchy
- Tree display with box-drawing characters (├─, └─, │)
- Path resolution for nested task references

## Custom Views System

**Storage**: YAML files in `~/.config/gosynctasks/views/`

**Features:**
- Custom field selection, ordering, formatting
- Built-in views: `default`, `all`
- Filters: status, priority, tags, dates
- Sorting: by any field (asc/desc)
- Hierarchical task display support

**Available Fields:**
- `status`, `summary`, `description`, `priority`
- Dates: `due_date`, `start_date`, `created`, `modified`, `completed`
- `tags`, `uid`, `parent`

**Key Modules:**
- `internal/views/types.go`: View data structures
- `internal/views/renderer.go`: Rendering engine with formatters
- `internal/views/filter.go`: Filtering and sorting logic
- `internal/views/builder/`: Interactive TUI builder

**Plugin Formatters:**
- External scripts for custom field formatting
- Language-agnostic: supports bash, python, ruby, etc.
- Input: JSON task data via stdin
- Output: formatted string via stdout
- Security: timeout enforcement (max 5s), error handling
- Example plugins: `examples/view-plugins/`

**Plugin Configuration (YAML):**
```yaml
fields:
  - name: status
    plugin:
      command: "/path/to/script.sh"
      args: ["--style", "emoji"]
      timeout: 1000  # milliseconds
      env:
        CUSTOM_VAR: "value"
```

**Plugin Input Format:**
```json
{
  "uid": "task-123",
  "summary": "Task name",
  "status": "TODO",
  "priority": 1,
  "due_date": "2025-01-15T00:00:00Z",
  "categories": ["tag1", "tag2"]
}
```

**Key Files:**
- `internal/views/formatters/plugin.go`: Plugin execution
- `internal/views/formatters/plugin_test.go`: Plugin tests
- `examples/view-plugins/`: Example scripts

## Common Patterns

**Terminal Width Detection:**
- Uses `golang.org/x/term` for cross-platform detection
- Default fallback: 80 chars, constraints: 40-100
- Used for borders and dynamic formatting

**Task Search:**
- Intelligent matching: exact → single partial (confirm) → multiple (select)
- `findTaskBySummary()` handles selection flow
- User can cancel at any prompt

**Status Translation:**
- Always use `StatusStringTranslateToStandardStatus()` for CalDAV
- Use `StatusStringTranslateToAppStatus()` for display

**Adding New Backend:**
1. Implement `TaskManager` interface in `backend/`
2. Add URL scheme case to `ConnectorConfig.TaskManager()`

## Testing

**Key Test Files:**
- `backend/*_test.go`: Backend-specific tests
- `internal/views/*_test.go`: View system tests
- See [TESTING.md](TESTING.md) for manual testing workflow

**Run:** `go test ./...` or `go test ./backend -v`

## SQLite Sync System

**Bidirectional synchronization** with offline support (local SQLite ↔ remote Nextcloud).

### Architecture
```
CLI → SQLiteBackend (CRUD, queueing) → SyncManager (pull/push) ↔ Remote Backend
```

### Database Schema (`backend/schema.go`)
**Tables:**
- `tasks`: Main data (VTODO format, with `parent_uid` for hierarchy)
- `sync_metadata`: Per-task sync state (etags, flags, timestamps)
- `list_sync_metadata`: Per-list sync state (ctags, tokens)
- `sync_queue`: Pending operations (create/update/delete) with retry tracking
- `schema_version`: Migration tracking

**Indexes**: Optimized for `list_id`, `status`, `due_date`, `parent_uid`, `priority`, sync flags

### Key Components

**SQLiteBackend** (`backend/sqliteBackend.go`):
- Full CRUD with transactional operations
- Sync methods: `MarkLocallyModified()`, `MarkLocallyDeleted()`, `ClearSyncFlags()`, `UpdateSyncMetadata()`
- Automatic UID generation, status translation

**SyncManager** (`backend/syncManager.go`):
- **Pull**: Fetch remote → update local → detect conflicts
- **Push**: Process sync queue → execute on remote → retry on failure
- **Conflict strategies**: `server_wins` (default), `local_wins`, `merge`, `keep_both`
- **Hierarchical sorting**: Parents before children (prevents FK violations)
- **Error handling**: Exponential backoff, max 5 retries

**Database** (`backend/database.go`):
- XDG-compliant path: `$XDG_DATA_HOME/gosynctasks/tasks.db`
- Methods: `InitDatabase()`, `GetStats()`, `Vacuum()`

### Configuration Example
```yaml
backends:
  sqlite:
    type: sqlite
    enabled: true
  nextcloud:
    type: nextcloud
    enabled: true
    url: nextcloud://user:pass@host

sync:
  enabled: true
  local_backend: sqlite
  remote_backend: nextcloud
  conflict_resolution: server_wins

backend_priority:
  - nextcloud
```

**Backend Selection Logic:**
- When `sync.enabled = true`: CLI automatically uses `sync.local_backend` (sqlite) for all operations
- When `sync.enabled = false`: CLI uses `backend_priority` or `default_backend`
- Explicit `--backend` flag always overrides sync selection
- You don't need to include the local backend in `backend_priority` when sync is enabled

**Auto-Sync Behavior:**
- When `sync.auto_sync = true`: Write operations (add/update/complete/delete) trigger background sync
- **Operations return immediately** after writing to sqlite (truly instant)
- **Detached daemon process** spawned to run `gosynctasks sync --quiet` in background
- Daemon process runs independently and completes after parent exits
- Operations are queued in sqlite `sync_queue` table and persist between runs
- Background sync processes the queue automatically
- User experiences instant operations with no waiting
- Background sync is automatic - no manual `sync` command needed for normal operations

**Troubleshooting Auto-Sync:**
- Check startup logs for `Auto-sync initialized successfully` message
- Ensure both `local_backend` and `remote_backend` are configured and enabled
- SQLite `db_path` can be empty (uses XDG default: `~/.local/share/gosynctasks/tasks.db`)
- If sync fails silently, check stderr for `[AutoSync]` log messages
- Background sync errors are logged to stderr but don't interrupt CLI operations

### Testing
- **Unit**: `backend/schema_test.go`, `sqliteBackend_test.go`, `syncManager_test.go`
- **Integration**: `backend/integration_test.go` (7 scenarios including offline mode, conflicts, hierarchical sync)
- **Benchmarks**: `backend/sync_bench_test.go` (1000 tasks <30s)

### Documentation
See [SYNC_GUIDE.md](SYNC_GUIDE.md) for complete usage guide.
