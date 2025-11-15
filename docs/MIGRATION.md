# Multi-Backend Migration Guide

This guide explains how to migrate from the old single-backend configuration to the new multi-backend system. The migration process is **automatic**, but this guide provides details on what happens and how to customize your setup afterward.

## Table of Contents

- [Overview](#overview)
- [Automatic Migration](#automatic-migration)
- [Manual Migration](#manual-migration)
- [Post-Migration Setup](#post-migration-setup)
- [Common Migration Scenarios](#common-migration-scenarios)
- [Troubleshooting](#troubleshooting)
- [Rollback](#rollback)

## Overview

### What Changed?

**Old System (Single Backend):**
- One backend configured via `connector` field
- No backend selection
- No auto-detection

**New System (Multi-Backend):**
- Multiple backends in `backends` map
- Backend selection via CLI flags or auto-detection
- Support for Nextcloud, Git, and future backend types

### Why Migrate?

The new multi-backend system provides:
- ✅ **Flexibility**: Use multiple backends simultaneously
- ✅ **Git Integration**: Manage tasks in markdown files
- ✅ **Auto-Detection**: Automatic backend selection based on context
- ✅ **Future-Proof**: Easy to add new backend types
- ✅ **Backward Compatible**: Old configs still work

## Automatic Migration

### How It Works

When you run gosynctasks with an old config, it will:

1. **Detect old format** (presence of `connector` field)
2. **Create backup** at `config.json.backup`
3. **Convert to new format**
4. **Save new config**
5. **Continue normally**

### What Gets Migrated

**Before (`~/.config/gosynctasks/config.json`):**
```json
{
  "connector": {
    "url": "nextcloud://user:pass@server.com",
    "insecure_skip_verify": false
  },
  "ui": "cli"
}
```

**After (automatically):**
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
  "ui": "cli",
  "can_write_config": true
}
```

**Backup created at:** `~/.config/gosynctasks/config.json.backup`

### Migration Example

```bash
# Before migration - old format
$ cat ~/.config/gosynctasks/config.json
{
  "connector": {
    "url": "nextcloud://admin:pass@localhost:8080"
  },
  "ui": "cli"
}

# Run gosynctasks (triggers migration)
$ gosynctasks

# After migration - new format
$ cat ~/.config/gosynctasks/config.json
{
  "backends": {
    "nextcloud": {
      "type": "nextcloud",
      "enabled": true,
      "url": "nextcloud://admin:pass@localhost:8080",
      "insecure_skip_verify": false
    }
  },
  "default_backend": "nextcloud",
  ...
}

# Backup preserved
$ cat ~/.config/gosynctasks/config.json.backup
{
  "connector": {
    "url": "nextcloud://admin:pass@localhost:8080"
  },
  "ui": "cli"
}
```

### What's Preserved

✅ Nextcloud URL and credentials
✅ `insecure_skip_verify` setting
✅ UI preference
✅ All other settings

### What's New

🆕 `backends` map structure
🆕 `default_backend` field
🆕 `auto_detect_backend` field
🆕 `backend_priority` list
🆕 `can_write_config` field

## Manual Migration

If you prefer to migrate manually or need custom configuration:

### Step 1: Backup Current Config

```bash
cp ~/.config/gosynctasks/config.json ~/.config/gosynctasks/config.json.manual-backup
```

### Step 2: Choose Template

Pick a template from `docs/config-examples/`:

```bash
# List available templates
ls docs/config-examples/

# Copy desired template
cp docs/config-examples/01-nextcloud-only.json ~/.config/gosynctasks/config.json
```

### Step 3: Customize

Edit the config with your credentials and preferences:

```bash
# Edit with your preferred editor
nano ~/.config/gosynctasks/config.json
# or
vim ~/.config/gosynctasks/config.json
```

### Step 4: Verify

```bash
# Test the config
gosynctasks --list-backends

# Try listing tasks
gosynctasks
```

## Post-Migration Setup

After migration, you can enhance your configuration with new features.

### Adding Git Backend

1. **Create TODO.md in your project:**

```bash
cd /path/to/your/project

cat > TODO.md << 'EOF'
<!-- gosynctasks:enabled -->

## Tasks
- [ ] Your first task
EOF

git add TODO.md
git commit -m "Add TODO.md for gosynctasks"
```

2. **Add git backend to config:**

```json
{
  "backends": {
    "nextcloud": {
      "type": "nextcloud",
      "url": "nextcloud://user:pass@server.com"
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
  "backend_priority": ["git", "nextcloud"]
}
```

3. **Test auto-detection:**

```bash
cd /path/to/your/project
gosynctasks --detect-backend
# Should output: git

cd ~
gosynctasks --detect-backend
# Should output: nextcloud
```

### Enabling Auto-Detection

If you want gosynctasks to automatically choose backends:

```json
{
  "auto_detect_backend": true,
  "backend_priority": ["git", "nextcloud"]
}
```

**Behavior:**
- In git repos with TODO.md → Uses git backend
- Outside git repos → Uses default backend

### Adding Multiple Backends

```json
{
  "backends": {
    "nextcloud": {
      "type": "nextcloud",
      "url": "nextcloud://user:pass@server.com"
    },
    "work-git": {
      "type": "git",
      "file": "/home/user/work/TODO.md",
      "auto_detect": false
    },
    "personal-git": {
      "type": "git",
      "file": "/home/user/personal/TODO.md",
      "auto_detect": false
    }
  },
  "default_backend": "nextcloud"
}
```

**Usage:**
```bash
# Default backend (nextcloud)
gosynctasks MyList get

# Explicit backend selection
gosynctasks --backend work-git Tasks get
gosynctasks --backend personal-git Todo get
```

## Common Migration Scenarios

### Scenario 1: "I only use Nextcloud and want minimal changes"

**Action:** Nothing! The automatic migration creates a compatible config.

**Result:** Everything works exactly as before.

```json
{
  "backends": {
    "nextcloud": { "type": "nextcloud", "url": "..." }
  },
  "default_backend": "nextcloud",
  "auto_detect_backend": false
}
```

### Scenario 2: "I want to try Git backend alongside Nextcloud"

**Action:** Add git backend and enable auto-detection.

**Steps:**
1. Let automatic migration happen
2. Add git backend to config
3. Enable `auto_detect_backend: true`
4. Set priority: `["git", "nextcloud"]`

**Result:**
- Git repos → Git backend
- Elsewhere → Nextcloud backend

**Config:**
```json
{
  "backends": {
    "nextcloud": { "type": "nextcloud", "url": "..." },
    "git": { "type": "git", "file": "TODO.md", "auto_detect": true }
  },
  "auto_detect_backend": true,
  "backend_priority": ["git", "nextcloud"]
}
```

### Scenario 3: "I want separate tasks for work and personal"

**Action:** Set up multiple backends.

**Steps:**
1. Create git repos for work and personal projects
2. Add TODO.md to each
3. Configure multiple backends
4. Use `--backend` flag to switch

**Config:**
```json
{
  "backends": {
    "work": {
      "type": "git",
      "file": "/home/user/work/TODO.md"
    },
    "personal": {
      "type": "git",
      "file": "/home/user/personal/TODO.md"
    }
  },
  "default_backend": "work"
}
```

**Usage:**
```bash
gosynctasks --backend work Tasks get
gosynctasks --backend personal Hobbies add "Learn Go"
```

### Scenario 4: "I manage multiple client projects"

**Action:** One backend per client.

**Config:**
```json
{
  "backends": {
    "clientA": {
      "type": "nextcloud",
      "url": "nextcloud://user:pass@clientA.com"
    },
    "clientB": {
      "type": "git",
      "file": "/home/user/clients/clientB/TODO.md"
    },
    "internal": {
      "type": "git",
      "file": "/home/user/company/TODO.md"
    }
  },
  "default_backend": "internal"
}
```

**Usage:**
```bash
gosynctasks --backend clientA "Sprint 1" get
gosynctasks --backend clientB "Development" add "Feature X"
gosynctasks --backend internal "Admin" add "Update docs"
```

## Troubleshooting

### Migration Didn't Happen

**Problem:** Config still in old format.

**Causes:**
- App can't write to config file (permissions)
- `can_write_config: false` in config

**Solution:**
```bash
# Check permissions
ls -la ~/.config/gosynctasks/config.json

# Fix permissions
chmod 644 ~/.config/gosynctasks/config.json

# Ensure can_write_config is true or missing
```

### Backup Not Created

**Problem:** No `config.json.backup` file.

**Possible reasons:**
- Migration already happened
- File permissions
- Disk space

**Solution:**
```bash
# Manually create backup
cp ~/.config/gosynctasks/config.json ~/.config/gosynctasks/config.json.backup
```

### Can't Connect After Migration

**Problem:** Authentication or connection errors.

**Diagnosis:**
```bash
# Check config format
cat ~/.config/gosynctasks/config.json | jq .

# Test connection
gosynctasks --list-backends
gosynctasks --backend nextcloud
```

**Common issues:**
- URL format changed → Should be same as before
- Credentials lost → Check backup file
- Backend name wrong → Use exact backend name

**Solution:**
```bash
# Compare with backup
diff ~/.config/gosynctasks/config.json ~/.config/gosynctasks/config.json.backup

# Restore if needed (see Rollback section)
```

### "Backend not found" Error

**Problem:** `Error: backend 'xyz' not found`

**Cause:** Trying to use backend name that doesn't exist in config.

**Solution:**
```bash
# List available backends
gosynctasks --list-backends

# Use correct backend name
gosynctasks --backend nextcloud MyList get
```

### Multiple Backends Detected

**Problem:** Auto-detection selects wrong backend.

**Cause:** Priority order in `backend_priority`.

**Solution:** Adjust priority:
```json
{
  "backend_priority": ["preferred", "second-choice", "fallback"]
}
```

## Rollback

If you need to revert to the old config:

### Option 1: Use Backup File

```bash
# Restore from automatic backup
cp ~/.config/gosynctasks/config.json.backup ~/.config/gosynctasks/config.json

# Or from manual backup
cp ~/.config/gosynctasks/config.json.manual-backup ~/.config/gosynctasks/config.json
```

### Option 2: Manual Rollback

Create old-format config manually:

```json
{
  "connector": {
    "url": "nextcloud://user:pass@server.com",
    "insecure_skip_verify": false
  },
  "ui": "cli"
}
```

**Note:** Old format will be migrated again on next run. To prevent this, use a version that doesn't support multi-backend (not recommended).

### Option 3: Keep New Format, Remove Extra Backends

If you just want to simplify:

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
  "auto_detect_backend": false,
  "backend_priority": ["nextcloud"],
  "ui": "cli"
}
```

This is functionally equivalent to the old single-backend setup.

## Best Practices

### 1. Always Keep Backups

```bash
# Before editing config
cp ~/.config/gosynctasks/config.json ~/.config/gosynctasks/config.json.$(date +%Y%m%d)
```

### 2. Test After Changes

```bash
# Verify config is valid
gosynctasks --list-backends

# Test basic operations
gosynctasks
gosynctasks MyList get
```

### 3. Use Version Control

```bash
# Track config changes (remove sensitive data first!)
cd ~/.config/gosynctasks
git init
git add config.json.example  # Sanitized version
git commit -m "Add gosynctasks config template"
```

### 4. Document Your Setup

Add comments (JSON doesn't support comments, so use a README):

```bash
cat > ~/.config/gosynctasks/README.md << 'EOF'
# My gosynctasks Configuration

## Backends

- `nextcloud`: Company Nextcloud server
- `work-git`: Work projects in ~/work/TODO.md
- `personal-git`: Personal projects in ~/personal/TODO.md

## Usage

- Default: Work git backend
- For company tasks: `--backend nextcloud`
- For personal: `--backend personal-git`
EOF
```

### 5. Secure Credentials

```bash
# Restrict config file permissions
chmod 600 ~/.config/gosynctasks/config.json

# Don't commit real credentials
# Use environment variables or keyring (future feature)
```

## FAQ

### Q: Will my old config stop working?

**A:** No. Old configs are automatically migrated and continue to work.

### Q: Do I need to change anything?

**A:** No. The migration is automatic and transparent. You can continue using gosynctasks exactly as before.

### Q: Can I use the old format?

**A:** The old format will be automatically migrated on next run. It's recommended to embrace the new format for access to new features.

### Q: What if I don't want multiple backends?

**A:** You don't have to use them. After migration, you'll have a single backend configured, just like before.

### Q: How do I know if migration succeeded?

**A:** Check for `backends` field in your config:
```bash
cat ~/.config/gosynctasks/config.json | jq .backends
```

### Q: Can I migrate back?

**A:** Yes, restore from `config.json.backup`. But you'll lose access to multi-backend features.

### Q: What happens to my tasks?

**A:** Nothing! Tasks are stored on the backend (Nextcloud server, git files). The config only controls how to connect.

### Q: Will future versions support old config?

**A:** The automatic migration will remain for backward compatibility, but new features will require the new format.

## Getting Help

If you encounter issues:

1. **Check logs/errors:** Look for specific error messages
2. **Verify config:** Use `jq` or JSON validator
3. **Compare with examples:** See `docs/config-examples/`
4. **Restore backup:** If all else fails
5. **Report issue:** Open issue on GitHub with config (sanitize credentials!)

## See Also

- [README.md](../README.md) - Main documentation
- [Configuration Examples](config-examples/README.md) - Example configs
- [TESTING.md](../TESTING.md) - Testing guide
- [CLAUDE.md](../CLAUDE.md) - Development docs

---

**Happy migrating! 🚀**
