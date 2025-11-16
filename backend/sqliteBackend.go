package backend

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite" // SQLite driver
)

// SQLiteError represents errors specific to SQLite backend operations
type SQLiteError struct {
	Op      string // Operation that failed
	Err     error  // Underlying error
	ListID  string // Optional: list ID if relevant
	TaskUID string // Optional: task UID if relevant
}

func (e *SQLiteError) Error() string {
	if e.ListID != "" && e.TaskUID != "" {
		return fmt.Sprintf("sqlite %s failed for task %s in list %s: %v", e.Op, e.TaskUID, e.ListID, e.Err)
	} else if e.ListID != "" {
		return fmt.Sprintf("sqlite %s failed for list %s: %v", e.Op, e.ListID, e.Err)
	} else if e.TaskUID != "" {
		return fmt.Sprintf("sqlite %s failed for task %s: %v", e.Op, e.TaskUID, e.Err)
	}
	return fmt.Sprintf("sqlite %s failed: %v", e.Op, e.Err)
}

func (e *SQLiteError) Unwrap() error {
	return e.Err
}

// SQLiteBackend implements TaskManager interface for local SQLite storage
type SQLiteBackend struct {
	Config BackendConfig
	db     *Database
}

// NewSQLiteBackend creates a new SQLite backend instance
func NewSQLiteBackend(config BackendConfig) (*SQLiteBackend, error) {
	backend := &SQLiteBackend{
		Config: config,
	}

	// Initialize database immediately
	if err := backend.initDB(); err != nil {
		return nil, &SQLiteError{Op: "init", Err: err}
	}

	return backend, nil
}

// initDB initializes the database connection (lazy initialization)
func (sb *SQLiteBackend) initDB() error {
	if sb.db != nil {
		return nil // Already initialized
	}

	db, err := InitDatabase(sb.Config.DBPath)
	if err != nil {
		return err
	}

	sb.db = db
	return nil
}

// GetDB returns the database connection, initializing if necessary
func (sb *SQLiteBackend) GetDB() (*Database, error) {
	if err := sb.initDB(); err != nil {
		return nil, err
	}
	return sb.db, nil
}

// Close closes the database connection
func (sb *SQLiteBackend) Close() error {
	if sb.db != nil {
		return sb.db.Close()
	}
	return nil
}

