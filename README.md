# gosynctasks

A flexible, multi-backend task synchronization tool written in Go. Manage your tasks across different storage backends including Nextcloud CalDAV, Git repositories with Markdown files, and local file storage.

## Features

- **Multi-Backend Support**: Work with multiple task storage backends simultaneously
- **Git/Markdown Backend**: Manage tasks directly in markdown files within git repositories
- **Nextcloud CalDAV**: Full CRUD support for Nextcloud Tasks
- **Auto-Detection**: Automatically detect and use the appropriate backend based on context
- **Flexible CLI**: Intuitive command-line interface with completion support
- **Hierarchical Tasks**: Support for subtasks and parent-child relationships
- **Custom Views**: Create custom task views with filtering, sorting, and custom formatting
- **Task Filtering**: Filter by status, priority, tags, and dates
- **Interactive Mode**: User-friendly interactive list and task selection

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

## Development

### Building

```bash
# Build
go build -o gosynctasks ./cmd/gosynctasks

# Run directly
go run ./cmd/gosynctasks

# Run tests
go test ./...

# Run specific tests
go test ./backend
go test ./internal/config
```

### Testing

See [TESTING.md](TESTING.md) for detailed testing procedures including:
- Manual feature testing workflows
- Git backend testing
- Multi-backend testing
- Docker test environment setup

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

Contributions are welcome! Please see [CLAUDE.md](CLAUDE.md) for development guidelines and architecture documentation.

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test ./backend -v
go test ./internal/config -v

# Integration tests
go test ./backend -run Integration
go test ./cmd/gosynctasks -run CLI
```

## License

[Add your license here]

## Roadmap

### Completed ✅
- Multi-backend support
- Git/Markdown backend
- Backend auto-detection
- Config migration
- Comprehensive testing

### In Progress 🚧
- File backend implementation
- SQLite sync layer

### Planned 📋
- GitHub/GitLab Issues backends
- Trello/Notion integration
- Cross-backend sync
- Mobile companion app
- Web UI

## Support

For issues, questions, or contributions:
- Open an issue on GitHub
- See [CLAUDE.md](CLAUDE.md) for development guidelines
- Check [TESTING.md](TESTING.md) for testing procedures

---

Built with ❤️ using Go and the Cobra CLI framework.
