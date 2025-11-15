# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

gosynctasks is a Go-based task synchronization tool that interfaces with multiple backends (primarily Nextcloud CalDAV) to manage tasks and task lists. It uses the Cobra CLI framework and supports filtering tasks by status.

After completing working changes, build again.
After doing TESTING.md, provide one or more command to test for the user for the added changes.

**IMPORTANT - Testing with Docker:**
When providing test commands or when test commands fail with connection errors, ALWAYS remind the user to ensure the Docker test server is running. You should not launch that command, it's the user:
```bash
./scripts/start-test-server.sh
```
The test config at `./gosynctasks/config/config.json` is pre-configured for the Docker test server (localhost:8080).

## Development Commands

### Building and Running
```bash
# Build the project
go build -o gosynctasks ./cmd/gosynctasks

# Run directly
go run ./cmd/gosynctasks

# For testing with the Docker test server, use the gst function:
gst() {
      go run ./cmd/gosynctasks --config ./gosynctasks/config "$@"
  }
eval "$(./gosynctasks/gosynctasks completion zsh)"
compdef gst=gosynctasks# See Shell Completion section for autocompletion setup with gst

# Basic usage patterns
gosynctasks                              # Interactive list selection
gosynctasks MyList                       # Show tasks from "MyList"
gosynctasks MyList get                   # Explicit get action (g also works)
gosynctasks MyList -s TODO,DONE          # Filter by status (T/D/P/C abbreviations)

# With test config (using gst alias)
gst                                      # Interactive list selection (test server)
gst Test                                 # Show tasks from "Test" list
gst Test get                             # Explicit get action

# Adding tasks
gosynctasks MyList add "Task summary"    # Create task (a also works)
gosynctasks MyList add                   # Will prompt for summary
gosynctasks MyList add "Task" -d "Details" -p 1 -S done  # With description, priority, status

# Adding tasks with test server (using gst)
gst Test add "Task summary"              # Create task on test server
gst Test add "Task" -d "Details" -p 1 -S done  # With options

# Adding subtasks (hierarchical tasks)
gosynctasks MyList add "Subtask" -P "Parent Task"              # Add subtask under parent
gosynctasks MyList add "Sub-subtask" -P "Parent/Subtask"       # Path-based parent reference
gosynctasks MyList add "Fix bug" -P "Feature X/Write code"     # Deep nesting support

# Path-based task creation shorthand (auto-creates missing parents)
gosynctasks MyList add "parent/child/grandchild"               # Creates entire hierarchy automatically
gosynctasks MyList add "Feature X/Write code/Fix auth bug"     # Creates Feature X, Write code, then Fix auth bug
gosynctasks MyList add -l "be a good/generous person"          # Use -l/--literal to disable path parsing
gosynctasks MyList add --literal "URL: http://example.com"     # Literal flag prevents "/" from being parsed

# Updating tasks
gosynctasks MyList update "task name" -s DONE     # Find and update status (u also works)
gosynctasks MyList update "partial" -p 5          # Partial match, update priority
gosynctasks MyList update "old" --summary "new"   # Rename task

# Updating tasks with test server (using gst)
gst Test update "task name" -s DONE      # Update task on test server
gst Test update "partial" -p 5           # Update priority

# Completing tasks (shortcut for status changes)
gosynctasks MyList complete "task name"           # Mark as DONE (c also works)
gosynctasks MyList complete "task" -s TODO        # Change to TODO
gosynctasks MyList complete "task" -s PROCESSING  # Change to PROCESSING

# View options
gosynctasks MyList -v all                # Show all metadata (dates, priority)
gst Test -v all                          # View with test server
gosynctasks MyList -v myview             # Use custom view named 'myview'

# Custom views management
gosynctasks view list                    # List all available views
gosynctasks view show myview             # Show view configuration
gosynctasks view create myview           # Create view interactively
gosynctasks view edit myview             # Edit existing view
gosynctasks view delete myview           # Delete a view
gosynctasks view validate myview         # Validate view configuration

# List management
gosynctasks list create "New List"            # Create new task list (c also works)
gosynctasks list create "Project X" -d "Description" --color "#ff0000"  # With options
gosynctasks list info MyList                  # Show list details (i also works)
gosynctasks list rename "Old Name" "New Name" # Rename list (r also works)
gosynctasks list delete "List Name"           # Delete list (d also works)

# List trash management (view and restore deleted lists)
gosynctasks list trash                        # Show all deleted lists in trash
gosynctasks list trash restore "List Name"    # Restore a deleted list from trash
gosynctasks list trash empty "List Name"      # Permanently delete a list from trash
gosynctasks list trash empty --all            # Empty entire trash (WARNING: irreversible!)

# List management with test server (using gst)
gst list create "New List"               # Create list on test server
gst list info Test                       # Show test list details
gst list                                 # List all lists on test server
gst list trash                           # Show deleted lists on test server
gst list trash restore "Old List"        # Restore deleted list on test server
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./backend
go test ./internal/config
```