// GetTaskLists retrieves all task lists from local storage
func (sb *SQLiteBackend) GetTaskLists() ([]TaskList, error) {
	db, err := sb.GetDB()
	if err != nil {
		return nil, &SQLiteError{Op: "GetTaskLists", Err: err}
	}

	query := `
		SELECT list_id, list_name, list_color, last_ctag, created_at, modified_at
		FROM list_sync_metadata
		ORDER BY list_name ASC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, &SQLiteError{Op: "GetTaskLists", Err: err}
	}
	defer rows.Close()

	var lists []TaskList
	for rows.Next() {
		var list TaskList
		var createdAt, modifiedAt sql.NullInt64
		var ctag sql.NullString

		err := rows.Scan(
			&list.ID,
			&list.Name,
			&list.Color,
			&ctag,
			&createdAt,
			&modifiedAt,
		)
		if err != nil {
			return nil, &SQLiteError{Op: "GetTaskLists", Err: err}
		}

		if ctag.Valid {
			list.CTags = ctag.String
		}

		lists = append(lists, list)
	}

	if err = rows.Err(); err != nil {
		return nil, &SQLiteError{Op: "GetTaskLists", Err: err}
	}

	return lists, nil
}

// GetTasks retrieves tasks from a list with optional filtering
func (sb *SQLiteBackend) GetTasks(listID string, taskFilter *TaskFilter) ([]Task, error) {
	db, err := sb.GetDB()
	if err != nil {
		return nil, &SQLiteError{Op: "GetTasks", ListID: listID, Err: err}
	}

	// Build query with filters
	query := `
		SELECT id, list_id, summary, description, status, priority,
		       created_at, modified_at, due_date, start_date, completed_at,
		       parent_uid, categories
		FROM tasks
		WHERE list_id = ?
	`

	args := []interface{}{listID}
	query, args = sb.applyFilters(query, args, taskFilter)
	query += " ORDER BY priority ASC, created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, &SQLiteError{Op: "GetTasks", ListID: listID, Err: err}
	}
	defer rows.Close()

	tasks, err := sb.scanTasks(rows)
	if err != nil {
		return nil, &SQLiteError{Op: "GetTasks", ListID: listID, Err: err}
	}

	return tasks, nil
}

// applyFilters adds WHERE clauses for task filtering
func (sb *SQLiteBackend) applyFilters(query string, args []interface{}, filter *TaskFilter) (string, []interface{}) {
	if filter == nil {
		return query, args
	}

	// Status filter
	if filter.Statuses != nil && len(*filter.Statuses) > 0 {
		placeholders := make([]string, len(*filter.Statuses))
		for i, status := range *filter.Statuses {
			placeholders[i] = "?"
			args = append(args, status)
		}
		query += fmt.Sprintf(" AND status IN (%s)", strings.Join(placeholders, ","))
	}

	// Due date filters
	if filter.DueBefore != nil {
		query += " AND due_date <= ?"
		args = append(args, filter.DueBefore.Unix())
	}
	if filter.DueAfter != nil {
		query += " AND due_date >= ?"
		args = append(args, filter.DueAfter.Unix())
	}

	// Created date filters
	if filter.CreatedBefore != nil {
		query += " AND created_at <= ?"
		args = append(args, filter.CreatedBefore.Unix())
	}
	if filter.CreatedAfter != nil {
		query += " AND created_at >= ?"
		args = append(args, filter.CreatedAfter.Unix())
	}

	// Priority filter (if we add it to TaskFilter in future)
	// Categories filter would need LIKE queries for the categories TEXT field

	return query, args
}

// scanTasks scans task rows from a query result
func (sb *SQLiteBackend) scanTasks(rows *sql.Rows) ([]Task, error) {
	var tasks []Task

	for rows.Next() {
		var task Task
		var listID string // Temporary variable for list_id (not stored in Task struct)
		var description, parentUID, categories sql.NullString
		var createdAt, modifiedAt, dueDate, startDate, completedAt sql.NullInt64

		err := rows.Scan(
			&task.UID,
			&listID, // Scan list_id but don't store in Task
			&task.Summary,
			&description,
			&task.Status,
			&task.Priority,
			&createdAt,
			&modifiedAt,
			&dueDate,
			&startDate,
			&completedAt,
			&parentUID,
			&categories,
		)
		if err != nil {
			return nil, err
		}

		// Handle nullable fields
		if description.Valid {
			task.Description = description.String
		}
		if parentUID.Valid {
			task.ParentUID = parentUID.String
		}
		if categories.Valid && categories.String != "" {
			task.Categories = strings.Split(categories.String, ",")
		}

		// Convert timestamps
		if createdAt.Valid {
			task.Created = time.Unix(createdAt.Int64, 0)
		}
		if modifiedAt.Valid {
			task.Modified = time.Unix(modifiedAt.Int64, 0)
		}
		if dueDate.Valid {
			t := time.Unix(dueDate.Int64, 0)
			task.DueDate = &t
		}
		if startDate.Valid {
			t := time.Unix(startDate.Int64, 0)
			task.StartDate = &t
		}
		if completedAt.Valid {
			t := time.Unix(completedAt.Int64, 0)
			task.Completed = &t
		}

		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// FindTasksBySummary searches for tasks by summary (case-insensitive)
func (sb *SQLiteBackend) FindTasksBySummary(listID string, summary string) ([]Task, error) {
	db, err := sb.GetDB()
	if err != nil {
		return nil, &SQLiteError{Op: "FindTasksBySummary", ListID: listID, Err: err}
	}

	query := `
		SELECT id, list_id, summary, description, status, priority,
		       created_at, modified_at, due_date, start_date, completed_at,
		       parent_uid, categories
		FROM tasks
		WHERE list_id = ? AND LOWER(summary) LIKE LOWER(?)
		ORDER BY
			CASE WHEN LOWER(summary) = LOWER(?) THEN 0 ELSE 1 END,
			priority ASC,
			created_at DESC
	`

	searchPattern := "%" + summary + "%"
	rows, err := db.Query(query, listID, searchPattern, summary)
	if err != nil {
		return nil, &SQLiteError{Op: "FindTasksBySummary", ListID: listID, Err: err}
	}
	defer rows.Close()

	tasks, err := sb.scanTasks(rows)
	if err != nil {
		return nil, &SQLiteError{Op: "FindTasksBySummary", ListID: listID, Err: err}
	}

	return tasks, nil
}

// AddTask creates a new task in the database
func (sb *SQLiteBackend) AddTask(listID string, task Task) error {
	db, err := sb.GetDB()
	if err != nil {
		return &SQLiteError{Op: "AddTask", ListID: listID, Err: err}
	}

	// Generate UID if not provided
	if task.UID == "" {
		task.UID = generateUID()
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return &SQLiteError{Op: "AddTask", ListID: listID, TaskUID: task.UID, Err: err}
	}
	defer tx.Rollback()

	// Set timestamps
	now := time.Now()
	if task.Created.IsZero() {
		task.Created = now
	}
	if task.Modified.IsZero() {
		task.Modified = now
	}

	// Insert task
	query := `
		INSERT INTO tasks (
			id, list_id, summary, description, status, priority,
			created_at, modified_at, due_date, start_date, completed_at,
			parent_uid, categories
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.Exec(query,
		task.UID,
		listID,
		task.Summary,
		nullString(task.Description),
		task.Status,
		task.Priority,
		timeValueToNullInt64(task.Created),
		timeValueToNullInt64(task.Modified),
		timeToNullInt64(task.DueDate),
		timeToNullInt64(task.StartDate),
		timeToNullInt64(task.Completed),
		nullString(task.ParentUID),
		nullString(strings.Join(task.Categories, ",")),
	)
	if err != nil {
		return &SQLiteError{Op: "AddTask", ListID: listID, TaskUID: task.UID, Err: err}
	}

	// Insert sync metadata
	_, err = tx.Exec(`
		INSERT INTO sync_metadata (task_uid, list_id, locally_modified, local_modified_at)
		VALUES (?, ?, 1, ?)
	`, task.UID, listID, now.Unix())
	if err != nil {
		return &SQLiteError{Op: "AddTask", ListID: listID, TaskUID: task.UID, Err: err}
	}

	// Queue sync operation
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO sync_queue (task_uid, list_id, operation, created_at)
		VALUES (?, ?, 'create', ?)
	`, task.UID, listID, now.Unix())
	if err != nil {
		return &SQLiteError{Op: "AddTask", ListID: listID, TaskUID: task.UID, Err: err}
	}

	return tx.Commit()
}

// UpdateTask updates an existing task
func (sb *SQLiteBackend) UpdateTask(listID string, task Task) error {
	db, err := sb.GetDB()
	if err != nil {
		return &SQLiteError{Op: "UpdateTask", ListID: listID, TaskUID: task.UID, Err: err}
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return &SQLiteError{Op: "UpdateTask", ListID: listID, TaskUID: task.UID, Err: err}
	}
	defer tx.Rollback()

	// Update modified timestamp
	now := time.Now()
	task.Modified = now

	// Update task
	query := `
		UPDATE tasks
		SET summary = ?, description = ?, status = ?, priority = ?,
		    modified_at = ?, due_date = ?, start_date = ?, completed_at = ?,
		    parent_uid = ?, categories = ?
		WHERE id = ? AND list_id = ?
	`

	result, err := tx.Exec(query,
		task.Summary,
		nullString(task.Description),
		task.Status,
		task.Priority,
		timeValueToNullInt64(task.Modified),
		timeToNullInt64(task.DueDate),
		timeToNullInt64(task.StartDate),
		timeToNullInt64(task.Completed),
		nullString(task.ParentUID),
		nullString(strings.Join(task.Categories, ",")),
		task.UID,
		listID,
	)
	if err != nil {
		return &SQLiteError{Op: "UpdateTask", ListID: listID, TaskUID: task.UID, Err: err}
	}

	// Check if task exists
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return &SQLiteError{Op: "UpdateTask", ListID: listID, TaskUID: task.UID, Err: err}
	}
	if rowsAffected == 0 {
		return NewBackendError("UpdateTask", 404, fmt.Sprintf("task %s not found in list %s", task.UID, listID))
	}

	// Update sync metadata
	_, err = tx.Exec(`
		UPDATE sync_metadata
		SET locally_modified = 1, local_modified_at = ?
		WHERE task_uid = ?
	`, now.Unix(), task.UID)
	if err != nil {
		return &SQLiteError{Op: "UpdateTask", ListID: listID, TaskUID: task.UID, Err: err}
	}

	// Queue sync operation
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO sync_queue (task_uid, list_id, operation, created_at)
		VALUES (?, ?, 'update', ?)
	`, task.UID, listID, now.Unix())
	if err != nil {
		return &SQLiteError{Op: "UpdateTask", ListID: listID, TaskUID: task.UID, Err: err}
	}

	return tx.Commit()
}

