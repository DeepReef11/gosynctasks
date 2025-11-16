# GoSyncTasks Codebase Quick Reference

## File Paths & Purposes

### Backend System (~12K lines)
| File | Lines | Purpose |
|------|-------|---------|
| `backend/taskManager.go` | 726 | Core interface & data models |
| `backend/selector.go` | 178 | Backend registry & selection |
| `backend/sqliteBackend.go` | 969 | Local SQLite storage |
| `backend/syncManager.go` | 765 | Bidirectional sync orchestration |
| `backend/nextcloudBackend.go` | 825 | Nextcloud CalDAV integration |
| `backend/gitBackend.go` | 621 | Git/Markdown task storage |
| `backend/database.go` | ~150 | SQLite connection & schema |
| `backend/schema.go` | ~110 | Database table definitions |
| `backend/parseVTODOs.go` | ~200 | iCalendar VTODO parser |
| `backend/markdownParser.go` | ~180 | Markdown task parser |
| `backend/errors.go` | ~40 | Error type definitions |

### Internal Modules (~7.9K lines)
| Module | Lines | Key Files | Purpose |
|--------|-------|-----------|---------|
| `config/` | 453 | config.go | Configuration loading & validation |
| `operations/` | 1360 | actions.go, tasks.go, subtasks.go | Business logic for all actions |
| `views/` | 2500+ | types.go, renderer.go, filter.go, formatters/* | Custom task display system |
| `cache/` | 100 | cache.go | Task list caching |
| `cli/` | ~150 | display.go, completion.go | Terminal output & completion |
| `utils/` | ~100 | inputs.go, validation.go | Input handling & validation |
| `app/` | 160 | app.go | Application state & initialization |

### CLI Commands (~2.1K lines)
| File | Lines | Purpose |
|------|-------|---------|
| `cmd/gosynctasks/main.go` | ~150 | Root command & flag setup |
| `cmd/gosynctasks/list.go` | 650 | List management subcommands |
| `cmd/gosynctasks/sync.go` | 544 | Sync operations & offline detection |
| `cmd/gosynctasks/view.go` | 626 | View management subcommands |

---

## Key Classes & Interfaces

### Core Interface: TaskManager
```go
// 20+ methods across 5 categories:
1. CRUD:
   - GetTasks(listID, filter) → []Task
   - FindTasksBySummary(listID, summary) → []Task
   - AddTask(listID, task) → error
   - UpdateTask(listID, task) → error
   - DeleteTask(listID, taskUID) → error

2. List Management:
   - GetTaskLists() → []TaskList
   - CreateTaskList(name, desc, color) → string
   - DeleteTaskList(listID) → error
   - RenameTaskList(listID, newName) → error
   - GetDeletedTaskLists() → []TaskList
   - RestoreTaskList(listID) → error

3. Status & Display:
   - ParseStatusFlag(flag) → string
   - StatusToDisplayName(status) → string
   - GetPriorityColor(priority) → string
   - GetBackendDisplayName() → string
   - GetBackendType() → string
   - GetBackendContext() → string

4. Sorting:
   - SortTasks(tasks)
```

### Data Models
```go
type Task struct {
    UID, Summary, Description, Status string
    Priority int          // 0-9
    Created, Modified, DueDate, StartDate, Completed time.Time
    Categories []string   // Tags
    ParentUID string      // For hierarchy
}

type TaskList struct {
    ID, Name, Description, Color, URL, CTags, DeletedAt string
}

type TaskFilter struct {
    Statuses *[]string
    DueAfter, DueBefore, CreatedAfter, CreatedBefore *time.Time
}
```

### Backend Registry
```go
type BackendRegistry struct {
    backends map[string]TaskManager
    configs map[string]BackendConfig
}

type BackendSelector struct {
    registry *BackendRegistry
}
// Selection priority: explicit → auto-detect → default → first enabled
```

### Sync Manager
```go
type SyncManager struct {
    local *SQLiteBackend
    remote TaskManager
    strategy ConflictResolutionStrategy
}

type SyncResult struct {
    PulledTasks, PushedTasks, ConflictsFound, ConflictsResolved int
    Errors []error
    Duration time.Duration
}
```

---

## Common Code Patterns

### 1. Creating a New Backend
```go
// 1. Create type implementing TaskManager
type MyBackend struct {
    config BackendConfig
    // ... backend-specific fields
}

// 2. Implement all 20+ TaskManager methods
func (mb *MyBackend) GetTasks(listID string, filter *TaskFilter) ([]Task, error) {
    // Implementation
}

// 3. Add to BackendConfig.TaskManager()
case "mytype":
    return NewMyBackend(*bc)

// 4. Optional: Implement DetectableBackend for auto-detection
func (mb *MyBackend) CanDetect() (bool, error) {
    // Check if backend is available
}
```

### 2. Adding a New Action
```go
// 1. Add handler in operations/actions.go
func HandleMyAction(cmd *cobra.Command, tm backend.TaskManager, cfg *config.Config, list *backend.TaskList) error {
    // Get flags: cmd.Flags().GetString("flag-name")
    // Call backend methods
    // Return result or error
}

// 2. Route in ExecuteAction()
case "myaction":
    return HandleMyAction(cmd, taskManager, cfg, selectedList)

// 3. Add to main.go flags
rootCmd.Flags().StringP("myaction", "m", "", "description")
```

### 3. Adding a View Field
```go
// 1. Define in views/types.go FieldConfig
// Name must match: status, summary, priority, due_date, etc.

// 2. Create formatter in views/formatters/
type MyFormatter struct {
    ctx *FormatContext
}

func (f *MyFormatter) Format(task backend.Task, config views.FieldConfig) string {
    // Return formatted output with optional ANSI codes
}

// 3. Register in ViewRenderer.initializeFormatters()
case "myfield":
    formatter = NewMyFormatter(r.ctx)
```

---

## Configuration Examples

### Multi-Backend Setup
```json
{
  "backends": {
    "local": {
      "type": "sqlite",
      "enabled": true,
      "db_path": "/custom/path.db"
    },
    "cloud": {
      "type": "nextcloud",
      "enabled": true,
      "url": "nextcloud://user:pass@cloud.example.com"
    },
    "notes": {
      "type": "git",
      "enabled": true,
      "file": "TODO.md",
      "auto_detect": true
    }
  },
  "default_backend": "local",
  "auto_detect_backend": true,
  "backend_priority": ["notes", "cloud", "local"],
  "sync": {
    "enabled": true,
    "local_backend": "local",
    "remote_backend": "cloud",
    "conflict_resolution": "server_wins"
  },
  "ui": "cli",
  "date_format": "2006-01-02"
}
```

### Sync Conflict Resolution
```
server_wins (default): Remote version overwrites local
local_wins: Local version uploaded to remote
merge: Combine fields (newer timestamps win)
keep_both: Create duplicate with suffix "_local"
```

---

## Testing Guide

### Running Tests
```bash
go test ./...                    # All tests
go test ./backend -v             # Verbose backend tests
go test -run TestName            # Specific test
go test -bench ./...             # Benchmarks
go test -count 100 ./...         # Repeat 100 times
```

### Test Structure
- Unit tests: Test individual functions
- Integration tests: End-to-end workflows
- Mock backend: In-memory implementation for testing
- Test helpers: Common setup/teardown functions

### Key Test Files
- `backend/integration_test.go`: 7 end-to-end scenarios
- `backend/syncManager_test.go`: Sync operations
- `backend/sqliteBackend_test.go`: Local storage
- `backend/nextcloudBackend_test.go`: CalDAV integration

---

## Error Handling Patterns

### Custom Errors
```go
// SQLiteError
type SQLiteError struct {
    Op string
    Err error
    ListID, TaskUID string
}

// BackendError (from Nextcloud)
type BackendError struct {
    StatusCode int
    Message string
    // Methods: IsNotFound(), IsUnauthorized()
}
```

### Error Classification
```go
// Check error type
if err != nil {
    if backendErr, ok := err.(*backend.BackendError); ok {
        if backendErr.IsUnauthorized() {
            // Handle auth error
        }
    }
}
```

---

## Performance Considerations

### Optimization Points
1. **Cache**: Task lists cached in `~/.cache/gosynctasks/lists.json`
2. **Indexes**: Database has 6 indexes on common queries
3. **Lazy Loading**: Backends initialized only when needed
4. **Connection Pooling**: Nextcloud uses HTTP client with pooling
5. **Sync Efficiency**: ETags & CTags prevent re-fetching

### Bottlenecks
1. Full list refresh on every operation
2. No cache TTL (always refreshes)
3. Large task lists require full parsing
4. No pagination support

---

## Code Organization Quality

### Strengths
- Clean separation: CLI → Operations → Backend
- Pluggable design: Easy to add new backends
- Comprehensive interfaces: Clear contracts
- Well-tested: 31 test files with integration tests
- Standards-based: iCalendar VTODO format

### Areas for Improvement
1. Error handling inconsistency
2. Sync queue limitations
3. No concurrent operation locking
4. View renderer coupling
5. Status translation fragility

---

## Module Dependency Map

```
cli/ (main.go)
  ├─ operations/ (actions, tasks, subtasks)
  │    ├─ backend/ (TaskManager interface)
  │    │    ├─ sqliteBackend → syncManager → remote backend
  │    │    ├─ nextcloudBackend (CalDAV)
  │    │    ├─ gitBackend (Markdown)
  │    │    └─ fileBackend (stub)
  │    ├─ config/ (configuration loading)
  │    ├─ cache/ (task list caching)
  │    └─ views/ (custom display)
  │         ├─ formatters/ (status, priority, date)
  │         └─ builder/ (interactive TUI)
  └─ app/ (application state)
       ├─ backend/selector (backend registry)
       └─ cache/
```

---

## XDG Compliance

**Configuration**: `$XDG_CONFIG_HOME/gosynctasks/config.json` (~/. config default)
**Cache**: `$XDG_CACHE_HOME/gosynctasks/lists.json` (~/.cache default)
**Data**: `$XDG_DATA_HOME/gosynctasks/tasks.db` (~/.local/share default)

---

## Common Tasks

### Debug Sync Issues
1. Check `backend/syncManager.go` pull/push phases
2. Review `sync_metadata` and `sync_queue` tables
3. Check `last_error` column for retry failures
4. Verify conflict resolution strategy in config

### Add New Status Type
1. Update `backend/taskManager.go` status translation functions
2. Update each backend's ParseStatusFlag()
3. Update formatters/status.go for display
4. Add tests for new status

### Customize Task Display
1. Create YAML in `~/.config/gosynctasks/views/myview.yaml`
2. Define fields, filters, display options
3. Use `--view myview` flag

### Test Integration
1. Run `./scripts/start-test-server.sh` (requires Docker)
2. Run `go test ./backend -v`
3. Review integration_test.go for scenarios

---

## Documentation Files
- **CODEBASE_OVERVIEW.md**: Comprehensive architecture overview
- **ARCHITECTURE_DIAGRAMS.md**: Visual diagrams and data flows
- **CLAUDE.md**: Development guidelines
- **SYNC_GUIDE.md**: Sync system detailed documentation
- **TESTING.md**: Test methodology and coverage
- **README.md**: User-facing documentation
- **BUILD_AND_TEST.md**: Build instructions
