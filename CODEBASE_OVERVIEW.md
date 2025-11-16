# GoSyncTasks Codebase Overview

## Project Summary
**gosynctasks** is a Go-based task synchronization CLI application (~25K lines of code) that manages tasks across multiple backends with offline sync support. Built with Cobra framework, it implements a pluggable backend architecture supporting Nextcloud CalDAV, Git/Markdown, SQLite, and File backends.

**Key Statistics:**
- Total Go files: 77
- Backend code: ~12K lines (taskManager, backends, sync, database)
- Internal modules: ~7.9K lines (operations, views, config, cache, CLI)
- CLI/Commands: ~2.1K lines (main, list, sync, view commands)
- Test files: 31 test files with integration and unit tests

---

## 1. PROJECT ORGANIZATION & DIRECTORY STRUCTURE

```
/home/user/gosynctasks/
├── cmd/gosynctasks/              # CLI entry points & command implementations
│   ├── main.go                   # Cobra root command, flags, arg parsing
│   ├── list.go                   # List management subcommands
│   ├── sync.go                   # Sync operations
│   ├── view.go                   # View management
│   └── view_test.go
├── backend/                      # Core backend system & implementations
│   ├── taskManager.go            # TaskManager interface & data models (726 lines)
│   ├── selector.go               # Backend registry and selection logic
│   ├── sqliteBackend.go          # SQLite local storage (969 lines)
│   ├── nextcloudBackend.go       # Nextcloud CalDAV (825 lines)
│   ├── gitBackend.go             # Git/Markdown support (621 lines)
│   ├── fileBackend.go            # Placeholder file backend (non-functional)
│   ├── database.go               # SQLite wrapper & initialization
│   ├── schema.go                 # Database schema definitions
│   ├── syncManager.go            # Bidirectional sync orchestration (765 lines)
│   ├── parseVTODOs.go            # iCalendar VTODO parser
│   ├── markdownParser.go         # Markdown task parser
│   ├── markdownWriter.go         # Markdown task writer
│   ├── errors.go                 # Custom error types
│   ├── *_test.go                 # Backend tests (11 test files)
│   └── testing_helpers.go        # Test utilities
├── internal/                     # Internal modules
│   ├── app/
│   │   ├── app.go               # Application state & initialization
│   │   └── app_test.go
│   ├── config/
│   │   ├── config.go            # Configuration management (453 lines)
│   │   ├── config.sample.json
│   │   └── config_test.go
│   ├── operations/              # Business logic for actions
│   │   ├── actions.go           # Action handlers (479 lines)
│   │   ├── tasks.go             # Task operations & search
│   │   ├── subtasks.go          # Hierarchical task support (476 lines)
│   │   ├── lists.go             # List operations
│   │   ├── dates_test.go        # Date parsing tests
│   │   └── *_test.go
│   ├── views/                   # Custom views system
│   │   ├── types.go             # View data structures
│   │   ├── renderer.go          # Rendering engine
│   │   ├── filter.go            # Filtering & sorting logic
│   │   ├── loader.go            # View loading
│   │   ├── storage.go           # View persistence
│   │   ├── resolver.go          # View resolution
│   │   ├── validation.go        # View validation (464 lines)
│   │   ├── fields.go            # Field definitions
│   │   ├── formatters/          # Field-specific formatters
│   │   │   ├── base.go
│   │   │   ├── status.go
│   │   │   ├── priority.go
│   │   │   ├── date.go
│   │   │   ├── text.go
│   │   │   └── *_test.go
│   │   ├── builder/             # Interactive view builder (TUI)
│   │   │   ├── builder.go
│   │   │   ├── model.go
│   │   │   ├── types.go
│   │   │   ├── views.go
│   │   │   └── *_test.go
│   │   ├── testdata/            # Test fixtures
│   │   └── *_test.go
│   ├── cache/
│   │   ├── cache.go             # Task list caching
│   │   └── cache_test.go
│   ├── cli/
│   │   ├── display.go           # Terminal output formatting
│   │   └── completion.go        # Shell completion
│   └── utils/
│       ├── inputs.go            # User input handling
│       └── validation.go        # Input validation
├── docs/                        # Documentation
├── scripts/                     # Test & deployment scripts
├── CLAUDE.md                    # Development guidelines
├── SYNC_GUIDE.md               # Sync system documentation
├── TESTING.md                  # Test documentation
└── go.mod, go.sum             # Dependencies
```