// DeleteTask removes a task from the database
func (sb *SQLiteBackend) DeleteTask(listID string, taskUID string) error {
	db, err := sb.GetDB()
	if err != nil {
		return &SQLiteError{Op: "DeleteTask", ListID: listID, TaskUID: taskUID, Err: err}
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return &SQLiteError{Op: "DeleteTask", ListID: listID, TaskUID: taskUID, Err: err}
	}
	defer tx.Rollback()

	// Check if task exists
	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM tasks WHERE id = ? AND list_id = ?)", taskUID, listID).Scan(&exists)
	if err != nil {
		return &SQLiteError{Op: "DeleteTask", ListID: listID, TaskUID: taskUID, Err: err}
	}
	if !exists {
		return NewBackendError("DeleteTask", 404, fmt.Sprintf("task %s not found in list %s", taskUID, listID))
	}

	// Mark as locally deleted (soft delete for sync)
	now := time.Now().Unix()
	_, err = tx.Exec(`
		UPDATE sync_metadata
		SET locally_deleted = 1, local_modified_at = ?
		WHERE task_uid = ?
	`, now, taskUID)
	if err != nil {
		return &SQLiteError{Op: "DeleteTask", ListID: listID, TaskUID: taskUID, Err: err}
	}

	// Queue delete operation
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO sync_queue (task_uid, list_id, operation, created_at)
		VALUES (?, ?, 'delete', ?)
	`, taskUID, listID, now)
	if err != nil {
		return &SQLiteError{Op: "DeleteTask", ListID: listID, TaskUID: taskUID, Err: err}
	}

	// Delete task (cascade will delete sync_metadata)
	_, err = tx.Exec("DELETE FROM tasks WHERE id = ? AND list_id = ?", taskUID, listID)
	if err != nil {
		return &SQLiteError{Op: "DeleteTask", ListID: listID, TaskUID: taskUID, Err: err}
	}

	return tx.Commit()
}

// CreateTaskList creates a new task list
func (sb *SQLiteBackend) CreateTaskList(name, description, color string) (string, error) {
	db, err := sb.GetDB()
	if err != nil {
		return "", &SQLiteError{Op: "CreateTaskList", Err: err}
	}

	listID := generateUID()
	now := time.Now().Unix()

	_, err = db.Exec(`
		INSERT INTO list_sync_metadata (list_id, list_name, list_color, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?)
	`, listID, name, color, now, now)
	if err != nil {
		return "", &SQLiteError{Op: "CreateTaskList", Err: err}
	}

	return listID, nil
}

// DeleteTaskList removes a task list and all its tasks
func (sb *SQLiteBackend) DeleteTaskList(listID string) error {
	db, err := sb.GetDB()
	if err != nil {
		return &SQLiteError{Op: "DeleteTaskList", ListID: listID, Err: err}
	}

	tx, err := db.Begin()
	if err != nil {
		return &SQLiteError{Op: "DeleteTaskList", ListID: listID, Err: err}
	}
	defer tx.Rollback()

	// Delete all tasks in the list (cascade will delete sync_metadata)
	_, err = tx.Exec("DELETE FROM tasks WHERE list_id = ?", listID)
	if err != nil {
		return &SQLiteError{Op: "DeleteTaskList", ListID: listID, Err: err}
	}

	// Delete list metadata
	result, err := tx.Exec("DELETE FROM list_sync_metadata WHERE list_id = ?", listID)
	if err != nil {
		return &SQLiteError{Op: "DeleteTaskList", ListID: listID, Err: err}
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return &SQLiteError{Op: "DeleteTaskList", ListID: listID, Err: err}
	}
	if rowsAffected == 0 {
		return NewBackendError("DeleteTaskList", 404, fmt.Sprintf("list %s not found", listID))
	}

	return tx.Commit()
}

// RenameTaskList renames a task list
func (sb *SQLiteBackend) RenameTaskList(listID, newName string) error {
	db, err := sb.GetDB()
	if err != nil {
		return &SQLiteError{Op: "RenameTaskList", ListID: listID, Err: err}
	}

	result, err := db.Exec(`
		UPDATE list_sync_metadata
		SET list_name = ?, modified_at = ?
		WHERE list_id = ?
	`, newName, time.Now().Unix(), listID)
	if err != nil {
		return &SQLiteError{Op: "RenameTaskList", ListID: listID, Err: err}
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return &SQLiteError{Op: "RenameTaskList", ListID: listID, Err: err}
	}
	if rowsAffected == 0 {
		return NewBackendError("RenameTaskList", 404, fmt.Sprintf("list %s not found", listID))
	}

	return nil
}

// GetDeletedTaskLists returns deleted task lists (not supported for SQLite yet)
func (sb *SQLiteBackend) GetDeletedTaskLists() ([]TaskList, error) {
	// SQLite backend doesn't support trash yet
	return []TaskList{}, nil
}

// RestoreTaskList restores a deleted task list (not supported for SQLite yet)
func (sb *SQLiteBackend) RestoreTaskList(listID string) error {
	return fmt.Errorf("trash not supported for SQLite backend")
}

// PermanentlyDeleteTaskList permanently deletes a task list (not supported for SQLite yet)
func (sb *SQLiteBackend) PermanentlyDeleteTaskList(listID string) error {
	return fmt.Errorf("trash not supported for SQLite backend")
}

// ParseStatusFlag converts status abbreviations to backend status format
func (sb *SQLiteBackend) ParseStatusFlag(statusFlag string) (string, error) {
	// SQLite uses standard CalDAV status names
	flag := strings.ToUpper(statusFlag)

	switch flag {
	case "T", "TODO", "NEEDS-ACTION":
		return "NEEDS-ACTION", nil
	case "D", "DONE", "COMPLETED":
		return "COMPLETED", nil
	case "P", "PROCESSING", "IN-PROCESS":
		return "IN-PROCESS", nil
	case "C", "CANCELLED":
		return "CANCELLED", nil
	default:
		return "", fmt.Errorf("invalid status flag: %s (valid: T/TODO, D/DONE, P/PROCESSING, C/CANCELLED)", statusFlag)
	}
}

// StatusToDisplayName converts backend status to display name
func (sb *SQLiteBackend) StatusToDisplayName(backendStatus string) string {
	switch strings.ToUpper(backendStatus) {
	case "NEEDS-ACTION":
		return "TODO"
	case "COMPLETED":
		return "DONE"
	case "IN-PROCESS":
		return "PROCESSING"
	case "CANCELLED":
		return "CANCELLED"
	default:
		return backendStatus
	}
}

// SortTasks sorts tasks by priority (1=highest, 0=undefined goes last)
func (sb *SQLiteBackend) SortTasks(tasks []Task) {
	// Use the same sorting as Nextcloud (priority-based)
	// This is already handled in GetTasks ORDER BY clause
	// But we implement it here for consistency
	for i := 0; i < len(tasks)-1; i++ {
		for j := i + 1; j < len(tasks); j++ {
			// Priority 0 (undefined) goes last
			if tasks[i].Priority == 0 && tasks[j].Priority != 0 {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			} else if tasks[i].Priority != 0 && tasks[j].Priority != 0 && tasks[i].Priority > tasks[j].Priority {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}
}

// GetPriorityColor returns ANSI color code for priority
func (sb *SQLiteBackend) GetPriorityColor(priority int) string {
	// Use same color scheme as Nextcloud
	if priority >= 1 && priority <= 4 {
		return "\033[31m" // Red (high priority)
	} else if priority == 5 {
		return "\033[33m" // Yellow (medium priority)
	} else if priority >= 6 && priority <= 9 {
		return "\033[34m" // Blue (low priority)
	}
	return "" // No color for priority 0 (undefined)
}

// GetBackendDisplayName returns a formatted string for display in task list headers
func (sb *SQLiteBackend) GetBackendDisplayName() string {
	dbPath := sb.Config.DBPath
	if dbPath == "" {
		dbPath, _ = getDatabasePath(sb.Config.DBPath)
	}
	return fmt.Sprintf("[sqlite:%s]", dbPath)
}

// GetBackendType returns the backend type identifier
func (sb *SQLiteBackend) GetBackendType() string {
	return "sqlite"
}

// GetBackendContext returns contextual details specific to the backend
func (sb *SQLiteBackend) GetBackendContext() string {
	dbPath := sb.Config.DBPath
	if dbPath == "" {
		dbPath, _ = getDatabasePath(sb.Config.DBPath)
	}
	return dbPath
}

// Sync-specific methods

// MarkLocallyModified marks a task as locally modified
func (sb *SQLiteBackend) MarkLocallyModified(taskUID string) error {
	db, err := sb.GetDB()
	if err != nil {
		return &SQLiteError{Op: "MarkLocallyModified", TaskUID: taskUID, Err: err}
	}

	_, err = db.Exec(`
		UPDATE sync_metadata
		SET locally_modified = 1, local_modified_at = ?
		WHERE task_uid = ?
	`, time.Now().Unix(), taskUID)
	if err != nil {
		return &SQLiteError{Op: "MarkLocallyModified", TaskUID: taskUID, Err: err}
	}

	return nil
}

// MarkLocallyDeleted marks a task as locally deleted
func (sb *SQLiteBackend) MarkLocallyDeleted(taskUID string) error {
	db, err := sb.GetDB()
	if err != nil {
		return &SQLiteError{Op: "MarkLocallyDeleted", TaskUID: taskUID, Err: err}
	}

	_, err = db.Exec(`
		UPDATE sync_metadata
		SET locally_deleted = 1, local_modified_at = ?
		WHERE task_uid = ?
	`, time.Now().Unix(), taskUID)
	if err != nil {
		return &SQLiteError{Op: "MarkLocallyDeleted", TaskUID: taskUID, Err: err}
	}

	return nil
}

// GetLocallyModifiedTasks retrieves tasks that have been modified locally
func (sb *SQLiteBackend) GetLocallyModifiedTasks() ([]Task, error) {
	db, err := sb.GetDB()
	if err != nil {
		return nil, &SQLiteError{Op: "GetLocallyModifiedTasks", Err: err}
	}

	query := `
		SELECT t.id, t.list_id, t.summary, t.description, t.status, t.priority,
		       t.created_at, t.modified_at, t.due_date, t.start_date, t.completed_at,
		       t.parent_uid, t.categories
		FROM tasks t
		INNER JOIN sync_metadata sm ON t.id = sm.task_uid
		WHERE sm.locally_modified = 1
		ORDER BY sm.local_modified_at ASC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, &SQLiteError{Op: "GetLocallyModifiedTasks", Err: err}
	}
	defer rows.Close()

	tasks, err := sb.scanTasks(rows)
	if err != nil {
		return nil, &SQLiteError{Op: "GetLocallyModifiedTasks", Err: err}
	}

	return tasks, nil
}