For manual feature testing, see [TESTING.md](TESTING.md) which provides a quick workflow for verifying core functionality using a test configuration.

### Shell Completion

gosynctasks supports autocompletion for bash, zsh, fish, and PowerShell via Cobra's built-in completion:

```bash
# Zsh (load completion)
eval "$(gosynctasks completion zsh)"

# Bash (load completion)
eval "$(gosynctasks completion bash)"

# Fish
gosynctasks completion fish | source

# PowerShell
gosynctasks completion powershell | Out-String | Invoke-Expression
```

**Creating a test function with autocompletion:**
```bash
# Zsh (from project directory)
gst() {
    ./gosynctasks --config ./gosynctasks/config/config.json "$@"
}
eval "$(./gosynctasks completion zsh)"
compdef gst=gosynctasks

# Bash (from project directory)
gst() {
    ./gosynctasks --config ./gosynctasks/config/config.json "$@"
}
eval "$(./gosynctasks completion bash)"
complete -F __start_gosynctasks gst

# If gosynctasks is in your PATH, use:
# gst() { gosynctasks --config ./gosynctasks/config/config.json "$@"; }
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

- **`backend.TaskManager` interface** (backend/taskManager.go:69-80): Core interface that all backends must implement
  - Required methods:
    - `GetTaskLists()`: Retrieve all task lists
    - `GetTasks(listID, filter)`: Get filtered tasks from a list
    - `FindTasksBySummary(listID, summary)`: Search tasks by summary (exact/partial)
    - `AddTask(listID, task)`: Create a new task
    - `UpdateTask(listID, task)`: Update an existing task
    - `SortTasks(tasks)`: Sort tasks using backend-specific ordering
    - `GetPriorityColor(priority)`: Get ANSI color code for task priority
  - Backends are selected via URL scheme in config (e.g., `nextcloud://`, `file://`)

- **`backend.ConnectorConfig`** (backend/taskManager.go:20-61): Factory that creates TaskManager instances based on URL scheme
  - Parses connector URL from config
  - Routes to appropriate backend implementation

### Backend Implementations

#### NextcloudBackend (backend/nextcloudBackend.go)
- Implements CalDAV protocol for Nextcloud task management
- Uses HTTP REPORT/PROPFIND/PUT methods with XML queries
- Credentials extracted from URL (e.g., `nextcloud://user:pass@host`)
- **Full CRUD support**: Create, Read, Update operations
- Key methods:
  - `GetTaskLists()` (line 184-227): PROPFIND request to discover calendars with VTODO support
  - `GetTasks()` (line 119-156): REPORT request with calendar-query XML filter
  - `FindTasksBySummary()` (line 158-182): Client-side task search (exact/partial matches)
  - `AddTask()` (line 230-283): PUT request with iCalendar VTODO content
  - `UpdateTask()` (line 285-334): PUT request to update existing task
  - `buildCalendarQuery()` (line 65-118): Constructs CalDAV XML queries with status/date filters
  - `buildICalContent()` (line 336-385): Builds iCalendar format for task creation/update
  - `parseVTODOs()`: Parses iCalendar VTODO format from responses
  - `SortTasks()` (line 387-403): Nextcloud-specific priority sorting (1=highest, 0=undefined goes last)
  - `GetPriorityColor()` (line 405-415): Nextcloud color scheme (1-4=red, 5=yellow, 6-9=blue)

#### FileBackend (backend/fileBackend.go)
- Placeholder implementation (not yet functional)
- Intended for local file-based task storage
- Contains stub implementations for all TaskManager interface methods
- Returns nil/empty values for all operations

### Status Translation Layer
The app uses **dual status naming** (backend/taskManager.go:89-133):
- **Internal app statuses**: TODO, DONE, PROCESSING, CANCELLED
- **CalDAV standard statuses**: NEEDS-ACTION, COMPLETED, IN-PROCESS, CANCELLED
- Translation functions: `StatusStringTranslateToStandardStatus()` and `StatusStringTranslateToAppStatus()`
- CLI supports abbreviations (T/D/P/C) which are expanded throughout main.go (lines 371-380, 475-490, 534-547, etc.)

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
- **Argument order**: `gosynctasks [list-name] [action] [task-summary]`
  - List name comes first (context-based approach)
  - Action is optional (defaults to `get`)
  - Task summary for add/update/complete operations
- **App struct** encapsulates state:
  - `taskLists`: Cached list of available task lists
  - `taskManager`: Active backend implementation
  - `config`: Global configuration
- **Task list caching** (lines 62-153):
  - Cache location: `$XDG_CACHE_HOME/gosynctasks/lists.json`
  - Stores task lists with timestamp
  - Methods: `loadTaskListsFromCache()`, `saveTaskListsToCache()`, `refreshTaskLists()`
