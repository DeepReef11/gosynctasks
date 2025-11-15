package backend

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestInitDatabase tests database initialization
func TestInitDatabase(t *testing.T) {
	// Create temporary directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Initialize database
	db, err := InitDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Verify database was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("Database file was not created at %s", dbPath)
	}

	// Verify database path is correct
	if db.Path() != dbPath {
		t.Errorf("Expected path %s, got %s", dbPath, db.Path())
	}
}

// TestAllTablesCreated tests that all required tables are created
func TestAllTablesCreated(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	expectedTables := []string{
		"tasks",
		"sync_metadata",
		"list_sync_metadata",
		"sync_queue",
		"schema_version",
	}

	for _, table := range expectedTables {
		var count int
		query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"
		err := db.QueryRow(query, table).Scan(&count)
		if err != nil {
			t.Errorf("Failed to query for table %s: %v", table, err)
			continue
		}
		if count != 1 {
			t.Errorf("Table %s was not created (count: %d)", table, count)
		}
	}
}

// TestTasksTableSchema tests the tasks table schema
func TestTasksTableSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Test inserting a task with all fields
	now := time.Now().Unix()
	_, err = db.Exec(`
		INSERT INTO tasks (
			id, list_id, summary, description, status, priority,
			created_at, modified_at, due_date, start_date, completed_at,
			parent_uid, categories
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		"task-1", "list-1", "Test Task", "Description", "NEEDS-ACTION", 5,
		now, now, now+86400, now, nil,
		nil, "work,urgent",
	)
	if err != nil {
		t.Errorf("Failed to insert task: %v", err)
	}

	// Verify task was inserted
	var summary string
	err = db.QueryRow("SELECT summary FROM tasks WHERE id = ?", "task-1").Scan(&summary)
	if err != nil {
		t.Errorf("Failed to query task: %v", err)
	}
	if summary != "Test Task" {
		t.Errorf("Expected summary 'Test Task', got '%s'", summary)
	}
}

// TestSyncMetadataTableSchema tests the sync_metadata table schema
func TestSyncMetadataTableSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// First create a task (foreign key dependency)
	now := time.Now().Unix()
	_, err = db.Exec(`
		INSERT INTO tasks (id, list_id, summary, status, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, "task-1", "list-1", "Test Task", "NEEDS-ACTION", now)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Insert sync metadata
	_, err = db.Exec(`
		INSERT INTO sync_metadata (
			task_uid, list_id, remote_etag, last_synced_at,
			locally_modified, locally_deleted,
			remote_modified_at, local_modified_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "task-1", "list-1", "etag-123", now, 0, 0, now, now)
	if err != nil {
		t.Errorf("Failed to insert sync metadata: %v", err)
	}

	// Verify metadata was inserted
	var etag string
	err = db.QueryRow("SELECT remote_etag FROM sync_metadata WHERE task_uid = ?", "task-1").Scan(&etag)
	if err != nil {
		t.Errorf("Failed to query sync metadata: %v", err)
	}
	if etag != "etag-123" {
		t.Errorf("Expected etag 'etag-123', got '%s'", etag)
	}
}

// TestListSyncMetadataTableSchema tests the list_sync_metadata table schema
func TestListSyncMetadataTableSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()
	_, err = db.Exec(`
		INSERT INTO list_sync_metadata (
			list_id, list_name, list_color, last_ctag,
			last_full_sync, sync_token, created_at, modified_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "list-1", "Work Tasks", "#ff0000", "ctag-456", now, "sync-token-123", now, now)
	if err != nil {
		t.Errorf("Failed to insert list sync metadata: %v", err)
	}

	// Verify metadata was inserted
	var listName string
	err = db.QueryRow("SELECT list_name FROM list_sync_metadata WHERE list_id = ?", "list-1").Scan(&listName)
	if err != nil {
		t.Errorf("Failed to query list sync metadata: %v", err)
	}
	if listName != "Work Tasks" {
		t.Errorf("Expected list name 'Work Tasks', got '%s'", listName)
	}
}

// TestSyncQueueTableSchema tests the sync_queue table schema
func TestSyncQueueTableSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()
	_, err = db.Exec(`
		INSERT INTO sync_queue (
			task_uid, list_id, operation, created_at, retry_count, last_error
		) VALUES (?, ?, ?, ?, ?, ?)
	`, "task-1", "list-1", "create", now, 0, nil)
	if err != nil {
		t.Errorf("Failed to insert sync queue entry: %v", err)
	}

	// Verify queue entry was inserted
	var operation string
	err = db.QueryRow("SELECT operation FROM sync_queue WHERE task_uid = ?", "task-1").Scan(&operation)
	if err != nil {
		t.Errorf("Failed to query sync queue: %v", err)
	}
	if operation != "create" {
		t.Errorf("Expected operation 'create', got '%s'", operation)
	}
}