// SyncOperation represents a pending sync operation
type SyncOperation struct {
	ID         int
	TaskUID    string
	ListID     string
	Operation  string // "create", "update", "delete"
	CreatedAt  time.Time
	RetryCount int
	LastError  string
}

// GetPendingSyncOperations retrieves operations queued for sync
func (sb *SQLiteBackend) GetPendingSyncOperations() ([]SyncOperation, error) {
	db, err := sb.GetDB()
	if err != nil {
		return nil, &SQLiteError{Op: "GetPendingSyncOperations", Err: err}
	}

	query := `
		SELECT id, task_uid, list_id, operation, created_at, retry_count, last_error
		FROM sync_queue
		ORDER BY created_at ASC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, &SQLiteError{Op: "GetPendingSyncOperations", Err: err}
	}
	defer rows.Close()

	var operations []SyncOperation
	for rows.Next() {
		var op SyncOperation
		var createdAt int64
		var lastError sql.NullString

		err := rows.Scan(
			&op.ID,
			&op.TaskUID,
			&op.ListID,
			&op.Operation,
			&createdAt,
			&op.RetryCount,
			&lastError,
		)
		if err != nil {
			return nil, &SQLiteError{Op: "GetPendingSyncOperations", Err: err}
		}

		op.CreatedAt = time.Unix(createdAt, 0)
		if lastError.Valid {
			op.LastError = lastError.String
		}

		operations = append(operations, op)
	}

	return operations, rows.Err()
}

// ClearSyncFlags clears locally_modified and locally_deleted flags for a task
// Note: This does NOT remove pending sync operations from the queue.
// Use ClearSyncFlagsAndQueue() if you need to remove queue entries as well.
func (sb *SQLiteBackend) ClearSyncFlags(taskUID string) error {
	db, err := sb.GetDB()
	if err != nil {
		return &SQLiteError{Op: "ClearSyncFlags", TaskUID: taskUID, Err: err}
	}

	_, err = db.Exec(`
		UPDATE sync_metadata
		SET locally_modified = 0, locally_deleted = 0
		WHERE task_uid = ?
	`, taskUID)
	if err != nil {
		return &SQLiteError{Op: "ClearSyncFlags", TaskUID: taskUID, Err: err}
	}

	return nil
}

// ClearSyncFlagsAndQueue clears locally_modified and locally_deleted flags for a task
// and removes all pending sync operations from the queue.
// This should be called after successfully pushing all operations for a task.
func (sb *SQLiteBackend) ClearSyncFlagsAndQueue(taskUID string) error {
	db, err := sb.GetDB()
	if err != nil {
		return &SQLiteError{Op: "ClearSyncFlagsAndQueue", TaskUID: taskUID, Err: err}
	}

	// Start transaction to ensure both operations succeed or fail together
	tx, err := db.Begin()
	if err != nil {
		return &SQLiteError{Op: "ClearSyncFlagsAndQueue", TaskUID: taskUID, Err: err}
	}
	defer tx.Rollback()

	// Get current task modified timestamp to update remote_modified_at
	var modifiedAt sql.NullInt64
	err = tx.QueryRow(`
		SELECT modified_at
		FROM tasks
		WHERE id = ?
	`, taskUID).Scan(&modifiedAt)
	if err != nil {
		return &SQLiteError{Op: "ClearSyncFlags", TaskUID: taskUID, Err: err}
	}

	// Clear sync metadata flags and update remote_modified_at
	// This indicates the task is now in sync with remote at this timestamp
	_, err = tx.Exec(`
		UPDATE sync_metadata
		SET locally_modified = 0, locally_deleted = 0, remote_modified_at = ?
		WHERE task_uid = ?
	`, modifiedAt, taskUID)
	if err != nil {
		return &SQLiteError{Op: "ClearSyncFlagsAndQueue", TaskUID: taskUID, Err: err}
	}

	// Remove all pending sync operations for this task
	_, err = tx.Exec(`
		DELETE FROM sync_queue
		WHERE task_uid = ?
	`, taskUID)
	if err != nil {
		return &SQLiteError{Op: "ClearSyncFlagsAndQueue", TaskUID: taskUID, Err: err}
	}

	return tx.Commit()
}

// UpdateSyncMetadata updates sync metadata for a task
func (sb *SQLiteBackend) UpdateSyncMetadata(taskUID, listID, etag string, remoteModifiedAt time.Time) error {
	db, err := sb.GetDB()
	if err != nil {
		return &SQLiteError{Op: "UpdateSyncMetadata", TaskUID: taskUID, Err: err}
	}

	now := time.Now().Unix()

	_, err = db.Exec(`
		INSERT INTO sync_metadata (
			task_uid, list_id, remote_etag, last_synced_at,
			remote_modified_at, locally_modified, locally_deleted
		) VALUES (?, ?, ?, ?, ?, 0, 0)
		ON CONFLICT(task_uid) DO UPDATE SET
			remote_etag = excluded.remote_etag,
			last_synced_at = excluded.last_synced_at,
			remote_modified_at = excluded.remote_modified_at
	`, taskUID, listID, etag, now, remoteModifiedAt.Unix())
	if err != nil {
		return &SQLiteError{Op: "UpdateSyncMetadata", TaskUID: taskUID, Err: err}
	}

	return nil
}

// RemoveSyncOperation removes a sync operation from the queue
func (sb *SQLiteBackend) RemoveSyncOperation(taskUID, operation string) error {
	db, err := sb.GetDB()
	if err != nil {
		return &SQLiteError{Op: "RemoveSyncOperation", TaskUID: taskUID, Err: err}
	}

	_, err = db.Exec(`
		DELETE FROM sync_queue
		WHERE task_uid = ? AND operation = ?
	`, taskUID, operation)
	if err != nil {
		return &SQLiteError{Op: "RemoveSyncOperation", TaskUID: taskUID, Err: err}
	}

	return nil
}

// Helper functions

// generateUID generates a unique identifier for tasks/lists
func generateUID() string {
	return fmt.Sprintf("task-%d-%s", time.Now().Unix(), randomString(8))
}

// randomString generates a random alphanumeric string
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// nullString converts string to sql.NullString
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// timeToNullInt64 converts *time.Time to sql.NullInt64
func timeToNullInt64(t *time.Time) sql.NullInt64 {
	if t == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: t.Unix(), Valid: true}
}

// timeValueToNullInt64 converts time.Time (non-pointer) to sql.NullInt64
func timeValueToNullInt64(t time.Time) sql.NullInt64 {
	if t.IsZero() {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: t.Unix(), Valid: true}
}
