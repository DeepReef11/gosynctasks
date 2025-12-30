# Configuration Examples

Example configurations for different use cases. Copy to `~/.config/gosynctasks/config.yaml` and customize.

## Quick Examples

### 1. Nextcloud Only

**Use case:** Single Nextcloud backend

```yaml
backends:
  nextcloud:
    type: nextcloud
    url: nextcloud://username:password@your-server.com

default_backend: nextcloud
```

### 2. Git Only

**Use case:** Markdown tasks in git repositories

```yaml
backends:
  git:
    type: git
    file: TODO.md
    auto_detect: true

default_backend: git
auto_detect_backend: true
```

### 3. Git + Nextcloud Fallback

**Use case:** Git when in repo, Nextcloud otherwise

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
backend_priority: [git, nextcloud]
```

### 4. Offline Sync with Nextcloud

**Use case:** Work offline, sync when online

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
  local_backend: sqlite
  remote_backend: nextcloud
  conflict_resolution: server_wins
  auto_sync: true
```

## Backend Reference

### Nextcloud
```yaml
backends:
  nextcloud:
    type: nextcloud
    url: nextcloud://username:password@host[:port][/path]
    insecure_skip_verify: false
```

### Git
```yaml
backends:
  git:
    type: git
    file: TODO.md
    auto_detect: true
    auto_commit: false
```

Markdown format:
```markdown
<!-- gosynctasks:enabled -->
## Tasks
- [ ] Task @priority:1 @due:2025-12-31
- [x] Completed @completed:2025-01-10
```

### SQLite
```yaml
backends:
  sqlite:
    type: sqlite
    enabled: true
    db_path: ""  # Empty = use XDG default
```

## Security

Protect your config file:
```bash
chmod 600 ~/.config/gosynctasks/config.yaml
```

## Migration

Old JSON configs are automatically migrated to YAML on first run. Backup created.

## See Also

- [README.md](../../README.md) - Main documentation
- [SYNC_GUIDE.md](../../SYNC_GUIDE.md) - Sync documentation