// TestForeignKeyConstraints tests that foreign key constraints work
func TestForeignKeyConstraints(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Verify foreign keys are enabled
	var fkEnabled int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("Failed to check foreign keys pragma: %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("Foreign keys are not enabled (got %d)", fkEnabled)
	}

	// Test foreign key constraint: sync_metadata -> tasks
	now := time.Now().Unix()
	_, err = db.Exec(`
		INSERT INTO sync_metadata (task_uid, list_id, last_synced_at)
		VALUES (?, ?, ?)
	`, "nonexistent-task", "list-1", now)
	if err == nil {
		t.Error("Expected foreign key constraint error, but insert succeeded")
	}

	// Create parent task
	_, err = db.Exec(`
		INSERT INTO tasks (id, list_id, summary, status, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, "task-1", "list-1", "Test Task", "NEEDS-ACTION", now)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Now sync_metadata insert should work
	_, err = db.Exec(`
		INSERT INTO sync_metadata (task_uid, list_id, last_synced_at)
		VALUES (?, ?, ?)
	`, "task-1", "list-1", now)
	if err != nil {
		t.Errorf("Failed to insert sync metadata with valid foreign key: %v", err)
	}
}

// TestCascadeDelete tests that cascade delete works for sync_metadata
func TestCascadeDelete(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()

	// Create task and sync metadata
	_, err = db.Exec(`
		INSERT INTO tasks (id, list_id, summary, status, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, "task-1", "list-1", "Test Task", "NEEDS-ACTION", now)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO sync_metadata (task_uid, list_id, last_synced_at)
		VALUES (?, ?, ?)
	`, "task-1", "list-1", now)
	if err != nil {
		t.Fatalf("Failed to create sync metadata: %v", err)
	}

	// Delete task
	_, err = db.Exec("DELETE FROM tasks WHERE id = ?", "task-1")
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	// Verify sync_metadata was also deleted (cascade)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sync_metadata WHERE task_uid = ?", "task-1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query sync metadata: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected sync_metadata to be cascade deleted, but found %d rows", count)
	}
}

// TestIndexesCreated tests that all indexes are created
func TestIndexesCreated(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	expectedIndexes := []string{
		"idx_tasks_list_id",
		"idx_tasks_status",
		"idx_tasks_due_date",
		"idx_tasks_parent_uid",
		"idx_tasks_priority",
		"idx_sync_metadata_locally_modified",
		"idx_sync_metadata_locally_deleted",
		"idx_sync_metadata_list_id",
		"idx_sync_queue_operation",
		"idx_sync_queue_created_at",
		"idx_sync_queue_retry_count",
	}

	for _, index := range expectedIndexes {
		var count int
		query := "SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?"
		err := db.QueryRow(query, index).Scan(&count)
		if err != nil {
			t.Errorf("Failed to query for index %s: %v", index, err)
			continue
		}
		if count != 1 {
			t.Errorf("Index %s was not created (count: %d)", index, count)
		}
	}
}

// TestSchemaVersion tests schema version recording and retrieval
func TestSchemaVersion(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Get schema version
	version, err := db.GetSchemaVersion()
	if err != nil {
		t.Errorf("Failed to get schema version: %v", err)
	}

	if version != SchemaVersion {
		t.Errorf("Expected schema version %d, got %d", SchemaVersion, version)
	}

	// Verify version was recorded only once
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_version WHERE version = ?", SchemaVersion).Scan(&count)
	if err != nil {
		t.Errorf("Failed to count version records: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 version record, got %d", count)
	}
}

// TestDatabaseStats tests database statistics collection
func TestDatabaseStats(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()

	// Add test data
	_, err = db.Exec(`
		INSERT INTO tasks (id, list_id, summary, status, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, "task-1", "list-1", "Test Task", "NEEDS-ACTION", now)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO list_sync_metadata (list_id, list_name, created_at)
		VALUES (?, ?, ?)
	`, "list-1", "Work", now)
	if err != nil {
		t.Fatalf("Failed to create list metadata: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO sync_metadata (task_uid, list_id, locally_modified, last_synced_at)
		VALUES (?, ?, ?, ?)
	`, "task-1", "list-1", 1, now)
	if err != nil {
		t.Fatalf("Failed to create sync metadata: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO sync_queue (task_uid, list_id, operation, created_at)
		VALUES (?, ?, ?, ?)
	`, "task-1", "list-1", "update", now)
	if err != nil {
		t.Fatalf("Failed to create sync queue entry: %v", err)
	}

	// Get stats
	stats, err := db.GetStats()
	if err != nil {
		t.Errorf("Failed to get database stats: %v", err)
	}

	// Verify stats
	if stats.TaskCount != 1 {
		t.Errorf("Expected 1 task, got %d", stats.TaskCount)
	}
	if stats.ListCount != 1 {
		t.Errorf("Expected 1 list, got %d", stats.ListCount)
	}
	if stats.LocallyModified != 1 {
		t.Errorf("Expected 1 locally modified task, got %d", stats.LocallyModified)
	}
	if stats.PendingSyncOps != 1 {
		t.Errorf("Expected 1 pending sync operation, got %d", stats.PendingSyncOps)
	}
	if stats.DatabaseSize == 0 {
		t.Error("Expected non-zero database size")
	}

	// Test String() method
	statsStr := stats.String()
	if statsStr == "" {
		t.Error("Expected non-empty stats string")
	}
}

// TestDatabasePathResolution tests XDG path resolution
func TestDatabasePathResolution(t *testing.T) {
	tests := []struct {
		name        string
		customPath  string
		xdgDataHome string
		expectPath  func(string) string // Function to generate expected path
	}{
		{
			name:       "Custom path takes priority",
			customPath: "/tmp/custom.db",
			expectPath: func(home string) string { return "/tmp/custom.db" },
		},
		{
			name:        "XDG_DATA_HOME when set",
			customPath:  "",
			xdgDataHome: "/tmp/xdg",
			expectPath:  func(home string) string { return "/tmp/xdg/gosynctasks/tasks.db" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment
			if tt.xdgDataHome != "" {
				oldXDG := os.Getenv("XDG_DATA_HOME")
				os.Setenv("XDG_DATA_HOME", tt.xdgDataHome)
				defer os.Setenv("XDG_DATA_HOME", oldXDG)
			}

			homeDir, err := os.UserHomeDir()
			if err != nil {
				t.Fatalf("Failed to get home dir: %v", err)
			}

			path, err := getDatabasePath(tt.customPath)
			if err != nil {
				t.Errorf("Failed to get database path: %v", err)
			}

			expectedPath := tt.expectPath(homeDir)
			if path != expectedPath {
				t.Errorf("Expected path %s, got %s", expectedPath, path)
			}
		})
	}
}

// TestSyncQueueOperationConstraint tests the CHECK constraint on sync_queue.operation
func TestSyncQueueOperationConstraint(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()

	// Valid operations should succeed
	validOps := []string{"create", "update", "delete"}
	for _, op := range validOps {
		_, err = db.Exec(`
			INSERT INTO sync_queue (task_uid, list_id, operation, created_at)
			VALUES (?, ?, ?, ?)
		`, "task-"+op, "list-1", op, now)
		if err != nil {
			t.Errorf("Failed to insert valid operation %s: %v", op, err)
		}
	}

	// Invalid operation should fail
	_, err = db.Exec(`
		INSERT INTO sync_queue (task_uid, list_id, operation, created_at)
		VALUES (?, ?, ?, ?)
	`, "task-invalid", "list-1", "invalid_op", now)
	if err == nil {
		t.Error("Expected CHECK constraint error for invalid operation, but insert succeeded")
	}
}

// TestParentTaskReference tests parent-child task relationships
func TestParentTaskReference(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()

	// Create parent task
	_, err = db.Exec(`
		INSERT INTO tasks (id, list_id, summary, status, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, "parent-1", "list-1", "Parent Task", "NEEDS-ACTION", now)
	if err != nil {
		t.Fatalf("Failed to create parent task: %v", err)
	}

	// Create child task referencing parent
	_, err = db.Exec(`
		INSERT INTO tasks (id, list_id, summary, status, created_at, parent_uid)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "child-1", "list-1", "Child Task", "NEEDS-ACTION", now, "parent-1")
	if err != nil {
		t.Errorf("Failed to create child task: %v", err)
	}

	// Verify parent reference
	var parentUID sql.NullString
	err = db.QueryRow("SELECT parent_uid FROM tasks WHERE id = ?", "child-1").Scan(&parentUID)
	if err != nil {
		t.Errorf("Failed to query child task: %v", err)
	}
	if !parentUID.Valid || parentUID.String != "parent-1" {
		t.Errorf("Expected parent_uid 'parent-1', got %v", parentUID)
	}

	// Delete parent task - child's parent_uid should be set to NULL
	_, err = db.Exec("DELETE FROM tasks WHERE id = ?", "parent-1")
	if err != nil {
		t.Fatalf("Failed to delete parent task: %v", err)
	}

	// Verify child's parent_uid is NULL
	err = db.QueryRow("SELECT parent_uid FROM tasks WHERE id = ?", "child-1").Scan(&parentUID)
	if err != nil {
		t.Errorf("Failed to query child task after parent deletion: %v", err)
	}
	if parentUID.Valid {
		t.Errorf("Expected NULL parent_uid after parent deletion, got %v", parentUID.String)
	}
}

// TestVacuum tests database vacuum operation
func TestVacuum(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run vacuum
	err = db.Vacuum()
	if err != nil {
		t.Errorf("Failed to vacuum database: %v", err)
	}
}