- **Action-based commands** (line 389-628):
  - `get` (alias: `g`): List tasks from a list (default action)
  - `add` (alias: `a`): Create new task with summary/description/priority
  - `update` (alias: `u`): Update task by searching summary
  - `complete` (alias: `c`): Change task status (defaults to DONE)
- **Task search functionality** (lines 252-350):
  - `findTaskBySummary()`: Searches for tasks with exact/partial matching
  - `selectTask()`: Interactive selection when multiple matches found
  - `confirmTask()`: Confirmation prompt for single partial match
  - Handles exact matches, partial matches, and multiple matches
- **Interactive mode** (lines 205-270): Enhanced list selection when no list name provided
  - Shows formatted table with borders and colors
  - **Dynamic terminal width detection**: Borders adapt to terminal size (40-100 chars)
  - Cross-platform terminal size detection using `golang.org/x/term`
  - Displays task count for each list (e.g., "5 tasks")
  - Shows list descriptions in gray text
  - Allows cancellation with "0"
  - Clear visual hierarchy with cyan borders and bold list names
- **Shell completion** (lines 156-185):
  - First arg: completes to list names
  - Second arg: completes to actions (get/add/update/complete - full names only)
  - Third arg: no completion (user enters task summary)
- **Filter building**: Constructs `TaskFilter` from CLI flags with abbreviation support (T/D/P/C)

### Data Models

#### Task (backend/taskManager.go:135-148)
Follows iCalendar VTODO spec:
- `UID`, `Summary`, `Description`
- `Status`: NEEDS-ACTION, IN-PROCESS, COMPLETED, CANCELLED
- `Priority`: 0-9 (0=undefined, 1=highest, 9=lowest)
- Timestamps: `Created`, `Modified`, `DueDate`, `StartDate`, `Completed`
- `Categories`: List of category tags
- Supports subtasks via `ParentUID`
- **Formatting methods**:
  - `String()` (line 150-152): Default basic format output
  - `FormatWithView(view, backend, dateFormat)` (line 154-236): Rich formatting with:
    - Status indicators with color (✓ ○ ● ✗)
    - Priority-based coloring (from backend)
    - Due date display with overdue/upcoming highlighting
    - Description preview (truncated to 70 chars)
    - Metadata display in "all" view (created, modified, priority)

#### TaskList (backend/taskManager.go:243-250)
- Represents a calendar/list containing tasks
- Contains CalDAV-specific fields: `CTags` for sync, `Color` for UI
- **Formatting methods**:
  - `String()` (line 247-249): Renders top border with default 80-char width
  - `StringWithWidth(termWidth)` (line 251-280): Renders top border with dynamic width
  - `BottomBorder()` (line 282-284): Renders bottom border with default 80-char width
  - `BottomBorderWithWidth(termWidth)` (line 286-298): Renders bottom border with dynamic width
  - All methods use configurable width with min/max constraints (40-100 chars)

### iCalendar Parsing (backend/parseVTODOs.go)
- Manual XML and iCalendar parsing (no external parser library)
- `extractVTODOBlocks()`: Extracts VTODO blocks from CalDAV XML responses
- `parseVTODO()`: Parses individual VTODO into Task struct
- `parseICalTime()`: Handles multiple iCal date/time formats (UTC, local, date-only)
- `unescapeText()`: Handles iCalendar escape sequences (\n, \,, \;, \\)

### Subtask Support (internal/operations/subtasks.go)
The application supports hierarchical task organization with parent-child relationships:

**Features:**
- **Parent reference via -P flag**: Add subtasks using `-P "Parent Task"` or `-P "Parent/Child/Grandchild"` for deep nesting
- **Path-based task creation shorthand**: Use `gosynctasks List add "parent/child/grandchild"` or `gst Test add "parent/child/grandchild"` to automatically create the entire hierarchy
- **Auto-create missing parents**: When using path syntax, intermediate tasks are created automatically if they don't exist
- **Literal mode (-l/--literal flag)**: Disable path parsing for task summaries containing "/" (e.g., `gst Test add -l "be a good/generous person"`)
- **Path-based resolution**: Supports hierarchical paths like `"Feature X/Write code"` to reference nested tasks
- **Tree-based display**: Tasks are displayed with box-drawing characters (├─, └─, │) showing hierarchy
- **Enhanced disambiguation**: When multiple tasks match, displays hierarchical path `[Parent / Child]` for clarity

**Key Functions:**
- `CreateOrFindTaskPath()` (subtasks.go:10-64): Auto-creates hierarchical path of tasks from "parent/child/task" syntax
- `ResolveParentTask()` (subtasks.go:66+): Resolves parent reference (simple or path-based) to task UID
- `resolveParentPath()`: Handles path-based parent references like "A/B/C"
- `findTaskByParent()`: Finds task matching summary with specific parent UID
- `BuildTaskTree()`: Converts flat task list to hierarchical tree structure
- `FormatTaskTree()`: Renders tree with box-drawing characters and indentation
- `GetTaskPath()` (tasks.go): Returns hierarchical path string for a task (exported)

