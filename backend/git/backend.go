package git

import (
	"gosynctasks/backend"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func init() {
	// Register Git backend for config type "git"
	// Git backend supports auto-detection
	backend.RegisterDetectable("git", newGitBackendWrapper)
}

// newGitBackendWrapper wraps NewGitBackend to match BackendConfigConstructor signature
func newGitBackendWrapper(config backend.BackendConfig) (backend.TaskManager, error) {
	return NewGitBackend(config)
}

// GitBackend implements backend.TaskManager for git repositories with markdown task files.
// Tasks are stored in markdown format with a special marker to enable gosynctasks.
type GitBackend struct {
	config       backend.BackendConfig
	RepoPath     string            // Absolute path to git repository root
	FilePath     string            // Absolute path to task file (e.g., TODO.md)
	taskLists    map[string][]backend.Task // Tasks organized by list name (## headers)
	fileModTime  time.Time         // Last modification time of file
	detectedInfo string            // Human-readable detection info
}

const (
	// Marker that must be present in markdown file to enable gosynctasks
	gitBackendMarker = "<!-- gosynctasks:enabled -->"
)

// NewGitBackend creates a new Git backend instance.
func NewGitBackend(config backend.BackendConfig) (*GitBackend, error) {
	gb := &GitBackend{
		config:    config,
		taskLists: make(map[string][]backend.Task),
	}

	// Find git repository
	repoPath, err := gb.findGitRepo()
	if err != nil {
		return nil, fmt.Errorf("git repository not found: %w", err)
	}
	gb.RepoPath = repoPath

	// Find TODO file
	filePath, err := gb.findTodoFile()
	if err != nil {
		return nil, fmt.Errorf("TODO file not found: %w", err)
	}
	gb.FilePath = filePath

	// Load tasks from file
	if err := gb.loadFile(); err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	gb.detectedInfo = fmt.Sprintf("Git repository at %s with task file %s",
		filepath.Base(gb.RepoPath), filepath.Base(gb.FilePath))

	return gb, nil
}

// findGitRepo finds the git repository root by walking up the directory tree.
func (gb *GitBackend) findGitRepo() (string, error) {
	// Start from current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up directory tree looking for .git
	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil {
			// Found .git directory or file (submodule)
			if info.IsDir() || !info.IsDir() {
				return dir, nil
			}
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding .git
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not in a git repository")
}

// findTodoFile searches for the task file based on config.
func (gb *GitBackend) findTodoFile() (string, error) {
	// Build list of files to try
	filesToTry := []string{}

	// Add configured file if specified
	if gb.config.File != "" {
		filesToTry = append(filesToTry, gb.config.File)
	}

	// Add fallback files
	if len(gb.config.FallbackFiles) > 0 {
		filesToTry = append(filesToTry, gb.config.FallbackFiles...)
	}

	// Default fallbacks if nothing configured
	if len(filesToTry) == 0 {
		filesToTry = []string{"TODO.md", "todo.md", ".gosynctasks.md"}
	}

	// Try each file
	for _, filename := range filesToTry {
		fullPath := filepath.Join(gb.RepoPath, filename)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			// File exists, check for marker
			content, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}

			if gb.hasMarker(string(content)) {
				return fullPath, nil
			}
		}
	}

	return "", fmt.Errorf("no TODO file with gosynctasks marker found (tried: %s)",
		strings.Join(filesToTry, ", "))
}

// hasMarker checks if content contains the gosynctasks marker.
func (gb *GitBackend) hasMarker(content string) bool {
	return strings.Contains(content, gitBackendMarker)
}

// loadFile reads and parses the markdown file.
func (gb *GitBackend) loadFile() error {
	content, err := os.ReadFile(gb.FilePath)
	if err != nil {
		return err
	}

	// Get file modification time
	info, err := os.Stat(gb.FilePath)
	if err != nil {
		return err
	}
	gb.fileModTime = info.ModTime()

	// Parse markdown
	parser := NewMarkdownParser()
	taskLists, err := parser.Parse(string(content))
	if err != nil {
		return err
	}

	gb.taskLists = taskLists
	return nil
}

