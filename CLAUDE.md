# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

gosynctasks is a Go-based task synchronization tool that interfaces with multiple backends (primarily Nextcloud CalDAV) to manage tasks and task lists. It uses the Cobra CLI framework and supports filtering tasks by status.

## Development Commands

### Building and Running
```bash
# Build the project
go build -o gosynctasks ./cmd/gosynctasks

# Run directly
go run ./cmd/gosynctasks/main.go

# Run with arguments (list tasks from a specific list)
go run ./cmd/gosynctasks/main.go [action] [list-name]

# Filter by status (supports abbreviations: T=TODO, D=DONE, P=PROCESSING, C=CANCELLED)
go run ./cmd/gosynctasks/main.go -s TODO,DONE [list-name]
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./backend
go test ./internal/config
```

### Dependencies
```bash
# Download dependencies
go mod download

# Update dependencies
go mod tidy
```

## Architecture

### Backend System
The application uses a **backend abstraction pattern** with a pluggable architecture:

- **`backend.TaskManager` interface** (backend/taskManager.go): Core interface that all backends must implement
  - Methods: `GetTaskLists()`, `GetTasks()`, `AddTask()`
  - Backends are selected via URL scheme in config (e.g., `nextcloud://`, `file://`)

- **`backend.ConnectorConfig`** (backend/taskManager.go:20-61): Factory that creates TaskManager instances based on URL scheme
  - Parses connector URL from config
  - Routes to appropriate backend implementation

### Backend Implementations

#### NextcloudBackend (backend/nextcloudBackend.go)
- Implements CalDAV protocol for Nextcloud task management
- Uses HTTP REPORT/PROPFIND methods with XML queries
- Credentials extracted from URL (e.g., `nextcloud://user:pass@host`)
- Key methods:
  - `GetTaskLists()`: PROPFIND request to discover calendars with VTODO support
  - `GetTasks()`: REPORT request with calendar-query XML filter
  - `buildCalendarQuery()`: Constructs CalDAV XML queries with status/date filters
  - `parseVTODOs()`: Parses iCalendar VTODO format from responses

#### FileBackend (backend/fileBackend.go)
- Placeholder implementation (not yet functional)
- Intended for local file-based task storage

### Status Translation Layer
The app uses **dual status naming** (backend/taskManager.go:79-123):
- **Internal app statuses**: TODO, DONE, PROCESSING, CANCELLED
- **CalDAV standard statuses**: NEEDS-ACTION, COMPLETED, IN-PROCESS, CANCELLED
- Translation functions: `StatusStringTranslateToStandardStatus()` and `StatusStringTranslateToAppStatus()`
- CLI supports abbreviations (T/D/P/C) which are expanded in main.go:125-134

### Configuration System (internal/config/config.go)
- Uses singleton pattern with `sync.Once` for global config
- Config location: `$XDG_CONFIG_HOME/gosynctasks/config.json`
- On first run, prompts user to create config from embedded sample
- Validation using `github.com/go-playground/validator/v10`
- Structure:
  ```go
  type Config struct {
      Connector backend.ConnectorConfig  // Backend URL and settings
      CanWriteConfig bool
      UI string  // "cli" or "tui"
  }
  ```

### CLI Structure (cmd/gosynctasks/main.go)
- Built with `github.com/spf13/cobra`
- **App struct** encapsulates state:
  - `taskLists`: Cached list of available task lists
  - `taskManager`: Active backend implementation
  - `config`: Global configuration
- **Interactive mode**: If no list name provided, shows numbered selection
- **Shell completion**: Supports tab-completion of task list names
- **Filter building**: Constructs `TaskFilter` from CLI flags

### Data Models

#### Task (backend/taskManager.go:125-138)
Follows iCalendar VTODO spec:
- `UID`, `Summary`, `Description`
- `Status`: NEEDS-ACTION, IN-PROCESS, COMPLETED, CANCELLED
- `Priority`: 0-9 (0=undefined, 1=highest, 9=lowest)
- Timestamps: `Created`, `Modified`, `DueDate`, `StartDate`, `Completed`
- Supports subtasks via `ParentUID`

#### TaskList (backend/taskManager.go:172-179)
- Represents a calendar/list containing tasks
- Contains CalDAV-specific fields: `CTags` for sync, `Color` for UI