**Task Display:**
- Root tasks display normally
- Subtasks show with tree characters: `├─ Subtask` or `└─ Last subtask`
- Multi-line task output preserves tree structure with continuation characters
- Unlimited nesting depth supported
- Preserves status symbols, priority colors, and metadata display

**Parent Resolution Logic:**
1. Simple reference ("Parent") → Direct search by summary
2. Path reference ("A/B") → Walk hierarchy level by level
3. Multiple matches → Interactive selection with hierarchical paths shown
4. Single partial match → Confirmation prompt
5. Exact match → Proceed immediately

## Custom Views System

The application includes a comprehensive custom views system that allows users to define how tasks are displayed with customizable field selection, ordering, formatting, and filtering.

### Overview

- **Storage**: Views are stored as YAML files in `~/.config/gosynctasks/views/`
- **Built-in views**: `basic` and `all` are hardcoded legacy views
- **Custom views**: Users can create, edit, and delete custom views
- **View rendering**: Uses the `ViewRenderer` class with field formatters
- **Hierarchical support**: Custom views support task tree display with parent-child relationships
- **Filtering**: Views can define default filters (status, priority, tags, dates)
- **Sorting**: Views can specify sort order by any task field

### View Configuration Structure

Views are defined in YAML with the following structure:

```yaml
name: myview
description: High priority tasks with due dates
fields:
  - name: status
    format: symbol
    show: true
    color: true
  - name: summary
    format: full
    show: true
  - name: due_date
    format: relative
    show: true
    label: Due
  - name: priority
    format: number
    show: true
    color: true
filters:
  status: [NEEDS-ACTION, IN-PROCESS]
  priority_min: 1
  priority_max: 5
display:
  compact_mode: false
  show_border: true
  date_format: "2006-01-02"
  sort_by: due_date
  sort_order: asc
```

### Available Fields

- `status`: Task status (formats: symbol, text, emoji, short)
- `summary`: Task title (formats: full, truncate)
- `description`: Task description (formats: full, preview, truncate)
- `priority`: Priority 0-9 (formats: number, text, none)
- `due_date`: Due date (formats: full, short, relative)
- `start_date`: Start date (formats: full, short, relative)
- `created`: Creation date (formats: full, short, relative)
- `modified`: Last modified date (formats: full, short, relative)
- `completed`: Completion date (formats: full, short, relative)
- `tags`: Task categories/tags (formats: list, compact)
- `uid`: Unique identifier (formats: full, short)
- `parent`: Parent task UID (formats: full, short)

### View Filters

Views can define default filters that are applied when the view is used:

- **Status**: Filter by task status (e.g., `[NEEDS-ACTION, IN-PROCESS]`)
- **Priority**: Filter by priority range (`priority_min`, `priority_max`)
- **Tags**: Filter tasks that have all specified tags
- **Due dates**: Filter by due date range (`due_before`, `due_after`)
- **Start dates**: Filter by start date range (`start_before`, `start_after`)

Filters are applied BEFORE rendering and are combined with CLI status filters.

### View Sorting

Views can specify how tasks should be sorted:

- **sort_by**: Field to sort by (`status`, `summary`, `priority`, `due_date`, `start_date`, `created`, `modified`)
- **sort_order**: `asc` (ascending) or `desc` (descending)

Sorting is applied AFTER backend-specific sorting and view filtering.

### Architecture

**Key Components:**
- **`internal/views/types.go`** (views/types.go:7-95): Core view data structures
  - `View`: Main view configuration
  - `FieldConfig`: Individual field settings
  - `ViewFilters`: Filter criteria
  - `DisplayOptions`: Presentation settings
- **`internal/views/loader.go`**: View loading from YAML files
- **`internal/views/storage.go`**: View persistence and file management
- **`internal/views/renderer.go`** (views/renderer.go:10-247): View rendering engine
  - `NewViewRenderer()`: Creates renderer with field formatters
  - `RenderTask()`: Renders single task according to view
  - `RenderTaskHierarchical()`: Renders task with tree indentation
  - `RenderTasks()`: Renders multiple tasks
- **`internal/views/filter.go`** (views/filter.go:8-172): View filtering and sorting
  - `ApplyFilters()`: Filters tasks based on view filters
  - `ApplySort()`: Sorts tasks based on view configuration
- **`internal/views/formatters/`**: Field-specific formatters
  - `StatusFormatter`: Formats status with symbols/colors
  - `PriorityFormatter`: Formats priority with colors
  - `DateFormatter`: Formats dates (full, short, relative)
  - `SummaryFormatter`: Formats task summary (full, truncate)
  - `DescriptionFormatter`: Formats description (full, preview)
  - `TagsFormatter`: Formats categories/tags
  - `UIDFormatter`: Formats UIDs