// saveFile writes tasks back to the markdown file.
func (gb *GitBackend) saveFile() error {
	writer := NewMarkdownWriter()
	content := writer.Write(gb.taskLists)

	// Check if file was modified externally
	if info, err := os.Stat(gb.FilePath); err == nil {
		if info.ModTime().After(gb.fileModTime) {
			return fmt.Errorf("file was modified externally, refusing to overwrite")
		}
	}

	// Write to file
	if err := os.WriteFile(gb.FilePath, []byte(content), 0644); err != nil {
		return err
	}

	// Update modification time
	if info, err := os.Stat(gb.FilePath); err == nil {
		gb.fileModTime = info.ModTime()
	}

	// Auto-commit if enabled
	if gb.config.AutoCommit {
		return gb.commitChanges()
	}

	return nil
}

// commitChanges commits the task file to git.
func (gb *GitBackend) commitChanges() error {
	// Add file
	cmd := exec.Command("git", "add", gb.FilePath)
	cmd.Dir = gb.RepoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// Check if there are changes to commit
	cmd = exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = gb.RepoPath
	if err := cmd.Run(); err == nil {
		// No changes to commit
		return nil
	}

	// Commit
	commitMsg := fmt.Sprintf("gosynctasks: Update tasks in %s", filepath.Base(gb.FilePath))
	cmd = exec.Command("git", "commit", "-m", commitMsg)
	cmd.Dir = gb.RepoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	return nil
}

// generateUID generates a unique task ID.
func (gb *GitBackend) generateUID() string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 4)
	_, _ = rand.Read(randomBytes)
	randomHex := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("task-%d-%s", timestamp, randomHex)
}

// CanDetect checks if this backend can be used in the current environment.
func (gb *GitBackend) CanDetect() (bool, error) {
	// Try to find git repo
	if _, err := gb.findGitRepo(); err != nil {
		return false, nil
	}

	// Try to find TODO file
	if _, err := gb.findTodoFile(); err != nil {
		return false, nil
	}

	return true, nil
}

// DetectionInfo returns human-readable detection information.
func (gb *GitBackend) DetectionInfo() string {
	return gb.detectedInfo
}

// GetTaskLists retrieves all task lists (headers) from the markdown file.
func (gb *GitBackend) GetTaskLists() ([]backend.TaskList, error) {
	// Reload file to get latest changes
	if err := gb.loadFile(); err != nil {
		return nil, err
	}

	var lists []backend.TaskList
	for name := range gb.taskLists {
		lists = append(lists, backend.TaskList{
			ID:          name,
			Name:        name,
			Description: fmt.Sprintf("%d tasks", len(gb.taskLists[name])),
		})
	}

	return lists, nil
}

// GetTasks retrieves tasks from a specific list with optional filtering.
func (gb *GitBackend) GetTasks(listID string, filter *backend.TaskFilter) ([]backend.Task, error) {
	// Reload file to get latest changes
	if err := gb.loadFile(); err != nil {
		return nil, err
	}

	tasks, exists := gb.taskLists[listID]
	if !exists {
		return nil, fmt.Errorf("task list %q not found", listID)
	}

	// Apply filter if provided
	if filter != nil {
		tasks = gb.filterTasks(tasks, filter)
	}

	// Sort tasks
	gb.SortTasks(tasks)

	return tasks, nil
}

// filterTasks applies a backend.TaskFilter to a slice of tasks.
func (gb *GitBackend) filterTasks(tasks []backend.Task, filter *backend.TaskFilter) []backend.Task {
	var filtered []backend.Task

	for _, task := range tasks {
		// Check status filter
		if filter.Statuses != nil && len(*filter.Statuses) > 0 {
			matchesStatus := false
			for _, status := range *filter.Statuses {
				if task.Status == status {
					matchesStatus = true
					break
				}
			}
			if !matchesStatus {
				continue
			}
		}

		// Check due date filters
		if filter.DueAfter != nil && !task.DueDate.IsZero() {
			if task.DueDate.Before(*filter.DueAfter) {
				continue
			}
		}

		if filter.DueBefore != nil && !task.DueDate.IsZero() {
			if task.DueDate.After(*filter.DueBefore) {
				continue
			}
		}

		// Check created after filter
		if filter.CreatedAfter != nil && !task.Created.IsZero() {
			if task.Created.Before(*filter.CreatedAfter) {
				continue
			}
		}

		filtered = append(filtered, task)
	}

	return filtered
}