---

## 2. ARCHITECTURE & DESIGN PATTERNS

### 2.1 Pluggable Backend Pattern (Factory + Interface)

**Core Interface: `TaskManager`** (backend/taskManager.go)
- Defines 20+ methods for CRUD operations, list management, status translation
- All backends must implement to support task operations
- Supports polymorphic behavior across different storage systems

**Key Responsibilities:**
```go
type TaskManager interface {
    GetTaskLists() ([]TaskList, error)
    GetTasks(listID string, taskFilter *TaskFilter) ([]Task, error)
    FindTasksBySummary(listID string, summary string) ([]Task, error)
    AddTask(listID string, task Task) error
    UpdateTask(listID string, task Task) error
    DeleteTask(listID string, taskUID string) error
    CreateTaskList(name, description, color string) (string, error)
    // ... 13 more methods for list management, status, priority, display
}
```

**Factory Pattern: BackendConfig.TaskManager()**
- Converts configuration to TaskManager instances
- Supports backward compatibility (ConnectorConfig → BackendConfig migration)
- Type-based routing (nextcloud → NextcloudBackend, sqlite → SQLiteBackend, etc.)

**Optional Extension: DetectableBackend Interface**
- Adds auto-detection capabilities
- Enables environment-based backend selection (e.g., Git repo detection)

### 2.2 Backend Registry & Selection System

**BackendRegistry** (selector.go)
- Manages multiple configured backends
- Lazy initialization: attempts to initialize only enabled backends
- Provides backend info and detection status

**BackendSelector** (selector.go)
- Implements priority-based selection logic:
  1. Explicit backend (--backend flag)
  2. Auto-detection (if enabled in config)
  3. Default backend
  4. First enabled backend
- Supports auto-detection across all available backends

### 2.3 Data Model: Follows iCalendar VTODO Standard

**Task Structure** (backend/taskManager.go)
```go
type Task struct {
    UID         string    // Unique identifier
    Summary     string    // Task title
    Description string    // Details
    Status      string    // Backend-specific (NEEDS-ACTION, COMPLETED, etc.)
    Priority    int       // 0-9 (0=undefined, 1=highest, 9=lowest)
    Created     time.Time
    Modified    time.Time
    DueDate     *time.Time
    StartDate   *time.Time
    Completed   *time.Time
    Categories  []string  // Tags
    ParentUID   string    // For subtask hierarchy
}
```

**TaskList Structure**
```go
type TaskList struct {
    ID          string    // Unique identifier
    Name        string    // Human-readable name
    Description string
    Color       string    // Hex color code
    URL         string    // Backend-specific URL
    CTags       string    // Sync token (CalDAV-specific)
    DeletedAt   string    // Trash tracking (Nextcloud-specific)
}
```

**Hierarchical Organization**
- Supports subtasks via ParentUID field
- Helper: `OrganizeTasksHierarchically()` builds visual hierarchy
- `TaskWithLevel` struct for indented display

### 2.4 Status Translation Layer

**Backend-Specific Status Maps**
- **CalDAV (Nextcloud)**: NEEDS-ACTION, IN-PROCESS, COMPLETED, CANCELLED
- **App Display**: TODO, PROCESSING, DONE, CANCELLED
- **Git/File**: Uses app display names directly

**Translation Functions**
- `StatusStringTranslateToStandardStatus()`: App → CalDAV
- `StatusStringTranslateToAppStatus()`: CalDAV → App
- Each backend implements `ParseStatusFlag()` and `StatusToDisplayName()`

---

## 3. BACKEND IMPLEMENTATIONS

### 3.1 Nextcloud Backend (825 lines)

**Purpose**: CalDAV protocol integration for Nextcloud Tasks

**Architecture**
- HTTP-based communication with Nextcloud server
- CalDAV PROPFIND for list discovery
- CalDAV REPORT for task queries
- PUT/DELETE for create/update/delete operations
- iCalendar VTODO format for data serialization