- **`internal/views/builder/`**: Interactive view builder (Bubble Tea TUI)
- **`internal/operations/actions.go`** (operations/actions.go:386-459): Integration with task display
  - `RenderWithCustomView()`: Renders tasks using custom view with filters/sorting
  - `RenderTaskTreeWithCustomView()`: Renders hierarchical tasks with custom view

**View Resolution:**
1. Check if view name is built-in (`basic`, `all`)
2. Load view from cache (in-memory)
3. If not cached, load from `~/.config/gosynctasks/views/<name>.yaml`
4. Validate view configuration
5. Cache for future use

**Rendering Flow:**
1. Get tasks from backend with CLI status filter
2. Resolve view by name
3. Create `ViewRenderer` with view config and backend
4. Apply view-specific filters (`ApplyFilters()`)
5. Apply view-specific sorting (`ApplySort()`)
6. Build task tree for hierarchical display
7. Render each task with `RenderTaskHierarchical()`
8. Display with list borders

### CLI Integration

**View Commands:**
- `gosynctasks view list`: List all available views
- `gosynctasks view show <name>`: Display view configuration
- `gosynctasks view create <name>`: Create view interactively
- `gosynctasks view edit <name>`: Edit existing view
- `gosynctasks view delete <name>`: Delete a view
- `gosynctasks view validate <name>`: Validate view configuration

**Using Views:**
- `gosynctasks MyList -v myview`: Use custom view for task display
- `gosynctasks MyList -v all`: Use built-in "all" view
- `gosynctasks MyList`: Uses default "basic" view

**Completion:**
- The `--view` / `-v` flag has completion support
- Suggests built-in views (`basic`, `all`) plus custom view names
- Custom view names loaded from `~/.config/gosynctasks/views/`

### Testing

- **`internal/views/filter_test.go`**: Tests for filtering and sorting
  - Status filtering
  - Priority range filtering
  - Tags filtering (all required tags)
  - Date filtering (before/after)
  - Sorting by various fields
- **`internal/views/renderer_test.go`**: Tests for view rendering
- **`internal/views/loader_test.go`**: Tests for YAML loading
- **`internal/views/formatters/*_test.go`**: Tests for field formatters

### Known Limitations

- View filters use OR logic for status (task matches ANY status in list)
- View filters use AND logic for tags (task must have ALL tags)
- Date filters exclude tasks with nil dates
- Priority 0 (undefined) is sorted last when sorting by priority
- View inheritance/composition not yet implemented
- Export/import functionality not yet implemented

## Common Patterns

### Terminal Width Detection
The application uses `golang.org/x/term` for cross-platform terminal size detection:
- **Helper function**: `getTerminalWidth()` (main.go:196-203)
- **Default fallback**: Returns 80 if terminal size cannot be detected
- **Platform support**: Linux, macOS, Windows (including Wayland on Linux)
- **Width constraints**: 40-100 characters (min-max)
- **Used for**:
  - List selection interface borders
  - Task list display borders
  - Dynamic formatting based on terminal size

### Adding a New Backend
1. Create new file in `backend/` (e.g., `sqliteBackend.go`)
2. Implement the `TaskManager` interface with all required methods:
   - `GetTaskLists()`: Retrieve all task lists
   - `GetTasks(listID, filter)`: Get filtered tasks
   - `FindTasksBySummary(listID, summary)`: Search for tasks
   - `AddTask(listID, task)`: Create new task
   - `UpdateTask(listID, task)`: Update existing task
   - `SortTasks(tasks)`: Implement backend-specific sorting
   - `GetPriorityColor(priority)`: Return ANSI color codes for priorities
3. Add URL scheme case to `ConnectorConfig.TaskManager()` (backend/taskManager.go:56-67)

### Task Search and Matching
The application implements intelligent task search for update/complete operations:

**Search Algorithm** (main.go:252-350):
1. `FindTasksBySummary()` - Backend searches for tasks (exact + partial matches)
2. Separate exact and partial matches
3. Single exact match → Proceed immediately
4. Single partial match → Show task, ask for confirmation
5. Multiple matches → Display all with metadata, prompt for selection

**Implementation Pattern**:
```go
// Find task by summary
task, err := app.findTaskBySummary(listID, searchSummary)
if err != nil {
    return err  // No match or user cancelled
}

// Use the found task
err = taskManager.UpdateTask(listID, *task)
```

**User Experience**:
- Exact matches proceed without confirmation
- Partial matches show task details for verification
- Multiple matches display numbered list with "all" view (includes dates, priority)
- User can cancel at selection prompt

### Working with Status Filters
Always use the translation functions when working between app and CalDAV statuses:
- Use `StatusStringTranslateToStandardStatus()` before sending to CalDAV backend
- Use `StatusStringTranslateToAppStatus()` when displaying to user

