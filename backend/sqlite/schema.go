package sqlite

// Schema version for migration management
const SchemaVersion = 1

// SQL statements for database schema creation

// TasksTableSQL creates the main tasks table following VTODO iCalendar format
const TasksTableSQL = `
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
    parent_uid TEXT,
    categories TEXT,

    FOREIGN KEY(parent_uid) REFERENCES tasks(id) ON DELETE SET NULL
);
`

// SyncMetadataTableSQL creates the sync metadata table for tracking sync state per task
const SyncMetadataTableSQL = `
CREATE TABLE IF NOT EXISTS sync_metadata (
    task_uid TEXT PRIMARY KEY,
    list_id TEXT NOT NULL,

    -- Server state tracking
    remote_etag TEXT,
    last_synced_at INTEGER,

    -- Local state flags
    locally_modified INTEGER DEFAULT 0,
    locally_deleted INTEGER DEFAULT 0,

    -- Conflict detection
    remote_modified_at INTEGER,
    local_modified_at INTEGER,

    FOREIGN KEY(task_uid) REFERENCES tasks(id) ON DELETE CASCADE
);
`

// ListSyncMetadataTableSQL creates the list sync metadata table for tracking sync state per list
const ListSyncMetadataTableSQL = `
CREATE TABLE IF NOT EXISTS list_sync_metadata (
    list_id TEXT PRIMARY KEY,
    list_name TEXT NOT NULL,
    list_color TEXT,

    -- Sync state tracking
    last_ctag TEXT,
    last_full_sync INTEGER,
    sync_token TEXT,

    -- List metadata
    created_at INTEGER,
    modified_at INTEGER
);
`

// SyncQueueTableSQL creates the sync queue table for operations to perform on next sync
const SyncQueueTableSQL = `
CREATE TABLE IF NOT EXISTS sync_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_uid TEXT NOT NULL,
    list_id TEXT NOT NULL,
    operation TEXT NOT NULL CHECK(operation IN ('create', 'update', 'delete')),
    created_at INTEGER NOT NULL,
    retry_count INTEGER DEFAULT 0,
    last_error TEXT,

    -- Ensure we don't queue duplicate operations for the same task
    UNIQUE(task_uid, operation)
);
`

// SchemaVersionTableSQL creates the schema version table for migration tracking
const SchemaVersionTableSQL = `
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL
);
`

// Index creation statements for performance optimization

// TasksIndexesSQL creates indexes on tasks table for common queries
const TasksIndexesSQL = `
CREATE INDEX IF NOT EXISTS idx_tasks_list_id ON tasks(list_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON tasks(due_date);
CREATE INDEX IF NOT EXISTS idx_tasks_parent_uid ON tasks(parent_uid);
CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);
`

// SyncMetadataIndexesSQL creates indexes on sync_metadata table
const SyncMetadataIndexesSQL = `
CREATE INDEX IF NOT EXISTS idx_sync_metadata_locally_modified ON sync_metadata(locally_modified);
CREATE INDEX IF NOT EXISTS idx_sync_metadata_locally_deleted ON sync_metadata(locally_deleted);
CREATE INDEX IF NOT EXISTS idx_sync_metadata_list_id ON sync_metadata(list_id);
`

// SyncQueueIndexesSQL creates indexes on sync_queue table
const SyncQueueIndexesSQL = `
CREATE INDEX IF NOT EXISTS idx_sync_queue_operation ON sync_queue(operation);
CREATE INDEX IF NOT EXISTS idx_sync_queue_created_at ON sync_queue(created_at);
CREATE INDEX IF NOT EXISTS idx_sync_queue_retry_count ON sync_queue(retry_count);
`

// AllTableSchemas returns all table creation statements in order
func AllTableSchemas() []string {
	return []string{
		SchemaVersionTableSQL,
		TasksTableSQL,
		SyncMetadataTableSQL,
		ListSyncMetadataTableSQL,
		SyncQueueTableSQL,
	}
}

// AllIndexes returns all index creation statements
func AllIndexes() []string {
	return []string{
		TasksIndexesSQL,
		SyncMetadataIndexesSQL,
		SyncQueueIndexesSQL,
	}
}

// PragmaStatements returns pragma statements to execute on database connection
func PragmaStatements() []string {
	return []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",   // Write-Ahead Logging for better concurrency
		"PRAGMA synchronous = NORMAL", // Balance between safety and performance
	}
}