// FindTasksBySummary searches for tasks by summary text.
func (gb *GitBackend) FindTasksBySummary(listID string, summary string) ([]backend.Task, error) {
	tasks, err := gb.GetTasks(listID, nil)
	if err != nil {
		return nil, err
	}

	summary = strings.ToLower(summary)
	var matches []backend.Task

	for _, task := range tasks {
		if strings.Contains(strings.ToLower(task.Summary), summary) {
			matches = append(matches, task)
		}
	}

	return matches, nil
}

// AddTask creates a new task in the specified list.
func (gb *GitBackend) AddTask(listID string, task backend.Task) (string, error) {
	// Reload file to get latest changes
	if err := gb.loadFile(); err != nil {
		return "", err
	}

	// Generate UID if not provided
	if task.UID == "" {
		task.UID = gb.generateUID()
	}

	// Set timestamps
	if task.Created.IsZero() {
		task.Created = time.Now()
	}
	task.Modified = time.Now()

	// Add task to list
	gb.taskLists[listID] = append(gb.taskLists[listID], task)

	// Save file
	if err := gb.saveFile(); err != nil {
		return "", err
	}

	return task.UID, nil
}

// UpdateTask modifies an existing task.
func (gb *GitBackend) UpdateTask(listID string, task backend.Task) error {
	// Reload file to get latest changes
	if err := gb.loadFile(); err != nil {
		return err
	}

	tasks, exists := gb.taskLists[listID]
	if !exists {
		return fmt.Errorf("task list %q not found", listID)
	}

	// Find and update task
	found := false
	for i, t := range tasks {
		if t.UID == task.UID {
			task.Modified = time.Now()
			tasks[i] = task
			found = true
			break
		}
	}

	if !found {
		return backend.NewBackendError("UpdateTask", 404, fmt.Sprintf("task %q not found", task.UID))
	}

	gb.taskLists[listID] = tasks

	// Save file
	return gb.saveFile()
}