### HTTP Client Configuration
The NextcloudBackend uses a customized HTTP client:
- Lazy initialization via `getClient()` method (backend/nextcloudBackend.go:23-36)
- `InsecureSkipVerify` configurable via `ConnectorConfig.InsecureSkipVerify` (warns when enabled)
- Connection pooling: max 10 idle connections, 2 per host
- 30-second timeout for requests
- Credentials extracted via `getUsername()`, `getPassword()`, `getBaseURL()` helper methods

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
- FileBackend is a placeholder (not implemented - only stubs)
- FindTasksBySummary uses client-side filtering (could optimize with CalDAV text-match queries)
- Task list cache doesn't have expiration or CTag-based invalidation
- No config package tests yet (needs backup/restore functionality)
- No ETag support for optimistic locking (prevents race conditions in multi-user scenarios)
- **TODO:** Investigate add action status flag - may not be applying -S flag correctly

## SQLite Sync Implementation ✅

The application now features a complete **bidirectional synchronization system** with offline mode support. The sync layer enables working with tasks offline and automatically synchronizing with remote backends (Nextcloud) when connectivity is restored.

### Architecture

```
┌─────────────────────────────────────┐
│      CLI Command                    │
│  gosynctasks add "task"             │
└──────────────┬──────────────────────┘
               │
               ▼ (uses default_backend=local)
┌─────────────────────────────────────┐
│    SQLiteBackend                    │
│  - CRUD operations                  │
│  - Sync metadata tracking           │
│  - Operation queueing               │
└──────────────┬──────────────────────┘
               │
               ▼ (sync command)
┌─────────────────────────────────────┐
│       SyncManager                   │
│  - Pull phase (remote → local)      │
│  - Push phase (local → remote)      │
│  - Conflict detection               │
│  - Conflict resolution              │
└──────────┬────────────┬─────────────┘
           │            │
           ▼            ▼
  ┌────────────┐  ┌──────────────┐
  │  SQLite    │  │   Nextcloud  │
  │  Backend   │  │   Backend    │
  │  (local)   │  │   (remote)   │
  └────────────┘  └──────────────┘
```

### Key Components

#### 1. Database Schema (backend/schema.go)

Complete schema with sync metadata tracking:

**tasks** table: Main task data (follows iCalendar VTODO format)
- `id`, `list_id`, `summary`, `description`, `status`, `priority`
- Timestamps: `created_at`, `modified_at`, `due_date`, `start_date`, `completed_at`
- Hierarchy: `parent_uid` (for subtasks)
- Categories: `categories` (comma-separated tags)

**sync_metadata** table: Sync state per task
- `task_uid` (PRIMARY KEY, FK to tasks.id ON DELETE CASCADE)
- Remote state: `remote_etag`, `last_synced_at`, `remote_modified_at`
- Local flags: `locally_modified`, `locally_deleted`, `local_modified_at`

**list_sync_metadata** table: Sync state per list
- `list_id` (PRIMARY KEY)
- Sync tracking: `last_ctag`, `last_full_sync`, `sync_token`
- Metadata: `list_name`, `list_color`, `created_at`, `modified_at`

**sync_queue** table: Operations pending sync
- `id` (AUTOINCREMENT PRIMARY KEY)
- Operation: `task_uid`, `list_id`, `operation` (create/update/delete)
- Error tracking: `retry_count`, `last_error`
- Unique constraint: `UNIQUE(task_uid, operation)`

**schema_version** table: Migration tracking
- `version` (PRIMARY KEY), `applied_at`

**Indexes**: Optimized for common queries
- `idx_tasks_list_id`, `idx_tasks_status`, `idx_tasks_due_date`
- `idx_tasks_parent_uid`, `idx_tasks_priority`
- `idx_sync_metadata_locally_modified`, `idx_sync_metadata_locally_deleted`
- `idx_sync_queue_operation`, `idx_sync_queue_created_at`

#### 2. SQLiteBackend (backend/sqliteBackend.go)

Full `TaskManager` implementation with sync support:

**CRUD Operations:**
- `GetTaskLists()`, `GetTasks()`, `FindTasksBySummary()`
- `AddTask()`, `UpdateTask()`, `DeleteTask()`
- `CreateTaskList()`, `DeleteTaskList()`, `RenameTaskList()`

**Sync-Specific Methods:**
- `MarkLocallyModified(taskUID)`: Marks task for sync
- `MarkLocallyDeleted(taskUID)`: Marks task for deletion sync
- `GetLocallyModifiedTasks()`: Retrieves tasks needing sync
- `GetPendingSyncOperations()`: Returns queued operations
- `ClearSyncFlags(taskUID)`: Clears modification flags
- `UpdateSyncMetadata()`: Updates remote state tracking
- `RemoveSyncOperation()`: Removes operation from queue

**Features:**
- Transactional operations (rollback on failure)
- Foreign key constraint enforcement
- Automatic UID generation
- Status translation (app ↔ CalDAV)
- Efficient filtering and searching

