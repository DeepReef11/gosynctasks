# Background Sync Logger

## Overview

The background logger provides dedicated logging for the `_internal_background_sync` process. Each background sync spawns with its own PID-specific log file for easy debugging.

## Log File Location

When enabled, logs are written to:
```
/tmp/gosynctasks-_internal_background_sync-{PID}.log
```

Example:
```
/tmp/gosynctasks-_internal_background_sync-12345.log
```

## Enabling/Disabling Logging

To control background logging, edit `internal/utils/logger.go`:

```go
// Set to true to enable logging (default)
const ENABLE_BACKGROUND_LOGGING = true

// Set to false to disable logging completely
const ENABLE_BACKGROUND_LOGGING = false
```

When disabled:
- No log files are created
- All logging calls become no-ops (zero overhead)
- Background sync still runs normally, just without logging

## Log Content Example

```
[BackgroundSync] 2026/01/05 17:45:23 Started at 2026-01-05T17:45:23Z (PID: 12345)
[BackgroundSync] 2026/01/05 17:45:23 Log file: /tmp/gosynctasks-_internal_background_sync-12345.log
[BackgroundSync] 2026/01/05 17:45:23 Found 2 sync pairs
[BackgroundSync] 2026/01/05 17:45:23 Checking backend: nextcloud
[BackgroundSync] 2026/01/05 17:45:23 Backend nextcloud has 3 pending operations
[BackgroundSync] 2026/01/05 17:45:24 Successfully synced nextcloud: 3 tasks pushed
[BackgroundSync] 2026/01/05 17:45:24 Completed sync for nextcloud
[BackgroundSync] 2026/01/05 17:45:24 Checking backend: todoist
[BackgroundSync] 2026/01/05 17:45:24 Backend todoist has 0 pending operations
[BackgroundSync] 2026/01/05 17:45:24 Finished at 2026-01-05T17:45:24Z
```

## Viewing Logs

### Find current background sync processes
```bash
ps aux | grep _internal_background_sync
```

### View logs for a specific PID
```bash
tail -f /tmp/gosynctasks-_internal_background_sync-12345.log
```

### View all background sync logs
```bash
ls -lt /tmp/gosynctasks-_internal_background_sync-*.log
cat /tmp/gosynctasks-_internal_background_sync-*.log
```

### Clean up old log files
```bash
# Remove logs older than 7 days
find /tmp -name "gosynctasks-_internal_background_sync-*.log" -mtime +7 -delete

# Remove all background sync logs
rm /tmp/gosynctasks-_internal_background_sync-*.log
```

## Use Cases

### Debugging sync issues
1. Enable logging (if not already enabled)
2. Trigger a sync operation: `gosynctasks add "test task"`
3. Find the background sync PID from process list
4. View the log file: `tail -f /tmp/gosynctasks-_internal_background_sync-{PID}.log`

### Monitoring sync performance
The logs show:
- Which backends are being synced
- How many pending operations each backend has
- How many tasks were pushed
- Sync errors and timeouts
- Total sync duration

### Troubleshooting

**No log file created:**
- Check if `ENABLE_BACKGROUND_LOGGING = true` in `background_logger.go`
- Verify background sync is running: `ps aux | grep _internal_background_sync`
- Check sync is enabled in config: `sync.enabled = true` and `sync.auto_sync = true`

**Log file exists but empty:**
- Background sync may have completed quickly
- Check if there were any pending operations to sync

**Multiple log files accumulating:**
- Each background sync process creates a new log file
- Old logs are not automatically cleaned up
- Use the cleanup commands above to remove old logs

## Implementation Details

The background logger:
- Uses Go's standard `log` package
- Writes to a file handle opened in append mode
- Prefixes all messages with `[BackgroundSync]`
- Includes timestamps in format: `2006/01/02 15:04:05`
- Automatically flushes when the process exits
- Supports Printf, Print, and Println methods

## Performance Impact

When enabled:
- Minimal overhead (file I/O is buffered)
- Log files are small (typically <1KB per sync)

When disabled:
- Zero overhead (all calls are no-ops)
- No file I/O occurs