// DeleteTask removes a task from the specified list.
func (gb *GitBackend) DeleteTask(listID string, taskUID string) error {
	// Reload file to get latest changes
	if err := gb.loadFile(); err != nil {
		return err
	}

	tasks, exists := gb.taskLists[listID]
	if !exists {
		return fmt.Errorf("task list %q not found", listID)
	}

	// Find and remove task
	found := false
	for i, t := range tasks {
		if t.UID == taskUID {
			tasks = append(tasks[:i], tasks[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return backend.NewBackendError("DeleteTask", 404, fmt.Sprintf("task %q not found", taskUID))
	}

	gb.taskLists[listID] = tasks

	// Save file
	return gb.saveFile()
}

// CreateTaskList creates a new task list (header) in the markdown file.
func (gb *GitBackend) CreateTaskList(name, description, color string) (string, error) {
	// Reload file to get latest changes
	if err := gb.loadFile(); err != nil {
		return "", err
	}

	// Check if list already exists
	if _, exists := gb.taskLists[name]; exists {
		return "", fmt.Errorf("task list %q already exists", name)
	}

	// Create empty list
	gb.taskLists[name] = []backend.Task{}

	// Save file
	if err := gb.saveFile(); err != nil {
		return "", err
	}

	return name, nil
}

// DeleteTaskList removes a task list (header) and all its tasks from the markdown file.
func (gb *GitBackend) DeleteTaskList(listID string) error {
	// Reload file to get latest changes
	if err := gb.loadFile(); err != nil {
		return err
	}

	// Check if list exists
	if _, exists := gb.taskLists[listID]; !exists {
		return fmt.Errorf("task list %q not found", listID)
	}

	// Delete list
	delete(gb.taskLists, listID)

	// Save file
	return gb.saveFile()
}

// RenameTaskList changes the name of a task list (header) in the markdown file.
func (gb *GitBackend) RenameTaskList(listID, newName string) error {
	// Reload file to get latest changes
	if err := gb.loadFile(); err != nil {
		return err
	}

	// Check if old list exists
	tasks, exists := gb.taskLists[listID]
	if !exists {
		return fmt.Errorf("task list %q not found", listID)
	}

	// Check if new name already exists
	if _, exists := gb.taskLists[newName]; exists {
		return fmt.Errorf("task list %q already exists", newName)
	}

	// Rename by deleting old and creating new
	delete(gb.taskLists, listID)
	gb.taskLists[newName] = tasks

	// Save file
	return gb.saveFile()
}

// ParseStatusFlag converts user input to backend status format.
func (gb *GitBackend) ParseStatusFlag(statusFlag string) (string, error) {
	// Git backend uses app-style status names
	upper := strings.ToUpper(statusFlag)

	// Handle abbreviations
	switch upper {
	case "T":
		return "TODO", nil
	case "D":
		return "DONE", nil
	case "P":
		return "PROCESSING", nil
	case "C":
		return "CANCELLED", nil
	}

	// Handle full names
	switch upper {
	case "TODO", "DONE", "PROCESSING", "CANCELLED":
		return upper, nil
	}

	return "", fmt.Errorf("invalid status flag: %s (use TODO/T, DONE/D, PROCESSING/P, CANCELLED/C)", statusFlag)
}

// StatusToDisplayName converts backend status to display name.
func (gb *GitBackend) StatusToDisplayName(backendStatus string) string {
	// Git backend already uses display names
	return backendStatus
}

// SortTasks sorts tasks by priority (1=highest) and creation date.
func (gb *GitBackend) SortTasks(tasks []backend.Task) {
	// Simple bubble sort (good enough for typical task lists)
	for i := 0; i < len(tasks); i++ {
		for j := i + 1; j < len(tasks); j++ {
			// Priority 0 goes last
			iPrio := tasks[i].Priority
			jPrio := tasks[j].Priority
			if iPrio == 0 {
				iPrio = 100
			}
			if jPrio == 0 {
				jPrio = 100
			}

			// Lower priority number = higher priority
			if iPrio > jPrio {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			} else if iPrio == jPrio {
				// Same priority, sort by creation date (older first)
				if tasks[i].Created.After(tasks[j].Created) {
					tasks[i], tasks[j] = tasks[j], tasks[i]
				}
			}
		}
	}
}

// GetPriorityColor returns ANSI color code for priority.
func (gb *GitBackend) GetPriorityColor(priority int) string {
	// Similar to Nextcloud coloring
	switch {
	case priority >= 1 && priority <= 4:
		return "\033[31m" // Red (high priority)
	case priority == 5:
		return "\033[33m" // Yellow (medium priority)
	case priority >= 6 && priority <= 9:
		return "\033[34m" // Blue (low priority)
	default:
		return "" // No color (undefined priority)
	}
}

// GetBackendDisplayName returns a formatted string for display in task list headers.
func (gb *GitBackend) GetBackendDisplayName() string {
	repoName := filepath.Base(gb.RepoPath)
	fileName := filepath.Base(gb.FilePath)
	return fmt.Sprintf("[git:%s/%s]", repoName, fileName)
}

// GetBackendType returns the backend type identifier.
func (gb *GitBackend) GetBackendType() string {
	return "git"
}

// GetBackendContext returns contextual details specific to the backend.
func (gb *GitBackend) GetBackendContext() string {
	repoName := filepath.Base(gb.RepoPath)
	fileName := filepath.Base(gb.FilePath)
	return fmt.Sprintf("%s/%s", repoName, fileName)
}

// GetDeletedTaskLists retrieves deleted task lists (not supported for Git backend).
func (gb *GitBackend) GetDeletedTaskLists() ([]backend.TaskList, error) {
	// Git backend doesn't support trash functionality
	return []backend.TaskList{}, nil
}

// RestoreTaskList restores a deleted task list (not supported for Git backend).
func (gb *GitBackend) RestoreTaskList(listID string) error {
	return fmt.Errorf("GitBackend.RestoreTaskList not supported")
}

// PermanentlyDeleteTaskList permanently deletes a task list (not supported for Git backend).
func (gb *GitBackend) PermanentlyDeleteTaskList(listID string) error {
	return fmt.Errorf("GitBackend.PermanentlyDeleteTaskList not supported")
}