#### 3. SyncManager (backend/syncManager.go)

Coordinates bidirectional synchronization:

**Core Methods:**
- `Sync()`: Performs bidirectional sync (pull + push)
- `FullSync()`: Forces complete re-sync (ignores CTags)
- `GetSyncStats()`: Returns sync statistics

**Sync Algorithm (Pull Phase - backend/syncManager.go:81-228):**
1. Get all remote task lists
2. For each list:
   - Check CTag (has list changed?)
   - If changed or new:
     - Create/update list metadata locally
     - Fetch all remote tasks
     - **Sort by hierarchy** (parents before children) - critical for FK constraints
     - For each remote task:
       - If doesn't exist locally → insert with sync metadata
       - If exists but not locally modified → update
       - If exists and locally modified → **CONFLICT** (resolve)
     - Delete tasks missing from remote (unless locally modified)

**Sync Algorithm (Push Phase - backend/syncManager.go:236-305):**
1. Get pending sync operations from queue
2. For each operation:
   - Check retry count (skip if > 5)
   - Execute operation (create/update/delete on remote)
   - On success: remove from queue, clear sync flags
   - On failure: increment retry count, log error, apply exponential backoff

**Conflict Resolution (backend/syncManager.go:402-504):**

Four strategies implemented:

1. **ServerWins (default)**: Remote always wins
   - Updates local with remote version
   - Clears local modification flag
   - Safest - prevents data loss on server

2. **LocalWins**: Local always wins
   - Keeps local version for push
   - Updates sync metadata with remote timestamp
   - Use with caution - can overwrite others' changes

3. **Merge**: Intelligent field-level merge
   - Summary: Remote wins (significant change)
   - Description: Keep local if remote is empty
   - Priority: Use higher priority (lower number)
   - Categories: Union of both sets
   - Dates: Use most recent
   - Marks for push to propagate merge

4. **KeepBoth**: Creates duplicate
   - Updates original with remote version
   - Creates copy with "(local copy)" suffix
   - Both tasks preserved

**Error Handling:**
- Exponential backoff: 2^retryCount seconds (max 5 minutes)
- Max 5 retries per operation
- Errors logged in `sync_queue.last_error`
- Continues sync on individual operation failures

