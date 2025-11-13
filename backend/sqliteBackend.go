package backend

import (
	"database/sql"
	// _ "modernc.org/sqlite" // Commented out due to network issues - TODO: uncomment when implementing
)

// SQLiteBackend implements TaskManager interface for local SQLite storage
// TODO: Future sync implementation - see CLAUDE.md for architecture plans
type SQLiteBackend struct {
	Connector ConnectorConfig
	db        *sql.DB
}

// initDB initializes SQLite database with basic schema
// TODO: Expand schema to include sync metadata (ETags, timestamps, conflict tracking)
// See CLAUDE.md "Future Work: SQLite Sync Implementation" for full schema design
func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "tasks.db")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS tasks (
            id TEXT PRIMARY KEY,
            content TEXT,
            status TEXT,
            created_at INTEGER,
            updated_at INTEGER
        )
    `)
	return db, err
}

// TODO: Implement TaskManager interface methods:
// - GetTaskLists() ([]TaskList, error)
// - GetTasks(listID string, filter *TaskFilter) ([]Task, error)
// - FindTasksBySummary(listID string, summary string) ([]Task, error)
// - AddTask(listID string, task Task) error
// - UpdateTask(listID string, task Task) error
// - SortTasks(tasks []Task)
// - GetPriorityColor(priority int) string

// TODO: Add sync-specific methods:
// - Sync() error
// - MarkLocallyModified(taskUID string) error
// - GetPendingSyncOperations() ([]SyncOperation, error)
// - ClearSyncFlag(taskUID string) error
