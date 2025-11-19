# ROADMAP.md

This document outlines planned future enhancements and features for gosynctasks. These are aspirational goals that may change based on project priorities and community feedback.

## Current Status

All major features have been implemented:
- ✅ Multi-backend support (Nextcloud, SQLite, File)
- ✅ Bidirectional sync with offline mode
- ✅ Custom views system with plugin formatters
- ✅ Subtask hierarchy support
- ✅ Interactive view builder
- ✅ List management (create, delete, trash, restore)

## Open Issues

For current open issues, see: https://github.com/DeepReef11/gosynctasks/issues

Notable documentation issue:
- **Issue #82**: Code review findings and positive patterns - Phase 1 complete, tracks ongoing code quality improvements

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