**Key Features**
- SSL verification support (with InsecureSkipVerify option)
- Connection pooling and timeouts
- Basic authentication via URL credentials
- Priority color coding (1-4=high, 5=medium, 6-9=low)
- Trash support (DeletedAt tracking)

**VTODO Parser** (parseVTODOs.go)
- Converts iCalendar format to Task objects
- Handles date parsing (DTSTART, DTDUE, DTSTAMP, COMPLETED)
- Extracts priority and status from VTODO components

**Sync Metadata Tracking**
- ETags for conflict detection
- CTag (collection tag) for efficient list synchronization
- Supports incremental sync without full re-fetch

### 3.2 SQLite Backend (969 lines)

**Purpose**: Local task storage with sync capabilities

**Architecture**
- XDG-compliant path: `$XDG_DATA_HOME/gosynctasks/tasks.db`
- Schema versioning for migrations
- Transactional operations for consistency
- Foreign keys for referential integrity

**Schema**
- `tasks`: Main task storage with VTODO fields
- `sync_metadata`: Per-task sync state (ETags, flags, timestamps)
- `list_sync_metadata`: Per-list sync state (CTags, tokens)
- `sync_queue`: Pending operations for retry on failure
- `schema_version`: Migration tracking

**Key Features**
- Full CRUD with transaction support
- Sync methods: `MarkLocallyModified()`, `MarkLocallyDeleted()`, `ClearSyncFlags()`
- Automatic UID generation
- Status translation for app display
- Optimized indexes on list_id, status, due_date, parent_uid, priority

**Database Wrapper** (database.go)
- Lazy initialization with error handling
- Pragmas for performance (foreign keys, WAL mode)
- Schema version checking
- Utility methods: `GetStats()`, `Vacuum()`

### 3.3 Git Backend (621 lines)

**Purpose**: Task management in Git repositories with Markdown files