### iCalendar Parsing (backend/parseVTODOs.go)
- Manual XML and iCalendar parsing (no external parser library)
- `extractVTODOBlocks()`: Extracts VTODO blocks from CalDAV XML responses
- `parseVTODO()`: Parses individual VTODO into Task struct
- `parseICalTime()`: Handles multiple iCal date/time formats (UTC, local, date-only)
- `unescapeText()`: Handles iCalendar escape sequences (\n, \,, \;, \\)

## Common Patterns

### Adding a New Backend
1. Create new file in `backend/` (e.g., `sqliteBackend.go`)
2. Implement the `TaskManager` interface
3. Add URL scheme case to `ConnectorConfig.TaskManager()` (backend/taskManager.go:50-61)
4. Implement required methods: `GetTaskLists()`, `GetTasks()`, `AddTask()`

### Working with Status Filters
Always use the translation functions when working between app and CalDAV statuses:
- Use `StatusStringTranslateToStandardStatus()` before sending to CalDAV backend
- Use `StatusStringTranslateToAppStatus()` when displaying to user

### HTTP Client Configuration
The NextcloudBackend uses a customized HTTP client (backend/nextcloudBackend.go:22-35):
- `InsecureSkipVerify` configurable via `ConnectorConfig.InsecureSkipVerify` (warns when enabled)
- Connection pooling configured
- 30-second timeout

### Security
- TLS certificate verification is enabled by default
- Can be disabled via config for self-signed certificates (shows warning)
- Credentials extracted from URL (consider using system keyring in future)

## Testing
- **backend/parseVTODOs_test.go**: Tests for iCalendar parsing functions
- **backend/taskManager_test.go**: Tests for status translation functions
- Run tests: `go test ./backend -v`

## Known Issues and TODOs

### Current Limitations
- FileBackend is a placeholder (not implemented)
- NextcloudBackend: Add flags for due date filter
- NextcloudBackend: Test due date filter
- No config package tests yet (needs backup/restore functionality)

## Future Work: SQLite Sync Implementation

Currently the app operates in **read-through mode** (directly queries backend each time). The SQLite database code (main.go:41-57) was initialized but never implemented. The plan is to build a **proper sync system** with offline mode and conflict resolution.

### Architecture Overview

```
┌─────────────────────────────────────┐
│          CLI Command                │
│  gosynctasks add "task"             │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│       SyncManager                   │
│  - Handles sync logic               │
│  - Conflict resolution              │
│  - Offline queue                    │
└──────────┬────────────┬─────────────┘
           │            │
           ▼            ▼
  ┌────────────┐  ┌──────────────┐
  │  SQLite    │  │   Nextcloud  │
  │  Backend   │  │   Backend    │
  │  (local)   │  │   (remote)   │
  └────────────┘  └──────────────┘
```

### Phase 1: Enhanced SQLite Schema with Sync Metadata

Create tables to track synchronization state:

```sql
-- Main tasks table (already exists in code)
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    list_id TEXT NOT NULL,
    summary TEXT NOT NULL,
    description TEXT,
    status TEXT,
    priority INTEGER DEFAULT 0,
    created_at INTEGER,
    modified_at INTEGER,
    due_date INTEGER,
    start_date INTEGER,
    completed_at INTEGER,
    parent_uid TEXT
);

-- Track sync state for each task
CREATE TABLE sync_metadata (
    task_uid TEXT PRIMARY KEY,
    list_id TEXT NOT NULL,

    -- Server state tracking
    remote_etag TEXT,              -- Server's ETag for this task
    last_synced_at INTEGER,        -- Timestamp of last successful sync

    -- Local state flags
    locally_modified INTEGER DEFAULT 0,  -- 1 if changed locally since sync
    locally_deleted INTEGER DEFAULT 0,   -- 1 if deleted locally (pending server delete)

    -- Conflict detection
    remote_modified_at INTEGER,    -- Server's last-modified timestamp
    local_modified_at INTEGER,     -- Our last-modified timestamp

    FOREIGN KEY(task_uid) REFERENCES tasks(id)
);

-- Track sync state for each task list
CREATE TABLE list_sync_metadata (
    list_id TEXT PRIMARY KEY,
    last_ctag TEXT,                -- Calendar's CTag (changes when any task changes)
    last_full_sync INTEGER,        -- Timestamp of last full sync
    sync_token TEXT                -- CalDAV sync-token for incremental sync
);

-- Queue for operations to perform on next sync
CREATE TABLE sync_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_uid TEXT NOT NULL,
    list_id TEXT NOT NULL,
    operation TEXT NOT NULL,       -- 'create', 'update', 'delete'
    created_at INTEGER NOT NULL,
    retry_count INTEGER DEFAULT 0,
    last_error TEXT
);
```

