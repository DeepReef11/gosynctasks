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

### Completed Features
- ✓ Full CRUD operations for tasks (Create, Read, Update, Delete)
- ✓ Task search with exact/partial/multiple match handling
- ✓ Action-based CLI with abbreviations (get/g, add/a, update/u, complete/c, delete/d)
- ✓ Task list caching for faster completion
- ✓ Rich task formatting with colors and status indicators
- ✓ Priority-based sorting and coloring
- ✓ Backend-specific status handling (each backend manages its own status format)
- ✓ Custom error types for structured error handling
- ✓ Comprehensive test coverage for backend methods (PR #57)
- ✓ Full godoc documentation for public APIs
- ✓ Clear error messages for connection and authentication failures (with actionable guidance)
- ✓ **Subtask support** (PR #54, #50) - Hierarchical task organization with parent-child relationships
  - Parent reference via `-P` flag
  - Path-based task creation shorthand (`parent/child/grandchild`)
  - Literal mode (`-l/--literal`) to disable path parsing
  - Tree-based display with box-drawing characters
  - Enhanced disambiguation with hierarchical paths
- ✓ **List management commands** (PR #51):
  - `list create` - Create new task lists
  - `list delete` - Delete task lists
  - `list rename` - Rename task lists
  - `list info` - Show detailed list information
- ✓ **Interactive view builder** (PR #45) - Create custom task views with interactive UI
- ✓ **Docker-based test environment** (PR #35) - Nextcloud test server for backend testing

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

### Recent Fixes

- [x] **Fixed misleading error message for connection failures** (2025-11-11)
  - **Issue**: Wrong credentials resulted in "list 'tasks' not found" instead of helpful error
  - **Fix**: Added HTTP status checking in GetTaskLists(), structured error propagation, improved error messages
  - **Commit**: a2b3141 - "fix: Improve error messages for connection and authentication failures"
  - **Files changed**:
    - backend/nextcloudBackend.go (HTTP status checking)
    - internal/app/app.go (error propagation)
    - internal/operations/lists.go (improved messages, formatAvailableLists helper)
    - backend/nextcloudBackend_test.go (test coverage)

### Known Issues

## Future Work: Multi-Backend Support & Git Backend

### Overview

Implement support for multiple backends with auto-detection and explicit selection. This allows gosynctasks to work with different storage backends (Nextcloud, Git/Markdown files, local files) and automatically detect the appropriate backend based on context.

### Goals

1. **Multiple Backend Support**: Configure and use multiple backends simultaneously
2. **Git/Markdown Backend**: Manage tasks directly in markdown files within git repositories
3. **Auto-Detection**: Automatically select backend based on current context
4. **Backward Compatibility**: Existing configurations continue to work
5. **Flexible Configuration**: Backend-specific settings organized logically

### Proposed Config Structure

**Current Structure:**
```json
{
  "connector": {
    "url": "nextcloud://user:pass@host"
  },
  "ui": "cli"
}
```

**New Structure:**
```json
{
  "backends": {
    "nextcloud": {
      "type": "nextcloud",
      "enabled": true,
      "url": "nextcloud://user:pass@host",
      "insecure_skip_verify": false
    },
    "git": {
      "type": "git",
      "enabled": true,
      "file": "TODO.md",
      "auto_detect": true,
      "fallback_files": ["todo.md", ".gosynctasks.md"],
      "auto_commit": false
    }
  },
  "default_backend": "nextcloud",
  "auto_detect_backend": true,
  "backend_priority": ["git", "nextcloud"],
  "ui": "cli"
}
```

### Backend Selection Priority

```
1. CLI flag: --backend <name>
2. Auto-detection (if auto_detect_backend: true)
   ├─> Git backend: Check for git repo + TODO.md with marker
   ├─> File backend: Check for local files
   └─> Use first detected from backend_priority
3. Default backend from config
4. First enabled backend
```

### Git Backend Specification

#### Detection Criteria

All must be met:
1. ✅ Current directory is inside a git repository
2. ✅ Configured markdown file exists (default: `TODO.md`)
3. ✅ File contains gosynctasks marker

#### Marker Format

**HTML Comment (invisible in rendered markdown):**
```markdown
<!-- gosynctasks:enabled -->

# My Project Tasks

## Work Tasks
- [ ] Review PR #123 @priority:1 @due:2025-01-20
- [x] Deploy to staging @completed:2025-01-10
- [ ] Update documentation
```

**Why HTML Comment:**
- Invisible in rendered markdown (GitHub, GitLab, etc.)
- Simple to detect and parse
- Doesn't interfere with existing markdown tools
- Can include additional config

#### Markdown Task Format

**Status Mapping:**
```markdown
- [ ] Task     → NEEDS-ACTION (TODO)
- [x] Task     → COMPLETED (DONE)
- [>] Task     → IN-PROCESS (PROCESSING)
- [-] Task     → CANCELLED
```

**Metadata Tags:**
```markdown
- [ ] Task @priority:1 @due:2025-01-20 @uid:task-123
```

**Supported Tags:**
- `@priority:N` - Priority 1-9 (1=highest)
- `@due:YYYY-MM-DD` - Due date
- `@created:YYYY-MM-DD` - Creation date
- `@completed:YYYY-MM-DD` - Completion date
- `@uid:string` - Unique identifier (auto-generated)
- `@status:STATUS` - Explicit status override

**Task Lists:**
- Level 2 headers (`##`) define task lists
- Tasks belong to most recent header above them

**Example:**
```markdown
<!-- gosynctasks:enabled -->

## Personal
- [ ] Buy groceries @priority:5 @due:2025-01-15
- [x] Call dentist @completed:2025-01-10

## Work
- [ ] Review PR @priority:1 @uid:task-001
  Description: Check authentication logic
  and error handling.
```

### Implementation Phases

#### Phase 1: Config Restructuring (2-3 days)
- Create new config structs with `backends` map
- Implement automatic migration from old format
- Backup old config before migration
- Support both formats during transition
- Update sample config

**Files:**
- `internal/config/config.go`
- `internal/config/config.sample.json`
- `backend/taskManager.go`

#### Phase 2: Backend Selection (1-2 days)
- Create backend registry and factory pattern
- Implement `BackendSelector` with priority logic
- Add `--backend` CLI flag
- Add `--list-backends`, `--detect-backend` flags
- Implement auto-detection interface

**New Interface:**
```go
type DetectableBackend interface {
    TaskManager
    CanDetect() (bool, error)
    DetectionInfo() string
}
```

**Files:**
- `internal/config/config.go`
- `backend/taskManager.go`
- `cmd/gosynctasks/main.go`

#### Phase 3: Git Backend (4-5 days)
- Implement git repo detection (`git rev-parse`)
- Create markdown parser for tasks
- Create markdown writer (preserve formatting)
- Implement all TaskManager methods
- UID auto-generation and embedding
- Optional auto-commit feature

**Files:**
- `backend/gitBackend.go`
- `backend/markdownParser.go`
- `backend/markdownWriter.go`
- `backend/gitBackend_test.go`

**Key Methods:**
```go
type GitBackend struct {
    repoPath  string
    filePath  string
    tasks     map[string][]Task
}

func (gb *GitBackend) isGitRepo() bool
func (gb *GitBackend) findTodoFile() (string, error)
func (gb *GitBackend) hasMarker(content string) bool
func (gb *GitBackend) parseMarkdown(content string) error
func (gb *GitBackend) saveFile() error
func (gb *GitBackend) commitChanges() error
```

#### Phase 4: Testing (2-3 days)
- Config migration tests
- Backend selection tests
- Git detection tests
- Markdown parsing/writing tests
- End-to-end workflow tests
- Edge case handling

#### Phase 5: Documentation (1-2 days)
- Update CLAUDE.md
- Create migration guide
- Add config examples
- Update README.md
- Usage examples for each backend

### CLI Changes

**New Flags:**
```bash
--backend <name>      # Override backend selection
--list-backends       # Show configured backends
--detect-backend      # Show detected backend
```

**Usage Examples:**
```bash
# Use specific backend
gosynctasks --backend git MyTasks get

# Auto-detect (if inside git repo with TODO.md)
gosynctasks MyTasks add "New task"

# List configured backends
gosynctasks --list-backends
```

### Config Migration

**Automatic Process:**
1. Detect old format (has `connector` field)
2. Create backup: `config.json.backup`
3. Convert to new format
4. Preserve all settings
5. Write new config
6. Show migration message

**Backward Compatibility:**
- Support both old and new formats
- Automatic migration on first load
- No manual intervention required
- Clear migration messages

### Key Decisions

1. **Marker Format**: HTML comment `<!-- gosynctasks:enabled -->`
   - Invisible when rendered
   - Simple to detect
   - Standard markdown

2. **Default File**: Check in order:
   - `TODO.md` (primary)
   - `todo.md` (fallback)
   - `.gosynctasks.md` (explicit)
   - User-configured

3. **Metadata**: `@tag:value` format
   - Easy regex parsing
   - Visually distinct
   - No conflicts with markdown

4. **UID Format**: `@uid:task-{timestamp}-{random}`
   - Example: `task-1705860000-a3b2`
   - Unique and sortable
   - Helps debugging

5. **Auto-commit**: Optional, disabled by default
   - Users control git workflow
   - Can enable per backend

6. **Multiple Files**: Single file per backend instance
   - Configure multiple git backends for multiple files
   - Simpler mental model

### Risk Mitigation

**High Risk - Data Loss:**
- Extensive testing with real markdown files
- Create backups before writing
- Parse-write-parse validation
- File integrity checks

**High Risk - Config Migration:**
- Always backup before migration
- Validate new config
- Rollback mechanism
- Clear error messages

**Medium Risk - Git Conflicts:**
- Detect file changes before writing
- Warn about external modifications
- Require manual resolution
- Optional `--force` flag

### Success Criteria

**Must Have:**
- ✅ Multiple backends configurable
- ✅ Backend selection works (CLI, auto-detect, default)
- ✅ Config migration automatic and safe
- ✅ Git backend reads/writes markdown correctly
- ✅ All TaskManager methods implemented
- ✅ Existing configs work
- ✅ No data loss

**Should Have:**
- ✅ Auto-detection reliable
- ✅ Markdown formatting preserved
- ✅ Stable UIDs
- ✅ Comprehensive tests (>80% coverage)
- ✅ Clear documentation
- ✅ Helpful error messages

**Nice to Have:**
- ✅ Auto-commit (optional)
- ✅ Multiple files via multiple backends
- ✅ Conflict detection
- ✅ Backend status commands

### Timeline

**Total: 10-15 days (2-3 weeks)**
- Phase 1 (Config): 2-3 days
- Phase 2 (Selection): 1-2 days
- Phase 3 (Git Backend): 4-5 days
- Phase 4 (Testing): 2-3 days
- Phase 5 (Documentation): 1-2 days

### Future Enhancements

- GitHub/GitLab Issues backends
- Trello/Notion backends
- Sync between backends
- Branch-specific tasks
- Git hooks integration
- Advanced conflict resolution
- Caching layer
- Per-project config overrides