**Architecture**
- Parses TODO.md files (or custom configured files)
- Enables gosynctasks via `<!-- gosynctasks:enabled -->` marker
- Organizes tasks by markdown headers (## Level 1, ### Level 2, etc.)

**File Discovery**
- Walks up directory tree to find `.git` directory
- Searches for configured file (default: TODO.md)
- Falls back to configured fallback_files
- Auto-detection when enabled in config

**Markdown Format**
- Markdown checkboxes: `- [ ]` (TODO), `- [x]` (DONE)
- Metadata via annotations: `@priority:1`, `@due:2025-01-20`, `@completed:2025-01-10`
- Hierarchical via markdown headers and indentation
- Auto-commit support for changes

**Key Features**
- File modification tracking for change detection
- Metadata parser for task attributes
- Optional auto-commit to git
- Task grouping by list name (markdown headers)

### 3.4 File Backend (Placeholder)

**Status**: Non-functional, placeholder implementation
- Returns nil for all operations
- Intended for future file system storage

---

## 4. SYNC SYSTEM (765 lines)

**Architecture**: Bidirectional sync with conflict resolution (SQLite ↔ Remote)

### 4.1 SyncManager Orchestration

```
App → SyncManager (pull/push) ↔ Remote Backend
        ↓
    SQLiteBackend (local persistence & queueing)
```

**Two-Phase Sync**
1. **Pull Phase**: Remote → Local
   - Fetch remote lists and tasks
   - Detect conflicts via timestamps
   - Apply conflict resolution strategy
   - Update local SQLite

2. **Push Phase**: Local → Remote
   - Process sync queue (CREATE/UPDATE/DELETE operations)
   - Execute on remote backend
   - Retry on failure with exponential backoff
   - Update sync metadata

### 4.2 Conflict Resolution Strategies

- **server_wins** (default): Discard local changes
- **local_wins**: Overwrite server with local version
- **merge**: Combine non-conflicting fields
- **keep_both**: Create duplicate with suffix

**Conflict Detection**
- Uses Modified timestamps to detect divergence
- ETags for server-side change tracking
- Flags: locally_modified, locally_deleted

### 4.3 Hierarchical Task Sorting

**Parents Before Children**
- Prevents foreign key violations during sync
- Ensures parent UIDs exist before children reference them
- Recursive depth-first ordering

### 4.4 Failure Handling

- **Exponential Backoff**: Retries with 2^attempt delays
- **Max 5 Retries**: Operations fail after 5 attempts
- **Offline Queue**: Pending operations persist across disconnections
- **Error Logging**: Tracks last_error for debugging

### 4.5 Configuration (config.json)

```json
{
  "sync": {
    "enabled": true,
    "local_backend": "local",        // SQLite backend name
    "remote_backend": "nextcloud",   // Nextcloud backend name
    "conflict_resolution": "server_wins",
    "auto_sync": false,
    "sync_interval": 0,
    "offline_mode": "auto"
  }
}
```

---

## 5. VIEWS SYSTEM (Custom Task Display)

**Storage**: YAML files in `~/.config/gosynctasks/views/`

### 5.1 View Architecture

**Types** (views/types.go)
```go
type View struct {
    Name        string         // Unique identifier
    Description string         // Purpose
    Fields      []FieldConfig  // Display configuration
    FieldOrder  []string       // Display order
    Filters     *ViewFilters   // Default filters
    Display     DisplayOptions // Presentation options
}
```

**Available Fields**
- Basic: status, summary, description
- Dates: due_date, start_date, created, modified, completed
- Metadata: priority, tags, uid, parent

### 5.2 Field Formatters

**Format Plugins** (views/formatters/)
- **StatusFormatter**: Symbol + color (✓/●/✗/○)
- **PriorityFormatter**: Numeric with color coding
- **DateFormatter**: Configurable date format with color by urgency
- **SummaryFormatter**: Text with optional truncation
- **TagsFormatter**: Comma-separated tag list
- **DescriptionFormatter**: Multi-line text with truncation

**Rendering Pipeline**
1. ViewRenderer reads View configuration
2. Initializes formatters for each field
3. Applies filters (status, priority, date ranges, tags)
4. Sorts tasks (by field, asc/desc)
5. Formats each task field
6. Renders with hierarchical indentation

### 5.3 View Builder (TUI)

**Interactive Builder** (views/builder/)
- Guides users through creating custom views
- Field selection and ordering
- Filter configuration
- Format options
- YAML export

### 5.4 Built-in Views

- **basic**: Summary + status (minimal)
- **all**: Complete metadata display

---

## 6. CONFIGURATION SYSTEM (453 lines)

**Location**: `$XDG_CONFIG_HOME/gosynctasks/config.json` (or `~/.config/gosynctasks/config.json`)

### 6.1 Multi-Backend Configuration (Current)

```json
{
  "backends": {
    "local": {
      "type": "sqlite",
      "enabled": true,
      "db_path": ""  // XDG-compliant default
    },
    "nextcloud": {
      "type": "nextcloud",
      "enabled": true,
      "url": "nextcloud://user:pass@host",
      "insecure_skip_verify": false,
      "suppress_ssl_warning": false
    },
    "git": {
      "type": "git",
      "enabled": true,
      "file": "TODO.md",
      "auto_detect": true,
      "fallback_files": ["TODO.md", "tasks.md"],
      "auto_commit": false
    }
  },
  "default_backend": "nextcloud",
  "auto_detect_backend": true,
  "backend_priority": ["git", "nextcloud", "local"],
  "sync": { /* ... */ },
  "ui": "cli",
  "date_format": "2006-01-02"
}
```

### 6.2 Legacy Configuration (Auto-Migration)

**Old Format**: Single connector
```json
{
  "connector": {
    "url": "nextcloud://user:pass@host"
  }
}
```

**Migration Process**
- Detects old format on load
- Creates backup: `config.json.backup`
- Converts to new multi-backend format
- Saves updated config

### 6.3 First-Run Setup

- Embedded sample config
- Automatic creation in XDG directory
- User prompts for backend selection

### 6.4 Configuration Loading

**Singleton Pattern** (sync.Once)
- Single global Config instance
- Lazy initialization on first access
- Custom path support via `--config` flag

**Priority Resolution**
- `--config <path>`: Override everything
- `$XDG_CONFIG_HOME/gosynctasks/config.json`
- `~/.config/gosynctasks/config.json`
- First-run setup if missing

---

## 7. CLI STRUCTURE & COMMAND ORGANIZATION

**Framework**: Cobra command framework

### 7.1 Root Command (main.go)

```
gosynctasks [list-name] [action] [task-summary]
```

**Global Flags**
- `--config <path>`: Override config location
- `--backend <name>`: Explicit backend selection
- `--list-backends`: Display all backends and exit
- `--detect-backend`: Show auto-detected backends and exit

**Default Command**: `get` (list tasks)

### 7.2 Action Commands

**Actions** (with abbreviations)
- `get` (g): List tasks from a list (default)
- `add` (a): Create new task
- `update` (u): Modify existing task
- `complete` (c): Mark as DONE
- `delete` (d): Remove task

**Action Flags**
- `--status, -s <status>`: Filter/set status
- `--view, -v <view>`: Display format
- `--description, -d <text>`: Task details
- `--priority, -p <0-9>`: Priority level
- `--add-status, -S <status>`: Status when adding
- `--summary <text>`: Rename task (update)
- `--due-date <YYYY-MM-DD>`: Deadline
- `--start-date <YYYY-MM-DD>`: Start date
- `--parent, -P <task>`: Parent for subtask
- `--literal, -l`: Disable path-based hierarchy

### 7.3 List Management (list.go)

**Subcommands**
- `list create <name>`: Create new list
- `list delete <name>`: Remove list
- `list rename <old> <new>`: Rename
- `list info <name>`: Show details
- `list trash`: View deleted lists
- `list trash restore <name>`: Restore from trash

### 7.4 View Management (view.go)

**Subcommands**
- `view list`: Show available views
- `view create <name>`: Interactive view builder
- `view delete <name>`: Remove custom view
- `view show <name>`: Display view configuration

### 7.5 Sync Operations (sync.go)

**Subcommands**
- `sync`: Perform bidirectional sync
- `sync --full`: Force complete re-sync
- `sync --dry-run`: Preview changes
- `sync -l <list>`: Sync specific list
- `sync status`: Show sync state
- `sync queue`: View pending operations
- `sync queue clear`: Clear failed operations

**Offline Detection**
- Network connectivity check
- Graceful degradation in offline mode
- Queue pending operations for sync when online

---

## 8. OPERATIONS LAYER (Business Logic)

**Entry Point**: `operations.ExecuteAction()` (actions.go)

### 8.1 Action Handlers

**Pattern**: Each action has dedicated handler function

1. **HandleGetAction** (actions.go)
   - Build filters from flags (status, priority, dates)
   - Fetch tasks from backend
   - Apply view formatting
   - Display with optional hierarchical indentation

2. **HandleAddAction** (actions.go)
   - Parse summary (with optional path-based hierarchy)
   - Resolve parent task reference
   - Set default status
   - Create task in backend

3. **HandleUpdateAction** (actions.go)
   - Find task by summary (intelligent matching)
   - Parse update flags
   - Apply changes to backend

4. **HandleCompleteAction** (actions.go)
   - Find task by summary
   - Set status to DONE
   - Set Completed timestamp

5. **HandleDeleteAction** (actions.go)
   - Find task by summary
   - Confirm deletion with user
   - Remove from backend

### 8.2 Task Search & Selection (tasks.go)

**FindTaskBySummary()** - Intelligent Matching
- Exact match: proceed immediately
- Single partial match: confirm with user
- Multiple matches: show selection with hierarchy paths
- Supports cancellation at any prompt

**Hierarchical Path Display**
- Shows full parent/child path for disambiguation
- Example: "Work > Bug Fix > Fix authentication"

### 8.3 Subtask Support (subtasks.go - 476 lines)

**Features**
- Path-based creation: `add "parent/child/grandchild"`
- Auto-creates missing hierarchy levels
- Supports `-P "Parent Task"` flag for simple parent reference
- Path resolution with confirmation for non-existent parents

**Functions**
- `CreateOrFindTaskPath()`: Build hierarchy recursively
- `ResolveParentTask()`: Convert reference to UID
- `findTaskByParent()`: Search within hierarchy level

### 8.4 List Operations (lists.go)

**Functions**
- `FindListByName()`: Case-insensitive list lookup
- `SelectListInteractively()`: TUI list selector
- `GetSelectedList()`: Explicit or interactive selection

---

## 9. CACHING SYSTEM

**Purpose**: Fallback task list cache for offline access

**Storage**: `$XDG_CACHE_HOME/gosynctasks/lists.json` (~/.cache/gosynctasks/lists.json)

**Operations**
- `LoadTaskListsWithFallback()`: Cache → backend with fallback
- `RefreshAndCacheTaskLists()`: Refresh from backend and update cache
- `RefreshTaskListsOrWarn()`: Non-critical refresh with logging

**Behavior**
- Cache loaded on startup if backend unavailable
- Automatically updated on successful backend access
- Includes timestamp for potential expiration logic

---

## 10. TESTING ORGANIZATION (31 Test Files)

### 10.1 Test Structure

**Backend Tests** (backend/*_test.go)
```
- taskManager_test.go          # Data model & interface tests
- taskManager_hierarchy_test.go # Hierarchical organization tests
- nextcloudBackend_test.go     # Nextcloud implementation (993 lines)
- sqliteBackend_test.go        # SQLite implementation (919 lines)
- gitBackend_test.go           # Git backend tests
- parseVTODOs_test.go          # VTODO parser (602 lines)
- syncManager_test.go          # Sync orchestration (677 lines)
- sync_bench_test.go           # Performance benchmarks
- schema_test.go               # Database schema (616 lines)
- integration_test.go          # End-to-end scenarios (649 lines)
- backendDisplay_test.go       # Display formatting
- formatWithView_test.go       # View rendering
- selector_test.go             # Backend selection
```

**Internal Tests** (internal/*_test.go)
```
- views/renderer_test.go       # View rendering
- views/filter_test.go         # Filtering logic
- views/validation_test.go     # View validation
- views/fields_test.go         # Field definitions
- views/formatters/status_test.go
- views/builder/model_test.go
- views/builder/types_test.go
- operations/subtasks_test.go
- operations/tasks_test.go
- config/config_test.go
- cache/cache_test.go
- utils/inputs_test.go
- cmd/gosynctasks/view_test.go
```

### 10.2 Integration Tests

**Integration Scenarios** (integration_test.go - 649 lines)
1. Basic sync workflow (remote → local → modify → push)
2. Conflict detection and resolution
3. Offline mode with queue persistence
4. Hierarchical task sync (parent/child ordering)
5. Partial sync (specific list)
6. Large-scale sync (1000+ tasks)
7. Edge cases (circular references, deleted parents)

**Mock Backend** (testing_helpers.go)
- In-memory implementation for testing
- Simulates remote server behavior
- Tracks operations for verification

### 10.3 Test Configuration

**Location**: `./gosynctasks/config/config.json` (pre-configured for localhost:8080)

**Docker Test Server**
```bash
./scripts/start-test-server.sh  # Start Nextcloud test instance
```

**Test Runner**
```bash
go test ./...                   # All tests
go test ./backend -v            # Verbose backend tests
go test -bench ./... -benchmem  # Benchmarks with memory
```

---

## 11. ARCHITECTURAL STRENGTHS

### 11.1 Clean Separation of Concerns
- **Backend layer**: Storage abstraction via TaskManager interface
- **Operations layer**: Business logic isolated from CLI
- **CLI layer**: User interaction via Cobra commands
- **Internal modules**: Utilities (config, cache, views) independent of backends

### 11.2 Pluggable Backend Architecture
- New backends added by implementing single interface
- No modification to existing code needed
- Backend discovery and registration automatic
- Supports multiple backends simultaneously

### 11.3 Robust Sync Implementation
- Bidirectional synchronization with offline support
- Multiple conflict resolution strategies
- Hierarchical task ordering prevents FK violations
- Exponential backoff with retry limits
- Persistent operation queue

### 11.4 Comprehensive Test Coverage
- Unit tests for core functionality
- Integration tests for end-to-end workflows
- Mock backend for testing without external services
- Performance benchmarks for sync operations

### 11.5 User Experience
- Intelligent task search (exact → partial → multiple with selection)
- Hierarchical path display for disambiguation
- Interactive list and task selection
- Flexible view system for custom display
- Shell completion support

### 11.6 Standards Compliance
- iCalendar VTODO format for CalDAV interoperability
- XDG Base Directory specification for file locations
- Git integration for version-controlled tasks

---

## 12. ARCHITECTURAL CONCERNS & AREAS FOR IMPROVEMENT

### 12.1 Potential Concerns

1. **Error Handling Variability**
   - Some methods return generic errors vs custom BackendError type
   - Inconsistent error wrapping across backends
   - Missing error classification in some operations

2. **Sync Queue Limitations**
   - Simple UNIQUE constraint on (task_uid, operation)
   - Doesn't distinguish between different types of modifications
   - No operation scheduling/prioritization

3. **Concurrent Access**
   - No explicit locking in SQLiteBackend
   - Multiple goroutines could race on database access
   - App state (taskLists) could become stale

4. **ViewRenderer Customization**
   - Fixed field formatters; hard to extend
   - Limited format options for dates and other fields
   - Coupling between View structure and Renderer

5. **Status Translation Fragility**
   - Hardcoded mappings in each backend
   - No unified translation strategy
   - Potential for inconsistency across backends

6. **Performance Considerations**
   - No connection pooling for Nextcloud
   - Full list refresh on every operation
   - Cache TTL not implemented
   - Large task lists could slow CLI

### 12.2 Code Organization Opportunities

1. **Backend Methods**: Some methods are contextual (GetPriorityColor, GetBackendDisplayName)
   - Could be better as separate strategy objects
   
2. **Operation Handlers**: Each action handler is separate function
   - Could benefit from common base or shared error handling
   
3. **View Filtering**: Complex filter logic in renderer
   - Could be extracted to separate filter engine

4. **Configuration Schema**: Uses validator tags without centralized schema
   - Could benefit from schema definition system

---

## 13. NOTABLE PATTERNS & TECHNIQUES

### 13.1 Embedded Resources
- Sample config included as embedded file (`//go:embed config.sample.json`)
- Auto-extracted on first run

### 13.2 Singleton Configuration
- Global config via `sync.Once` for safe initialization
- Custom path override support

### 13.3 Lazy Backend Initialization
- Backends initialized only when accessed
- Registry gracefully skips failed initializations
- Reduces startup time for unused backends

### 13.4 Type-Based Command Routing
- Cobra framework with polymorphic handlers
- Action normalization (abbreviations → full names)
- Smart argument parsing based on action type

### 13.5 ANSI Color Coding
- Status indicators (✓/●/✗/○) with colors
- Priority-based color highlighting
- Date urgency coloring (red=overdue, yellow=soon, gray=future)
- Smart color composition (priority + bold + status)

### 13.6 Hierarchical Indentation
- Task depth calculated via ParentUID traversal
- 2-space indentation per level
- Prevents circular reference infinite loops

### 13.7 Terminal Width Detection
- Uses golang.org/x/term for cross-platform detection
- Falls back to 80 chars with constraints (40-100)
- Adaptive formatting for narrow terminals

---

## 14. DEPENDENCY ANALYSIS

**Key External Dependencies** (go.mod):
- **github.com/spf13/cobra**: CLI framework (v1.9.1)
- **golang.org/x/term**: Terminal capabilities
- **gopkg.in/yaml.v3**: YAML parsing for views
- **github.com/go-playground/validator/v10**: Config validation
- **modernc.org/sqlite**: Pure Go SQLite driver (v1.37.1)
- **Charmbracelet libraries**: TUI components (for view builder)

**Internal Only** (No external task/backend dependencies):
- HTTP for Nextcloud (stdlib)
- iCalendar parsing (custom)
- Markdown parsing (custom)
- Git command execution (stdlib os/exec)

---

## 15. CODEBASE QUALITY METRICS

| Metric | Value |
|--------|-------|
| Total Lines | ~24,953 |
| Backend Code | ~12,017 (48%) |
| Internal Modules | ~7,894 (32%) |
| CLI Code | ~2,164 (9%) |
| Test Coverage | 31 test files |
| Largest Module | nextcloudBackend.go (825 lines) |
| Largest Test | sqliteBackend_test.go (919 lines) |
| Number of Backends | 4 (3 functional) |
| Number of Commands | 3 main + 15 subcommands |
| Number of Custom Views | Unlimited (user-defined) |

