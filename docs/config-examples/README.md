# Configuration Examples

This directory contains example configurations for different use cases. Copy the appropriate example to `~/.config/gosynctasks/config.json` and customize it for your needs.

## Available Examples

### 1. Nextcloud Only (`01-nextcloud-only.json`)

**Use case:** Connect only to a Nextcloud server for task management.

**Features:**
- Single Nextcloud backend
- No auto-detection
- Simple and straightforward

**Setup:**
```bash
cp docs/config-examples/01-nextcloud-only.json ~/.config/gosynctasks/config.json
# Edit the file to add your Nextcloud URL and credentials
```

**Configuration:**
```json
{
  "backends": {
    "nextcloud": {
      "type": "nextcloud",
      "url": "nextcloud://username:password@your-server.com"
    }
  },
  "default_backend": "nextcloud"
}
```

**Usage:**
```bash
gosynctasks MyList get
gosynctasks MyList add "New task"
```

---

### 2. Git Only (`02-git-only.json`)

**Use case:** Manage tasks in markdown files within git repositories.

**Features:**
- Single git backend
- Auto-detection enabled
- Works in any git repository with TODO.md

**Setup:**
```bash
cp docs/config-examples/02-git-only.json ~/.config/gosynctasks/config.json

# In your git repository
cd /path/to/your/repo
cat > TODO.md << 'EOF'
<!-- gosynctasks:enabled -->

## Tasks
- [ ] Your first task
EOF
git add TODO.md
git commit -m "Add TODO.md"
```

**Configuration:**
```json
{
  "backends": {
    "git": {
      "type": "git",
      "file": "TODO.md",
      "auto_detect": true
    }
  },
  "default_backend": "git",
  "auto_detect_backend": true
}
```

**Usage:**
```bash
cd /path/to/your/repo
gosynctasks Tasks add "New task"
# Automatically uses git backend
```

---

### 3. Git with Nextcloud Fallback (`03-git-with-nextcloud-fallback.json`)

**Use case:** Use git backend when in a repository, otherwise use Nextcloud.

**Features:**
- Automatic backend switching
- Git takes priority when detected
- Nextcloud as default fallback

**Setup:**
```bash
cp docs/config-examples/03-git-with-nextcloud-fallback.json ~/.config/gosynctasks/config.json
# Edit to add your Nextcloud credentials
```

**Configuration:**
```json
{
  "backends": {
    "git": {
      "type": "git",
      "file": "TODO.md",
      "auto_detect": true
    },
    "nextcloud": {
      "type": "nextcloud",
      "url": "nextcloud://user:pass@server.com"
    }
  },
  "default_backend": "nextcloud",
  "auto_detect_backend": true,
  "backend_priority": ["git", "nextcloud"]
}
```

**Behavior:**
- Inside git repo with TODO.md → Uses git backend
- Outside git repo → Uses Nextcloud backend
- Can override with `--backend` flag

**Usage:**
```bash
# In git repo - uses git
cd ~/projects/myproject
gosynctasks Tasks add "Code task"

# Outside git repo - uses Nextcloud
cd ~
gosynctasks Work add "Meeting notes"

# Explicit backend selection
gosynctasks --backend nextcloud Work add "Task"
```

---

### 4. Multiple Git Backends (`04-multiple-git-backends.json`)

**Use case:** Manage separate task lists for different projects/contexts.

**Features:**
- Multiple git backends
- Dedicated backend per project
- Auto-detection for current directory
- Explicit selection for specific projects

**Setup:**
```bash
cp docs/config-examples/04-multiple-git-backends.json ~/.config/gosynctasks/config.json
# Edit paths to match your project directories
```

**Configuration:**
```json
{
  "backends": {
    "work": {
      "type": "git",
      "file": "/home/user/work-projects/TODO.md",
      "auto_detect": false,
      "auto_commit": true
    },
    "personal": {
      "type": "git",
      "file": "/home/user/personal-projects/TODO.md",
      "auto_detect": false
    },
    "current": {
      "type": "git",
      "file": "TODO.md",
      "auto_detect": true
    }
  },
  "backend_priority": ["current", "work", "personal"]
}
```

**Usage:**
```bash
# Work on work tasks
gosynctasks --backend work "Sprint" add "Feature X"

# Personal tasks
gosynctasks --backend personal "Hobbies" add "Learn Go"

# Current directory (auto-detect)
cd ~/my-repo
gosynctasks Tasks add "Project task"
```

---

### 5. All Backends (`05-all-backends.json`)

**Use case:** Maximum flexibility with all backend types configured.

**Features:**
- Nextcloud for shared/synced tasks
- Multiple git backends for different contexts
- Auto-detection enabled
- Priority-based selection

**Setup:**
```bash
cp docs/config-examples/05-all-backends.json ~/.config/gosynctasks/config.json
# Edit all URLs and paths
```

**Configuration:**
```json
{
  "backends": {
    "nextcloud": {
      "type": "nextcloud",
      "url": "nextcloud://user:pass@server.com"
    },
    "git-main": {
      "type": "git",
      "file": "TODO.md",
      "auto_detect": true
    },
    "git-work": {
      "type": "git",
      "file": "/home/user/work/TODO.md",
      "auto_detect": false,
      "auto_commit": true
    }
  },
  "backend_priority": ["git-main", "git-work", "nextcloud"]
}
```