### Phase 2: SQLite Backend Implementation

Create `backend/sqliteBackend.go` implementing the `TaskManager` interface:

```go
type SQLiteBackend struct {
    Connector ConnectorConfig
    db        *sql.DB
}

// Implement TaskManager interface
func (sb *SQLiteBackend) GetTaskLists() ([]TaskList, error)
func (sb *SQLiteBackend) GetTasks(listID string, filter *TaskFilter) ([]Task, error)
func (sb *SQLiteBackend) AddTask(listID string, task Task) error

// Additional sync-specific methods
func (sb *SQLiteBackend) MarkLocallyModified(taskUID string) error
func (sb *SQLiteBackend) GetPendingSyncOperations() ([]SyncOperation, error)
func (sb *SQLiteBackend) ClearSyncFlag(taskUID string) error
```

### Phase 3: SyncManager

Create `backend/syncManager.go` to coordinate between local SQLite and remote backend:

```go
type SyncManager struct {
    local  *SQLiteBackend      // Local SQLite cache
    remote TaskManager         // Remote backend (Nextcloud, etc.)
}

func (sm *SyncManager) Sync() error {
    // 1. Check if remote changed (compare CTags)
    // 2. Pull remote changes
    // 3. Detect conflicts
    // 4. Resolve conflicts
    // 5. Push local changes
    // 6. Update metadata
}
```

**Sync Algorithm:**

1. **Pull Phase (Remote → Local)**
   - Fetch CTag from remote
   - If CTag changed, fetch modified tasks
   - For each remote task:
     - If `locally_modified = 0`: Update local copy (server wins)
     - If `locally_modified = 1`: **CONFLICT** (needs resolution)

2. **Conflict Resolution Strategies**
   - **Server Wins**: Discard local changes, use server version
   - **Local Wins**: Overwrite server with local version
   - **Merge**: Combine non-conflicting fields
   - **Keep Both**: Create duplicate with suffix "(local copy)"

3. **Push Phase (Local → Remote)**
   - Find all tasks with `locally_modified = 1`
   - For each:
     - Upload to remote backend
     - On success: Clear `locally_modified`, update `remote_etag`
     - On failure: Log error, increment retry count

### Phase 4: CLI Integration

**New Commands:**

```bash
# Sync with remote
gosynctasks sync

# Check sync status
gosynctasks sync status

# Force full re-sync
gosynctasks sync --full

# Configure sync settings
gosynctasks config set sync.conflict_resolution server_wins
```

**Offline Mode:**

When operating offline (remote unreachable):
- All operations work against local SQLite
- Changes queued in `sync_queue` table
- Warning displayed: "Working offline - changes will sync later"
- Next `sync` command pushes queued changes

### Phase 5: Implementation Checklist

- [ ] Create enhanced SQLite schema with sync metadata tables
- [ ] Implement SQLiteBackend with basic CRUD operations
- [ ] Add sync metadata tracking methods
- [ ] Create SyncManager with pull/push logic
- [ ] Implement conflict detection
- [ ] Implement conflict resolution strategies (at least server_wins and local_wins)
- [ ] Add `sync` command to CLI
- [ ] Handle offline mode (detect network failures, queue operations)
- [ ] Add sync status reporting
- [ ] Update config to specify sync strategy preference
- [ ] Write tests for sync logic
- [ ] Add incremental sync using CalDAV sync-token (optimization)

### Technical Considerations

**CTag vs ETag:**
- **CTag**: Calendar-level tag that changes when ANY task changes (use for "did anything change?")
- **ETag**: Task-level tag that changes when THAT task changes (use for "which tasks changed?")

**Conflict Resolution:**
- Default strategy should be **server_wins** (safest, prevents data loss on server)
- Make configurable via config file
- Consider three-way merge in future (requires storing baseline version)

**Performance:**
- Use CalDAV sync-token for incremental sync (only fetch changes since last sync)
- Index foreign keys and commonly queried fields
- Consider batch operations for large sync operations

**Error Handling:**
- Transactional sync operations (rollback on failure)
- Retry logic with exponential backoff
- Preserve failed operations for manual resolution