**Hierarchical Task Sorting (backend/syncManager.go:673-720):**
- Critical function: `sortTasksByHierarchy()`
- Ensures parent tasks are synced before children
- Prevents foreign key constraint violations
- Handles orphaned tasks (parent doesn't exist)
- Depth-first traversal of task tree

#### 4. Database Layer (backend/database.go)

Database management and helpers:

**Database struct:**
- Wraps `*sql.DB` with helper methods
- Tracks database path
- Schema initialization and migrations

**Key Methods:**
- `InitDatabase(customPath)`: Creates DB with schema
- `getDatabasePath()`: XDG-compliant path resolution
- `GetSchemaVersion()`: Returns current schema version
- `GetStats()`: Returns database statistics
- `Vacuum()`: Compacts database

**Path Resolution Priority:**
1. Custom path from config (`db_path`)
2. `$XDG_DATA_HOME/gosynctasks/tasks.db`
3. `~/.local/share/gosynctasks/tasks.db`

**Statistics (backend/database.go:131-183):**
- Task count, list count
- Locally modified count
- Pending sync operations
- Database size
- Last sync timestamp

#### 5. CLI Integration (cmd/gosynctasks/sync.go)

Complete sync command suite:

**Main Commands:**
- `gosynctasks sync`: Perform sync
- `gosynctasks sync --full`: Force full re-sync
- `gosynctasks sync --dry-run`: Preview (not yet implemented)
- `gosynctasks sync status`: Show sync status
- `gosynctasks sync queue`: View pending operations
- `gosynctasks sync queue clear [--failed]`: Clear operations
- `gosynctasks sync queue retry`: Retry failed operations

**Offline Detection (cmd/gosynctasks/sync.go:357-392):**
- Attempts remote connection with 5-second timeout
- Detects network errors, DNS errors, connection refused, timeouts
- Shows user-friendly error messages
- Gracefully degrades to offline mode

**Sync Status Display (cmd/gosynctasks/sync.go:107-167):**
```
=== Sync Status ===
Connection: Online / Offline (reason)
Local tasks: 42
Local lists: 3
Pending operations: 5
Locally modified: 2
Strategy: server_wins
Last sync: 5 minutes ago
```

### Configuration

**Minimal sync config:**
```json
{
  "backends": {
    "local": {
      "type": "sqlite",
      "enabled": true,
      "db_path": ""
    },
    "nextcloud": {
      "type": "nextcloud",
      "enabled": true,
      "url": "nextcloud://user:pass@server.com"
    }
  },
  "sync": {
    "enabled": true,
    "local_backend": "local",
    "remote_backend": "nextcloud",
    "conflict_resolution": "server_wins"
  },
  "default_backend": "local"
}
```

**Sync options:**
- `enabled`: Enable/disable sync (default: false)
- `local_backend`: SQLite backend name
- `remote_backend`: Remote backend name
- `conflict_resolution`: Strategy (server_wins, local_wins, merge, keep_both)
- `auto_sync`: Auto-sync on operations (not yet implemented)
- `sync_interval`: Seconds between auto-syncs (default: 300)

### Testing

Comprehensive test coverage across all components:

**Unit Tests:**
- `backend/schema_test.go`: Schema creation, indexes, constraints (17 tests)
- `backend/sqliteBackend_test.go`: CRUD operations, filtering, sync flags (92 tests)
- `backend/syncManager_test.go`: Pull/push, conflicts, retries (18 tests)

**Integration Tests (backend/integration_test.go):**
1. **TestBasicSyncWorkflow**: remote→local→modify→remote (end-to-end)
2. **TestOfflineModeWorkflow**: Offline operations, queue, sync when online
3. **TestConflictResolutionScenarios**: All 4 strategies
4. **TestLargeDatasetPerformance**: 1000+ tasks in <30s
5. **TestErrorRecoveryWithRetry**: Network errors, retry logic
6. **TestConcurrentSyncOperations**: Race condition detection
7. **TestHierarchicalTaskSync**: Parent-child task sync

**Benchmark Tests (backend/sync_bench_test.go):**
- `BenchmarkSyncPull`: Pull performance (10, 100, 1000 tasks)
- `BenchmarkSyncPush`: Push performance
- `BenchmarkConflictResolution`: Resolution strategy performance
- `BenchmarkDatabaseOperations`: CRUD benchmarks
- `BenchmarkSyncQueue`: Queue operations
- `BenchmarkHierarchicalTaskSorting`: Sorting performance

**Coverage Goals:**
- Unit tests: >85%
- Integration tests: >80%
- Overall coverage: >80%

### Performance

**Optimizations:**
- CTag-based change detection (only sync changed lists)
- Indexed queries (list_id, status, due_date, parent_uid, priority)
- Batch operations (transaction per list)
- Efficient hierarchical sorting (O(n log n))

**Benchmarks (1000 tasks):**
- Full sync: <30 seconds
- Pull phase: <15 seconds
- Push phase: <15 seconds
- Conflict resolution: <100ms per task

### Key Implementation Details

**Transaction Usage:**
- All CRUD operations use transactions
- Rollback on any error
- Atomic updates to tasks + sync_metadata + sync_queue

**Foreign Key Handling:**
- Foreign keys enforced (`PRAGMA foreign_keys = ON`)
- CASCADE delete for sync_metadata
- SET NULL for task parent_uid
- Hierarchical sort prevents FK constraint violations

**Timestamp Handling:**
- Unix timestamps for storage (INTEGER)
- Automatic conversion to/from Go `time.Time`
- `time.IsZero()` for null detection
- `sql.NullInt64` for nullable fields

**UID Generation:**
- Format: `task-{timestamp}-{random8}`
- Example: `task-1700000000-a3b2c1d4`
- Unique and sortable
- Used for tasks and lists

### Documentation

- **User Guide**: [SYNC_GUIDE.md](SYNC_GUIDE.md) - Complete sync documentation
- **README**: [README.md](README.md#synchronization) - Quick start and examples
- **Code Documentation**: Comprehensive godoc comments on all public APIs

### Known Limitations

- Auto-sync not yet implemented (manual `gosynctasks sync` required)
- Dry-run preview not yet implemented
- Sync-token (incremental sync) not yet implemented (uses full fetch)
- No sync per-list (syncs all lists)
- No encryption at rest for local SQLite database

### Future Enhancements

- Auto-sync with configurable intervals
- Incremental sync using CalDAV sync-tokens
- Selective sync (choose which lists to sync)
- Sync hooks (pre/post sync scripts)
- Encrypted local database
- Sync conflict UI for interactive resolution
- Background sync daemon
- Sync status notifications

### Error Handling

The application implements comprehensive error handling for backend operations:

**Connection and Authentication Errors:**
- HTTP status codes are checked in all backend operations (backend/nextcloudBackend.go)
- 401/403 errors return clear "Authentication failed" messages with guidance
- 404 errors indicate URL configuration issues
- Backend errors are propagated immediately via `BackendError` type (backend/errors.go)
- Error checking in app.Run() (internal/app/app.go:52-75) stops execution for auth failures

**User-Facing Error Messages:**
- Connection failures show actionable guidance to check URL/credentials
- Missing list names show available alternatives
- Helper function `formatAvailableLists()` provides list of valid options

**Testing:**
- Test coverage for authentication failures (backend/nextcloudBackend_test.go:179-215)
- Validates proper error type conversion and helpful error messages
