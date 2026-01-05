# gosynctasks

[![CI](https://github.com/DeepReef11/gosynctasks/actions/workflows/ci.yml/badge.svg)](https://github.com/DeepReef11/gosynctasks/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/DeepReef11/gosynctasks/branch/main/graph/badge.svg)](https://codecov.io/gh/DeepReef11/gosynctasks)
[![Go Report Card](https://goreportcard.com/badge/github.com/DeepReef11/gosynctasks)](https://goreportcard.com/report/github.com/DeepReef11/gosynctasks)
[![License: BSD-2](https://img.shields.io/badge/License-BSD--2--Clause-darkred)](https://opensource.org/license/bsd-2-clause)

##  Manage your tasks seamlessly from the comfort of your terminal

A fast, flexible, multi-backend task synchronization tool written in Go. Manage your tasks across different storage backends including Nextcloud CalDAV, Todoist, Git repositories with Markdown files, and local database.

## Features

- **Multi-Backend Support**: Work with multiple task storage backends simultaneously
- **Todoist Integration**: Full REST API v2 support with secure credential storage
- **Offline Sync**: Local SQLite cache with bidirectional synchronization to remote backends
- **Git/Markdown Backend**: Manage tasks directly in markdown files within git repositories
- **Nextcloud CalDAV**: Full CRUD support for Nextcloud Tasks
- **Auto-Detection**: Automatically detect and use the appropriate backend based on context
- **Secure Credentials**: Keyring, environment variables, or config file support
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

```bash
# Launch gosynctasks once to create the config file sample
gosynctasks 
# Configure sync (see SQLite Backend configuration above)

# Set credentials
gosynctasks credentials set backend-name my-user --prompt

# Initial sync
gosynctasks sync

gosynctasks list create "Work"
# Work normally - changes sync automatically with auto_sync: true
gosynctasks Work add "Task created"

# Manual sync if needed
gosynctasks sync
gosynctasks sync status
gosynctasks sync queue
```


**For detailed sync documentation, see [SYNC_GUIDE.md](SYNC_GUIDE.md)**

#### Multi-Backend Configuration

Here is a basic configuration using local sync, Nextcloud as a remote and automatically detect git repo TODO.md file:

```yaml
backends:

  # Use this line to name the backend 
  example-backend-name:
    type: nextcloud
    enabled: true
    host: "nextcloud.example.com"
    username: "myuser"
  # Docker Test Nextcloud
  nextcloud-test:
    type: nextcloud
    enabled: true
    host: "localhost:8080"
    username: "admin"
    allow_http: true
    suppress_http_warning: true
  # Same Docker Test Nextcloud config using unsafe url
  nextcloud-unsafe:
    type: nextcloud
    enabled: true
    url: nextcloud://admin:admin123@localhost:8080
    insecure_skip_verify: false
  # Todoist use token as username in keyring credentials
  todoist:
    type: todoist
    enabled: true
  git:
    type: git
    enabled: true
    file: TODO.md
    auto_detect: true
    auto_commit: false
  sqlite:
    type: sqlite
    enabled: true
    db_path: ""  # Empty = use XDG default (~/.local/share/gosynctasks/tasks.db)

sync:
  enabled: true
  local_backend: sqlite
  remote_backend: nextcloud-prod
  conflict_resolution: server_wins
  auto_sync: true        # Enable background daemon sync
  sync_interval: 5       # Minutes before data considered stale

default_backend: nextcloud-prod
auto_detect_backend: true
backend_priority:
  - git
  - nextcloud-prod
ui: cli
```

## Credentials Storage

###  System Keyring (Recommended)

Store credentials securely in your OS keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service):

**For Username/Password backends (Nextcloud):**
```bash
# Store credentials securely (interactive password prompt)
gosynctasks credentials set nextcloud myuser --prompt

# Verify credentials
gosynctasks credentials get nextcloud myuser
```

**Config example:**
```yaml
nextcloud:
  type: nextcloud
  enabled: true
  host: "nextcloud.example.com"
  username: "myuser"
  # Password retrieved from keyring automatically
```

**For API Token backends (Todoist):**
```bash
# Store API token as "password" with "token" as username hint
gosynctasks credentials set todoist token --prompt
# Enter your API token when prompted

# Verify
gosynctasks credentials get todoist token
```

**Config example:**
```yaml
todoist:
  type: todoist
  enabled: true
  username: "token"  # Username hint for keyring lookup
  # API token retrieved from keyring automatically
```

### Environment Variables

For CI/CD or containerized environments:

**Nextcloud:**
```bash
export GOSYNCTASKS_NEXTCLOUD_USERNAME=myuser
export GOSYNCTASKS_NEXTCLOUD_PASSWORD=secret
export GOSYNCTASKS_NEXTCLOUD_HOST=nextcloud.example.com
```

**Todoist:**
```bash
export GOSYNCTASKS_TODOIST_PASSWORD="your-api-token-here"
```

### Credentials in Config (Not Recommended)

**Nextcloud URL format:**
```yaml
nextcloud:
  type: nextcloud
  enabled: true
  url: "nextcloud://username:password@nextcloud.example.com"
```

**Todoist api_token field:**
```yaml
todoist:
  type: todoist
  enabled: true
  api_token: "your-api-token-here"
```

⚠️ **Warning:** Plain text credentials in config files are not recommended for production use.

**Credential Priority:** Keyring > Environment Variables > Config File

For more details, see [SECURITY.md](SECURITY.md).

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

```yaml
nextcloud:
  type: nextcloud
  enabled: true
  host: "nextcloud.example.com"
  username: "myuser"
  # Password retrieved from keyring automatically
```

### SQLite Backend

Local SQLite database that can be used for offline synchronization with remote backends to get fast operations.

**Features:**
- ✅ Offline mode - work without network connectivity
- ✅ Bidirectional sync with Nextcloud
- ✅ **Auto-sync with background daemon** - instant operations, sync happens after
- ✅ Conflict resolution (4 strategies)
- ✅ Operation queuing and retry logic
- ✅ Efficient sync with CTags/ETags

**Configuration:**
```yaml
backends:
  sqlite:
    type: sqlite
    enabled: true
    db_path: ""  # Empty = use XDG default (~/.local/share/gosynctasks/tasks.db)
  nextcloud:
    type: nextcloud
    enabled: true
    url: nextcloud://user:pass@server.com

sync:
  enabled: true
  local_backend: sqlite
  remote_backend: nextcloud
  conflict_resolution: server_wins
  auto_sync: true        # Enable background daemon sync
  sync_interval: 5       # Minutes before data considered stale

backend_priority:
  - nextcloud  # Local backend auto-selected when sync enabled
```

**Note:** When `sync.enabled = true`, the CLI automatically uses `sqlite` for all operations and the `backend_priority` only applies when sync is disabled.

### Todoist Backend

Cloud-based task management service with full API integration. Perfect for cross-platform sync and mobile access.

**Features:**
- ✅ Full CRUD operations via Todoist REST API v2
- ✅ Projects (mapped to task lists)
- ✅ Tasks with priorities, due dates, and labels
- ✅ Subtasks support (parent-child relationships)
- ✅ Secure credential storage (keyring, environment, or config)
- ✅ Smart priority mapping (Todoist 1-4 ↔ gosynctasks 0-9)

**Configuration:**

```yaml
backends:
  todoist:
    type: todoist
    enabled: true
    username: "token"  # Username hint for keyring (use "token" for API keys)
    # API token retrieved from keyring
```

**Credential Priority:** Keyring > Environment Variables > Config File

**Option 1: Keyring (Recommended)**
```bash
# Get API token from https://todoist.com/app/settings/integrations
gosynctasks credentials set todoist token --prompt
# Enter your API token when prompted
```

**Option 2: Environment Variables**
```bash
export GOSYNCTASKS_TODOIST_PASSWORD="your-api-token-here"
```

**Option 3: Config File (Less Secure)**
```yaml
todoist:
  type: todoist
  enabled: true
  api_token: "your-api-token-here"
```

**Data Mapping:**
- **Priority**: Todoist 4 (urgent) → gosynctasks 1 (highest)
- **Status**: is_completed → TODO/DONE (PROCESSING/CANCELLED via labels)
- **Projects**: Todoist projects → gosynctasks task lists
- **Subtasks**: parent_id → ParentUID

**Get your API token:** https://todoist.com/app/settings/integrations

### File Backend

Local file-based storage (work in progress).

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


## Configuration Examples

See [docs/config-examples/README.md](docs/config-examples/README.md) for detailed examples.

### Quick Examples

**Git + Nextcloud (auto-detect):**
```yaml
backends:
  git:
    type: git
    file: TODO.md
    auto_detect: true
  nextcloud:
    type: nextcloud
    url: nextcloud://user:pass@server.com

default_backend: nextcloud
auto_detect_backend: true
backend_priority: 
  - git
  - nextcloud

```

**Nextcloud with Sync:**
```yaml
backends:
  sqlite:
    type: sqlite
    enabled: true
  nextcloud:
    type: nextcloud
    enabled: true
    url: nextcloud://user:pass@server.com

sync:
  enabled: true
  auto_sync: true
  local_backend: sqlite
  remote_backend: nextcloud
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](.github/CONTRIBUTING.md) for detailed guidelines.

## Roadmap

### Completed ✅
- Multi-backend support (Nextcloud, Git, SQLite, Todoist)
- **Todoist backend** - Full REST API v2 integration with secure credential storage
- Git/Markdown backend with auto-detection
- Backend auto-detection and priority selection
- Automatic config migration
- **Comprehensive CI/CD pipeline** with automated releases
- **Nextcloud sync integration tests** with real CalDAV server
- SQLite sync layer with offline mode
- Conflict resolution (4 strategies: server_wins, local_wins, merge, keep_both)
- Bidirectional synchronization with etag handling
- **Auto-sync with background daemon** - instant operations
- **Secure credential storage** - Keyring, environment variables, config file support
- Path expansion (`$HOME`, `~`) in config files
- Custom views with plugin formatters
- Hierarchical tasks (subtasks)
- Interactive list and task selection

### In Progress 🚧
- File backend implementation
- Enhanced documentation

### Planned 📋
- Cross-backend task migration
- More cloud service backends (Google Tasks, Microsoft To Do)
- Documentation website

## Support

For issues, questions, or contributions:
- Open an issue on GitHub