**Backend Selection Logic:**
1. In git repo with TODO.md → `git-main`
2. Explicit `--backend git-work` → `git-work`
3. Outside git repos → `nextcloud` (default)

**Usage:**
```bash
# Auto-detect (uses git-main if in git repo)
gosynctasks Tasks get

# Explicit backend
gosynctasks --backend nextcloud Shared add "Team task"
gosynctasks --backend git-work Sprint add "Work item"

# List available backends
gosynctasks --list-backends

# Detect current backend
gosynctasks --detect-backend
```

---

## Common Configuration Options

### Backend Types

#### Nextcloud Backend
```json
{
  "type": "nextcloud",
  "enabled": true,
  "url": "nextcloud://username:password@host[:port][/path]",
  "insecure_skip_verify": false
}
```

**URL Examples:**
- `nextcloud://user:pass@cloud.example.com`
- `nextcloud://user:pass@localhost:8080`
- `nextcloud://user:pass@server.com:443/nextcloud`

**Note:** Use `insecure_skip_verify: true` only for self-signed certificates in test environments.

#### Git Backend
```json
{
  "type": "git",
  "enabled": true,
  "file": "TODO.md",
  "auto_detect": true,
  "auto_commit": false
}
```

**Options:**
- `file`: Path to markdown file (absolute or relative to repo root)
- `auto_detect`: Enable auto-detection in git repositories
- `auto_commit`: Automatically commit changes to git

**Markdown File Format:**
```markdown
<!-- gosynctasks:enabled -->

## Task List Name
- [ ] Task summary @priority:1 @due:2025-12-31
- [x] Completed task @completed:2025-01-10
- [>] In-progress task
- [-] Cancelled task
```

### Global Settings

```json
{
  "default_backend": "backend-name",
  "auto_detect_backend": true,
  "backend_priority": ["backend1", "backend2"],
  "ui": "cli",
  "can_write_config": true
}
```

**Options:**
- `default_backend`: Fallback backend when no auto-detection match
- `auto_detect_backend`: Enable automatic backend detection
- `backend_priority`: Order for auto-detection attempts
- `ui`: User interface mode (`cli` or `tui`)
- `can_write_config`: Allow app to write config changes

---

## Migration from Old Format

If you have an old single-backend configuration, it will be automatically migrated on first run.

**Old format:**
```json
{
  "connector": {
    "url": "nextcloud://user:pass@server.com"
  },
  "ui": "cli"
}
```

**Automatically becomes:**
```json
{
  "backends": {
    "nextcloud": {
      "type": "nextcloud",
      "enabled": true,
      "url": "nextcloud://user:pass@server.com"
    }
  },
  "default_backend": "nextcloud",
  "ui": "cli"
}
```

A backup is created at `config.json.backup`.

---

## Customization Tips

### 1. Project-Specific Tasks

Use multiple git backends for different projects:
```json
{
  "backends": {
    "projectA": { "file": "/path/to/projectA/TODO.md" },
    "projectB": { "file": "/path/to/projectB/TODO.md" }
  }
}
```

### 2. Work/Personal Split

```json
{
  "backends": {
    "work-nextcloud": { "url": "nextcloud://work@company.com" },
    "personal-git": { "file": "~/personal/TODO.md" }
  }
}
```

### 3. Team + Personal

```json
{
  "backends": {
    "team": { "url": "nextcloud://team@server.com" },
    "my-tasks": { "file": "TODO.md", "auto_detect": true }
  },
  "backend_priority": ["my-tasks", "team"]
}
```

When working in your projects, tasks go to your git backend. Use `--backend team` for shared tasks.

---

## Security Considerations

### Credentials in Config

The config file contains sensitive credentials. Protect it:

```bash
# Set restrictive permissions
chmod 600 ~/.config/gosynctasks/config.json

# Don't commit to git
echo ".config/gosynctasks/config.json" >> ~/.gitignore
```

### Environment Variables (Future)

Currently credentials are in config file. Future versions may support:
- Environment variables
- System keyring integration
- OAuth tokens

### Self-Signed Certificates

Only use `insecure_skip_verify: true` for:
- Local test servers
- Development environments
- Self-signed certificates you trust

**Never** use for production with untrusted certificates.

---

## Troubleshooting

### Backend Not Detected

```bash
# Check detection
gosynctasks --detect-backend

# List available backends
gosynctasks --list-backends

# Verify config
cat ~/.config/gosynctasks/config.json
```

### Git Backend Not Working

1. Check for marker in TODO.md:
```bash
head -1 TODO.md
# Should show: <!-- gosynctasks:enabled -->
```

2. Verify git repository:
```bash
git status
```

3. Check file path in config

### Multiple Backends Conflict

Adjust `backend_priority` to control selection order:
```json
{
  "backend_priority": ["most-preferred", "second-choice", "fallback"]
}
```

---

## See Also

- [README.md](../../README.md) - Main documentation
- [TESTING.md](../../TESTING.md) - Testing guide
- [CLAUDE.md](../../CLAUDE.md) - Development documentation
