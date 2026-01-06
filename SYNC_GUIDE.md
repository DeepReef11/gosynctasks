# gosynctasks Sync Guide

Complete guide to using the offline synchronization system in gosynctasks.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Getting Started](#getting-started)
- [Configuration](#configuration)
- [Usage](#usage)
- [Offline Mode](#offline-mode)
- [Conflict Resolution](#conflict-resolution)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)
- [Advanced Topics](#advanced-topics)

## Overview

gosynctasks includes a robust offline synchronization system that allows you to:

- **Work offline**: Create, update, and delete tasks without network connectivity
- **Automatic sync**: Synchronize changes with remote backend (Nextcloud) when online
- **Conflict resolution**: Handle conflicts when the same task is modified both locally and remotely
- **Performance**: Efficiently sync thousands of tasks
- **Reliability**: Retry failed operations with exponential backoff

### How It Works

gosynctasks uses a **bidirectional sync architecture**:

1. **Local SQLite cache**: Tasks are stored locally in a SQLite database
2. **Remote backend**: Tasks are stored on a remote service (e.g., Nextcloud CalDAV)
3. **Sync manager**: Coordinates synchronization between local and remote
4. **Operation queue**: Tracks pending changes for upload

```
┌─────────────────┐                    ┌─────────────────┐
│  Local SQLite   │ ◄─── Sync ───────► │  Nextcloud      │
│  Database       │                    │  CalDAV         │
└─────────────────┘                    └─────────────────┘
        │                                       │
        │                                       │
   ┌────▼────┐                            ┌────▼────┐
   │ Offline │                            │  Online │
   │  Queue  │                            │  Tasks  │
   └─────────┘                            └─────────┘
```

## Architecture

### Components

#### 1. SQLite Backend (`backend/sqliteBackend.go`)

The local storage layer that implements the `TaskManager` interface:

- **CRUD operations**: Create, Read, Update, Delete tasks
- **Sync tracking**: Tracks which tasks have been modified locally
- **Operation queue**: Queues changes for sync
- **Metadata**: Stores sync state (ETags, timestamps, flags)

**Key Tables**:
- `tasks`: Main task data (follows iCalendar VTODO format)
- `sync_metadata`: Sync state per task (modified flags, ETags, timestamps)
- `list_sync_metadata`: Sync state per list (CTags, last sync time)
- `sync_queue`: Pending operations to push to remote

#### 2. Sync Manager (`backend/syncManager.go`)

Coordinates bidirectional synchronization:

- **Pull phase**: Downloads changes from remote → local
- **Push phase**: Uploads local changes → remote
- **Conflict detection**: Identifies when both sides have changes
- **Conflict resolution**: Resolves conflicts using configured strategy
- **Error handling**: Retries failed operations with backoff

**Sync Algorithm**:

```
1. Pull Phase:
   - Get remote task lists
   - For each list:
     - Check CTag (has list changed?)
     - If changed:
       - Fetch all tasks from remote
       - Sort by hierarchy (parents first)
       - For each remote task:
         - If doesn't exist locally → insert
         - If exists but not locally modified → update
         - If exists and locally modified → CONFLICT
       - Delete tasks missing from remote

2. Push Phase:
   - Get pending sync operations from queue
   - For each operation (create/update/delete):
     - Try to push to remote
     - On success: remove from queue, clear sync flags
     - On failure: increment retry count, log error
     - Apply exponential backoff for retries
```

#### 3. CLI Integration (`cmd/gosynctasks/sync.go`)

Provides user-facing sync commands:

- `gosynctasks sync`: Perform synchronization
- `gosynctasks sync status`: Show sync status
- `gosynctasks sync queue`: View pending operations
- `gosynctasks sync queue clear`: Clear failed operations

### Database Schema

**tasks** table:
```sql
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,           -- UID (unique identifier)
    list_id TEXT NOT NULL,         -- Parent list
    summary TEXT NOT NULL,         -- Task title
    description TEXT,              -- Task description
    status TEXT,                   -- NEEDS-ACTION, COMPLETED, IN-PROCESS, CANCELLED
    priority INTEGER DEFAULT 0,    -- 0-9 (0=undefined, 1=highest)
    created_at INTEGER,            -- Unix timestamp
    modified_at INTEGER,           -- Unix timestamp
    due_date INTEGER,              -- Unix timestamp
    start_date INTEGER,            -- Unix timestamp
    completed_at INTEGER,          -- Unix timestamp
    parent_uid TEXT,               -- Parent task UID (for subtasks)
    categories TEXT,               -- Comma-separated tags

    FOREIGN KEY(parent_uid) REFERENCES tasks(id) ON DELETE SET NULL
);
```

**sync_metadata** table:
```sql
CREATE TABLE sync_metadata (
    task_uid TEXT PRIMARY KEY,
    list_id TEXT NOT NULL,

    -- Remote state
    remote_etag TEXT,              -- Server's ETag
    last_synced_at INTEGER,        -- When last synced
    remote_modified_at INTEGER,    -- Server's last-modified timestamp

    -- Local state
    locally_modified INTEGER DEFAULT 0,    -- 1 if modified locally
    locally_deleted INTEGER DEFAULT 0,     -- 1 if deleted locally
    local_modified_at INTEGER,             -- Local modification timestamp

    FOREIGN KEY(task_uid) REFERENCES tasks(id) ON DELETE CASCADE
);
```

**sync_queue** table:
```sql
CREATE TABLE sync_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_uid TEXT NOT NULL,
    list_id TEXT NOT NULL,
    operation TEXT NOT NULL CHECK(operation IN ('create', 'update', 'delete')),
    created_at INTEGER NOT NULL,
    retry_count INTEGER DEFAULT 0,
    last_error TEXT,

    UNIQUE(task_uid, operation)
);
```

## Getting Started

### Prerequisites

1. **Remote backend** (e.g., Nextcloud CalDAV server)
2. **Network connectivity** (for initial setup)
3. **Valid credentials** for remote backend

### Initial Setup

1. **Configure your remote backend** in `config.yaml`:

**Global Sync Configuration** (Automatic Caching for All Remote Backends)

```yaml
sync:
  enabled: true
  local_backend: sqlite       # Cache type (currently only sqlite supported)
  conflict_resolution: server_wins
  sync_interval: 5
  offline_mode: auto

backends:
  nextcloud:
    type: nextcloud
    enabled: true
    url: nextcloud://username:password@nextcloud.example.com
    insecure_skip_verify: false
    # Automatically cached at: ~/.local/share/gosynctasks/caches/nextcloud.db

default_backend: nextcloud
```

**How It Works:**
- When `sync.enabled = true`, each remote backend gets its own automatic cache database
- Cache databases are stored at: `~/.local/share/gosynctasks/caches/{backend-name}.db`
- Each remote backend has complete isolation - tasks never mix between backends
- Use `gosynctasks sync` to manually sync cache databases with their remote backends

**Opt-Out:** Remote backends can opt-out of automatic caching:
```yaml
backends:
  nextcloud:
    type: nextcloud
    enabled: true
    sync: {enabled: false}  # Don't cache this backend
```

2. **Perform initial sync** to download existing tasks:

```bash
gosynctasks sync
```

This will:
- Connect to your Nextcloud server
- Download all task lists and tasks
- Store them in local SQLite database
- Set up sync metadata for future syncs

3. **Verify sync worked**:

```bash
gosynctasks sync status
```

You should see:
```
=== Sync Status ===
Connection: Online
Local tasks: 42
Local lists: 3
Pending operations: 0
Locally modified: 0
Strategy: server_wins
Last sync: 1 minute ago
```

## Configuration

### Sync Settings

**NEW: Per-Backend Sync Configuration (Recommended)**

Configure sync behavior directly on each backend in `config.yaml`:

**Available Options:**

- `enabled` (boolean): Enable sync for this backend
- `remote_backend` (string): Remote backend name to sync with (e.g., Nextcloud)
- `conflict_resolution` (string): server_wins (default), local_wins, merge, or keep_both
- `auto_sync` (boolean): Enable background daemon sync for instant operations
- `sync_interval` (integer): Minutes between auto-syncs (0 = manual only)
- `offline_mode` (string): auto (default), online, or offline

**Example Configuration:**

```yaml
backends:
  local:
    type: sqlite
    enabled: true
    sync:
      enabled: true
      remote_backend: nextcloud
      conflict_resolution: server_wins
      auto_sync: true
      sync_interval: 5
      offline_mode: auto
```

**Auto-Sync Behavior:**

When `auto_sync: true`:
1. CLI operations return instantly (< 100ms)
2. Changes queued in `sync_queue` table
3. Background daemon spawned automatically
4. Queue synced to remote independently
5. Works offline - queue persists

### Backend Settings

**SQLite Backend:**
```yaml
backends:
  local:
    type: sqlite
    enabled: true
    db_path: ""  # Empty = XDG default (~/.local/share/gosynctasks/tasks.db)
```

**Nextcloud Backend:**
```yaml
backends:
  nextcloud:
    type: nextcloud
    enabled: true
    url: nextcloud://username:password@server.com
    insecure_skip_verify: false
```

URL format: `nextcloud://[user]:[pass]@[host][:port][/path]`

## Usage

### Basic Sync

Perform a synchronization:

```bash
gosynctasks sync
```

Output:
```
Syncing...

=== Sync Complete ===
Pulled tasks: 5
Pushed tasks: 2
Duration: 1.2s
```

### Auto-Sync (Recommended)

Enable automatic background synchronization for instant operations:

**Setup (Per-Backend Sync):**
```yaml
backends:
  local:
    type: sqlite
    enabled: true
    sync:
      enabled: true
      auto_sync: true
      remote_backend: nextcloud
```

**Setup (Legacy Global Sync):**
```yaml
sync:
  enabled: true
  auto_sync: true
  local_backend: local
  remote_backend: nextcloud
```

**Usage - Just work normally!**
```bash
# Operations return instantly - sync happens in background
$ gosynctasks MyList add "Buy milk"
Task 'Buy milk' added successfully to list 'MyList'
# ← Returns in < 100ms, sync happens after in background daemon

$ gosynctasks MyList complete "Buy milk"
Task 'Buy milk' marked as DONE in list 'MyList'
# ← Instant return, background sync

$ gosynctasks MyList update "Task" -s DONE
Task 'Task' updated successfully in list 'MyList'
# ← No waiting!
```

**Verify sync is working:**
```bash
# Check the queue (should be empty after a few seconds)
$ gosynctasks sync queue
No pending operations

# Check sync status
$ gosynctasks sync status
Last sync: 10 seconds ago
```

**Monitor background sync:**
```bash
# Check for background sync processes
$ ps aux | grep "gosynctasks sync"
# You might see: gosynctasks sync --quiet (daemon process)

# View queue if operations are pending
$ gosynctasks sync queue
Pending Operations (2):
  create: task-123 (list: MyList)
    Created: 2025-01-15 10:30:00
```

**Offline mode:**
When offline, operations queue up automatically:
```bash
# Disconnect from network
$ gosynctasks MyList add "Offline task 1"
Task 'Offline task 1' added successfully
# ← Still instant! Queued for later sync

$ gosynctasks MyList add "Offline task 2"
Task 'Offline task 2' added successfully
# ← Queue building up

# Check queue
$ gosynctasks sync queue
Pending Operations (2):
  create: task-xxx (list: MyList)
  create: task-yyy (list: MyList)

# Reconnect to network - next operation triggers sync
$ gosynctasks MyList get
# Background daemon syncs the queue automatically
```

### Manual Sync

Manually trigger synchronization:

```bash
gosynctasks sync
```

Output:
```
Syncing...

=== Sync Complete ===
Pulled tasks: 5
Pushed tasks: 2
Duration: 1.2s
```

**Use manual sync for:**
- Initial setup
- Forcing sync without adding tasks
- Troubleshooting
- When auto-sync is disabled

### Full Sync

Force a complete re-sync (ignores CTags):

```bash
gosynctasks sync --full
```

Use cases:
- After significant remote changes
- To resolve sync inconsistencies
- For troubleshooting

### Dry Run

Preview changes without applying them (not yet implemented):

```bash
gosynctasks sync --dry-run
```

### Sync Status

Check current sync state:

```bash
gosynctasks sync status
```

Output:
```
=== Sync Status ===
Connection: Online
Local tasks: 42
Local lists: 3
Pending operations: 2
Locally modified: 1
Strategy: server_wins
Last sync: 5 minutes ago
```

**Offline status example**:
```
=== Sync Status ===
Connection: Offline (Network unreachable)
Local tasks: 42
Local lists: 3
Pending operations: 5
Locally modified: 3
Strategy: server_wins
Last sync: 2 hours ago
```

### Sync Queue

View pending operations waiting to be pushed:

```bash
gosynctasks sync queue
```

Output:
```
Pending Operations (3):

  create: task-123 (list: work-tasks)
    Created: 2025-11-15 10:30:00

  update: task-456 (list: personal)
    Created: 2025-11-15 10:45:00
    Retries: 2
    Error: connection timeout

  delete: task-789 (list: work-tasks)
    Created: 2025-11-15 11:00:00
```

### Clear Failed Operations

Remove operations that have failed multiple times:

```bash
gosynctasks sync queue clear --failed
```

This is useful when:
- A task was deleted remotely but delete operation keeps failing
- Network errors persisted and operations are stuck
- You want to start fresh

### Retry Failed Operations

Reset retry count for failed operations:

```bash
gosynctasks sync queue retry
```

## Offline Mode

gosynctasks automatically detects when you're offline and operates seamlessly.

### How It Works

1. **Detection**: When you run a command, gosynctasks tries to reach the remote backend
2. **Offline operation**: If unreachable, operations proceed against local SQLite database
3. **Queue building**: Changes are automatically queued for later sync
4. **Sync on reconnect**: When back online, run `gosynctasks sync` to push changes

### Offline Workflow

```bash
# Disconnect from network (simulated)

# Add tasks offline
gosynctasks Work add "Review PR #123"
gosynctasks Work add "Update documentation"
gosynctasks Personal add "Buy groceries"

# Update tasks offline
gosynctasks Work update "Review PR" -s DONE

# Check status
gosynctasks sync status
# Output: Connection: Offline
#         Pending operations: 4

# Reconnect to network

# Sync changes
gosynctasks sync
# Output: Pushed tasks: 4
```

### Offline Indicators

When offline, you'll see warnings:

```
⚠ Offline mode: Network unreachable
Working with local cache. Changes will be synced when online.
```

### Limitations in Offline Mode

- **No remote list creation**: Can't create lists on remote while offline
- **No remote deletions**: Deleting lists/tasks queues deletion for later
- **No conflict detection**: Conflicts only detected during sync

## Conflict Resolution

Conflicts occur when:
- Same task modified both locally and remotely since last sync
- Task deleted remotely but modified locally (or vice versa)

### Strategies

#### Server Wins (Default - Safest)

Remote changes always override local changes.

**When to use**:
- Default strategy (prevents data loss on server)
- Multiple users editing same tasks
- Server is the source of truth

**Example**:
```
Local:  Task summary = "Finish report by Friday"
Remote: Task summary = "Finish report by Thursday"
Result: "Finish report by Thursday" (remote wins)
```

Configuration:
```yaml
sync:
  conflict_resolution: server_wins
```

#### Local Wins (Use with Caution)

Local changes override remote. Can overwrite other users' changes on shared lists.

Configuration:
```yaml
sync:
  conflict_resolution: local_wins
```

#### Merge (Experimental)

Intelligent merge of non-conflicting fields (summary: remote, priority: higher, categories: union).

Configuration:
```yaml
sync:
  conflict_resolution: merge
```

#### Keep Both

Keeps both versions - remote replaces local, local becomes copy with suffix.

Configuration:
```yaml
sync:
  conflict_resolution: keep_both
```

### Viewing Conflicts

During sync, conflicts are reported:

```
=== Sync Complete ===
Pulled tasks: 10
Pushed tasks: 5
Conflicts found: 2
Conflicts resolved: 2
Duration: 2.1s
```

## Troubleshooting

### Common Issues

#### 1. Sync Fails with Authentication Error

**Symptoms**:
```
Error: sync failed: failed to get remote lists: Authentication failed
```

**Solutions**:
- Verify credentials in config.json
- Check if password has special characters (needs URL encoding)
- Try app password instead of main password (Nextcloud)
- Verify Nextcloud CalDAV is enabled

#### 2. Tasks Not Syncing

**Symptoms**:
- `gosynctasks sync` reports 0 pulled/pushed tasks
- Changes not appearing on remote

**Solutions**:
```bash
# Check sync status
gosynctasks sync status

# Check pending operations
gosynctasks sync queue

# Try full sync
gosynctasks sync --full

# Clear and re-sync if needed
rm ~/.local/share/gosynctasks/tasks.db
gosynctasks sync

# If still not syncing, try launching sync directly with --backend argument, for instance:
gosynctasks --backend nextcloud-test sync

# If still not working, make sure sync is enabled in config and that the used backend is also enabled
```

#### 3. Duplicate Tasks

**Symptoms**: Same task appears multiple times or with " (local copy)" suffix

**Solution**:
```bash
# Delete duplicates
gosynctasks List delete "Task (local copy)"

# Switch conflict strategy in config.yaml:
# sync:
#   conflict_resolution: server_wins
```

#### 4. Sync Too Slow

**Symptoms**:
- Sync takes >30 seconds
- Large number of tasks (>1000)

**Solutions**:
```bash
# Check database size
du -h ~/.local/share/gosynctasks/tasks.db

# Vacuum database (compact)
sqlite3 ~/.local/share/gosynctasks/tasks.db "VACUUM;"

# Clear old completed tasks
gosynctasks List delete $(gosynctasks List get -s COMPLETED -v basic | grep UID)
```

#### 5. Foreign Key Constraint Errors

**Symptoms**:
```
Error: FOREIGN KEY constraint failed
```

**Causes**:
- Trying to create child task before parent
- Parent task missing

**Solutions**:
Sync uses `sortTasksByHierarchy()` to ensure parents are created first. If you see this error, it's a bug - please report it.

#### 6. Queue Growing Without Sync

**Symptoms**:
```
Pending operations: 100+
```

**Causes**:
- Network issues preventing push
- Authentication failures
- Remote backend down

**Solutions**:
```bash
# Check for errors
gosynctasks sync queue

# Retry failed operations
gosynctasks sync queue retry

# If stuck, clear queue
gosynctasks sync queue clear --failed
```

### Debug Mode

For detailed logging, set environment variable:

```bash
GOSYNCTASKS_DEBUG=1 gosynctasks sync
```

(Note: Debug mode not yet implemented)

### Database Inspection

Manually inspect the SQLite database:

```bash
# Open database
sqlite3 ~/.local/share/gosynctasks/tasks.db

# List tables
.tables

# Check sync queue
SELECT * FROM sync_queue;

# Check locally modified tasks
SELECT t.summary, sm.locally_modified, sm.local_modified_at
FROM tasks t
JOIN sync_metadata sm ON t.id = sm.task_uid
WHERE sm.locally_modified = 1;

# Check sync metadata
SELECT * FROM list_sync_metadata;
```

## Best Practices

### 1. Sync Regularly

```bash
# Set up a cron job to sync every 5 minutes
*/5 * * * * gosynctasks sync
```

### 2. Use Server Wins Strategy

Unless you have a specific reason, stick with `server_wins`:

```yaml
sync:
  conflict_resolution: server_wins
```

### 3. Monitor Sync Status

Regularly check sync health:

```bash
gosynctasks sync status
```

### 4. Handle Offline Gracefully

Before going offline for extended periods:

```bash
# Sync everything first
gosynctasks sync

# Note pending changes
gosynctasks sync queue
```

### 5. Backup Your Database

```bash
# Backup before major operations
cp ~/.local/share/gosynctasks/tasks.db ~/.local/share/gosynctasks/tasks.db.backup

# Restore if needed
cp ~/.local/share/gosynctasks/tasks.db.backup ~/.local/share/gosynctasks/tasks.db
```

### 6. Clean Up Completed Tasks

Periodically clean old completed tasks to keep sync fast:

```bash
# Delete completed tasks older than 30 days
# (Manual implementation needed)
```

### 7. Test Conflict Resolution

Before using a new conflict resolution strategy, test it:

```bash
# Create test list
gosynctasks list create "Test Conflicts"

# Add task
gosynctasks "Test Conflicts" add "Test task"

# Modify locally
gosynctasks "Test Conflicts" update "Test task" -s PROCESSING

# Modify remotely (via Nextcloud UI)

# Sync and observe
gosynctasks sync
```

## Advanced Topics

### Custom Database Location

Override default database path:

```yaml
backends:
  local:
    type: sqlite
    enabled: true
    db_path: /custom/path/to/tasks.db
```

### Performance Tuning

For very large datasets (>10,000 tasks):

1. **Database tuning**:
```sql
-- In SQLite
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -64000;  -- 64MB cache
PRAGMA temp_store = MEMORY;
```

2. **Batch operations**: Sync in smaller batches
3. **Selective sync**: Sync only specific lists (future feature)

### Data Migration

Moving to a new machine:

```bash
# On old machine
tar czf gosynctasks-backup.tar.gz \
  ~/.config/gosynctasks \
  ~/.local/share/gosynctasks

# On new machine
tar xzf gosynctasks-backup.tar.gz -C ~/

# Verify
gosynctasks sync status
```

## FAQ

**Q: Can I use multiple devices?**
A: Yes! Each device has its own local SQLite database that syncs with the same remote backend.

**Q: What happens if two devices sync at the same time?**
A: The sync manager handles this through CTags (change tags). The second sync will detect changes and resolve conflicts.

**Q: Can I disable sync temporarily?**
A: Yes, set `"enabled": false` in the sync configuration, or just don't run `gosynctasks sync`.

**Q: Does sync work with Git backend?**
A: Not yet. Sync currently only works with Nextcloud CalDAV backend.

**Q: How can I see sync errors?**
A: Check `gosynctasks sync queue` - errors are shown for failed operations.

**Q: Can I sync only specific lists?**
A: Not yet, but this is a planned feature.

**Q: Is my data encrypted?**
A: Communication with Nextcloud uses HTTPS (if configured). Local SQLite database is not encrypted.

## Support

- **Documentation**: https://docs.gosynctasks.dev
- **Issues**: https://github.com/DeepReef11/gosynctasks/issues
- **Discussions**: https://github.com/DeepReef11/gosynctasks/discussions

## See Also

- [README.md](README.md) - General usage guide
- [CLAUDE.md](CLAUDE.md) - Development guide
- [Configuration Guide](docs/configuration.md)
