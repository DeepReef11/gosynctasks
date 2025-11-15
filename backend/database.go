package backend

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // SQLite driver
)

// Database wraps sql.DB with helper methods for schema management
type Database struct {
	*sql.DB
	path string
}

// InitDatabase initializes the SQLite database with proper schema
// It creates the database at the XDG-compliant location and sets up all tables
func InitDatabase(customPath string) (*Database, error) {
	dbPath, err := getDatabasePath(customPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get database path: %w", err)
	}

	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	database := &Database{
		DB:   db,
		path: dbPath,
	}

	// Initialize schema
	if err := database.initializeSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return database, nil
}

// getDatabasePath returns the path to the SQLite database file
// Priority: customPath > $XDG_DATA_HOME/gosynctasks/tasks.db > ~/.local/share/gosynctasks/tasks.db
func getDatabasePath(customPath string) (string, error) {
	if customPath != "" {
		return customPath, nil
	}

	// Try XDG_DATA_HOME
	if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		return filepath.Join(xdgDataHome, "gosynctasks", "tasks.db"), nil
	}

	// Fallback to ~/.local/share
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(homeDir, ".local", "share", "gosynctasks", "tasks.db"), nil
}

// initializeSchema creates all tables, indexes, and sets pragmas
func (db *Database) initializeSchema() error {
	// Set pragmas first
	for _, pragma := range PragmaStatements() {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("failed to execute pragma %q: %w", pragma, err)
		}
	}

	// Create all tables
	for _, schema := range AllTableSchemas() {
		if _, err := db.Exec(schema); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Create all indexes
	for _, index := range AllIndexes() {
		if _, err := db.Exec(index); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Record schema version
	if err := db.recordSchemaVersion(); err != nil {
		return fmt.Errorf("failed to record schema version: %w", err)
	}

	return nil
}

// recordSchemaVersion records the current schema version in the database
func (db *Database) recordSchemaVersion() error {
	// Check if version already recorded
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_version WHERE version = ?", SchemaVersion).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check schema version: %w", err)
	}

	if count > 0 {
		return nil // Version already recorded
	}

	// Insert new version record
	_, err = db.Exec(
		"INSERT INTO schema_version (version, applied_at) VALUES (?, ?)",
		SchemaVersion,
		time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert schema version: %w", err)
	}

	return nil
}

// GetSchemaVersion returns the current schema version from the database
func (db *Database) GetSchemaVersion() (int, error) {
	var version int
	err := db.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get schema version: %w", err)
	}
	return version, nil
}

// Path returns the filesystem path to the database file
func (db *Database) Path() string {
	return db.path
}

// Vacuum runs VACUUM to optimize the database
func (db *Database) Vacuum() error {
	_, err := db.Exec("VACUUM")
	return err
}

// GetStats returns basic database statistics
func (db *Database) GetStats() (DatabaseStats, error) {
	stats := DatabaseStats{}

	// Count tasks
	err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&stats.TaskCount)
	if err != nil {
		return stats, fmt.Errorf("failed to count tasks: %w", err)
	}

	// Count lists
	err = db.QueryRow("SELECT COUNT(*) FROM list_sync_metadata").Scan(&stats.ListCount)
	if err != nil {
		return stats, fmt.Errorf("failed to count lists: %w", err)
	}

	// Count pending sync operations
	err = db.QueryRow("SELECT COUNT(*) FROM sync_queue").Scan(&stats.PendingSyncOps)
	if err != nil {
		return stats, fmt.Errorf("failed to count pending sync operations: %w", err)
	}

	// Count locally modified tasks
	err = db.QueryRow("SELECT COUNT(*) FROM sync_metadata WHERE locally_modified = 1").Scan(&stats.LocallyModified)
	if err != nil {
		return stats, fmt.Errorf("failed to count locally modified tasks: %w", err)
	}

	// Get database file size
	fileInfo, err := os.Stat(db.path)
	if err != nil {
		return stats, fmt.Errorf("failed to stat database file: %w", err)
	}
	stats.DatabaseSize = fileInfo.Size()

	return stats, nil
}

// DatabaseStats holds statistics about the database
type DatabaseStats struct {
	TaskCount       int
	ListCount       int
	PendingSyncOps  int
	LocallyModified int
	DatabaseSize    int64 // in bytes
}

// String returns a human-readable representation of database statistics
func (s DatabaseStats) String() string {
	sizeMB := float64(s.DatabaseSize) / (1024 * 1024)
	return fmt.Sprintf(
		"Tasks: %d | Lists: %d | Pending sync: %d | Modified: %d | Size: %.2f MB",
		s.TaskCount, s.ListCount, s.PendingSyncOps, s.LocallyModified, sizeMB,
	)
}
