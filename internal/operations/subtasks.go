package operations

import (
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/internal/config"
	"strings"
)

// CreateOrFindTaskPath creates a hierarchical path of tasks, creating any missing levels
// Returns the UID of the final parent and the actual task name to create
// Example: "parent/child/task" creates/finds "parent", creates/finds "child" under "parent",
// and returns the UID of "child" and "task" as the name
func CreateOrFindTaskPath(taskManager backend.TaskManager, cfg *config.Config, listID string, path string, taskStatus string) (parentUID string, taskName string, err error) {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return "", "", fmt.Errorf("empty path")
	}

	// If only one part, it's just a regular task (no parent)
	if len(parts) == 1 {
		return "", parts[0], nil
	}

	// The last part is the task name to create
	taskName = strings.TrimSpace(parts[len(parts)-1])
	if taskName == "" {
		return "", "", fmt.Errorf("task name cannot be empty")
	}

	// Create or find each parent level
	var currentParentUID string
	for i := 0; i < len(parts)-1; i++ {
		partName := strings.TrimSpace(parts[i])
		if partName == "" {
			return "", "", fmt.Errorf("empty path segment in '%s'", path)
		}

		// Try to find existing task at this level
		task, err := findTaskByParent(taskManager, cfg, listID, partName, currentParentUID)
		if err != nil {
			// Task doesn't exist - create it
			fmt.Printf("Creating intermediate task '%s'...\n", partName)
			newTask := backend.Task{
				Summary:   partName,
				Status:    taskStatus,
				ParentUID: currentParentUID,
			}
			if err := taskManager.AddTask(listID, newTask); err != nil {
				return "", "", fmt.Errorf("failed to create intermediate task '%s': %w", partName, err)
			}

			// Retrieve the newly created task to get its UID
			task, err = findTaskByParent(taskManager, cfg, listID, partName, currentParentUID)
			if err != nil {
				return "", "", fmt.Errorf("failed to retrieve newly created task '%s': %w", partName, err)
			}
		}

		currentParentUID = task.UID
	}

	return currentParentUID, taskName, nil
}

// ResolveParentTask resolves a parent task reference (simple name or path) to a task UID
// Supports both simple references ("Parent Task") and path-based references ("Feature X/Write code/Fix bug")
func ResolveParentTask(taskManager backend.TaskManager, cfg *config.Config, listID string, parentRef string) (string, error) {
	if parentRef == "" {
		return "", nil
	}

	// Check if it's a path-based reference (contains '/')
	if strings.Contains(parentRef, "/") {
		return resolveParentPath(taskManager, cfg, listID, parentRef)
	}

	// Simple reference - find the task by summary
	task, err := FindTaskBySummary(taskManager, cfg, listID, parentRef)
	if err != nil {
		return "", fmt.Errorf("failed to find parent task '%s': %w", parentRef, err)
	}

	return task.UID, nil
}

// resolveParentPath resolves a hierarchical path like "Feature X/Write code" to find the deepest task
func resolveParentPath(taskManager backend.TaskManager, cfg *config.Config, listID string, path string) (string, error) {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("empty parent path")
	}

	// Start from root level (tasks with no parent)
	var currentParentUID string

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return "", fmt.Errorf("empty path segment in '%s'", path)
		}

		// Find task matching this part with the current parent
		task, err := findTaskByParent(taskManager, cfg, listID, part, currentParentUID)
		if err != nil {
			pathSoFar := strings.Join(parts[:i+1], "/")
			return "", fmt.Errorf("failed to resolve '%s' in path '%s': %w", pathSoFar, path, err)
		}

		currentParentUID = task.UID
	}

	return currentParentUID, nil
}

// findTaskByParent finds a task with the given summary and parent UID
func findTaskByParent(taskManager backend.TaskManager, cfg *config.Config, listID string, summary string, parentUID string) (*backend.Task, error) {
	// Get all tasks and filter by parent
	allTasks, err := taskManager.GetTasks(listID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	// Filter tasks matching summary and parent
	var matches []backend.Task
	summaryLower := strings.ToLower(summary)

	for _, task := range allTasks {
		// Check if parent matches (both empty or both equal)
		parentMatches := (parentUID == "" && task.ParentUID == "") || (task.ParentUID == parentUID)
		if !parentMatches {
			continue
		}

		// Check if summary matches (case-insensitive, allows partial)
		if strings.Contains(strings.ToLower(task.Summary), summaryLower) {
			matches = append(matches, task)
		}
	}

	if len(matches) == 0 {
		if parentUID == "" {
			return nil, fmt.Errorf("no root-level tasks found matching '%s'", summary)
		}
		return nil, fmt.Errorf("no subtasks found matching '%s'", summary)
	}

	// Separate exact and partial matches
	var exactMatches []backend.Task
	var partialMatches []backend.Task

	for _, task := range matches {
		if strings.ToLower(task.Summary) == summaryLower {
			exactMatches = append(exactMatches, task)
		} else {
			partialMatches = append(partialMatches, task)
		}
	}

	// Single exact match - return it
	if len(exactMatches) == 1 && len(partialMatches) == 0 {
		return &exactMatches[0], nil
	}

	// Multiple matches - let user select
	if len(exactMatches) > 1 {
		return selectTaskWithPath(exactMatches, summary, taskManager, cfg, listID)
	}

	if len(exactMatches) == 0 && len(partialMatches) == 1 {
		// Single partial match - confirm
		task := &partialMatches[0]
		confirmed, err := confirmTask(task, taskManager, cfg)
		if err != nil {
			return nil, err
		}
		if !confirmed {
			return nil, fmt.Errorf("operation cancelled")
		}
		return task, nil
	}

	// Multiple matches (exact or partial)
	return selectTaskWithPath(matches, summary, taskManager, cfg, listID)
}

// selectTaskWithPath shows tasks with their hierarchical paths for disambiguation
func selectTaskWithPath(tasks []backend.Task, searchSummary string, taskManager backend.TaskManager, cfg *config.Config, listID string) (*backend.Task, error) {
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks found matching '%s'", searchSummary)
	}

	// Get all tasks to build hierarchy
	allTasks, err := taskManager.GetTasks(listID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks for hierarchy: %w", err)
	}

	// Build UID to task map
	taskMap := make(map[string]*backend.Task)
	for i := range allTasks {
		taskMap[allTasks[i].UID] = &allTasks[i]
	}

	// Show tasks with hierarchical paths
	fmt.Printf("\n%d tasks found matching '%s':\n", len(tasks), searchSummary)
	dateFormat := cfg.GetDateFormat()

	for i, task := range tasks {
		path := GetTaskPath(&task, taskMap)
		fmt.Printf("\n%d: [%s]", i+1, path)
		fmt.Print(task.FormatWithView("all", taskManager, dateFormat))
	}

	fmt.Printf("\nSelect task (1-%d) or 0 to cancel: ", len(tasks))
	var choice int
	if _, err := fmt.Scanf("%d", &choice); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if choice == 0 {
		return nil, fmt.Errorf("operation cancelled")
	}

	if choice < 1 || choice > len(tasks) {
		return nil, fmt.Errorf("invalid choice: %d", choice)
	}

	return &tasks[choice-1], nil
}

// TaskNode represents a node in the task tree
type TaskNode struct {
	Task     *backend.Task
	Children []*TaskNode
}

// BuildTaskTree builds a hierarchical tree from a flat list of tasks
func BuildTaskTree(tasks []backend.Task) []*TaskNode {
	// Build map of UID to task pointer
	taskMap := make(map[string]*backend.Task)
	for i := range tasks {
		taskMap[tasks[i].UID] = &tasks[i]
	}

	// Build map of parent UID to children
	childrenMap := make(map[string][]*backend.Task)
	var rootTasks []*backend.Task

	for i := range tasks {
		task := &tasks[i]
		if task.ParentUID == "" {
			rootTasks = append(rootTasks, task)
		} else {
			childrenMap[task.ParentUID] = append(childrenMap[task.ParentUID], task)
		}
	}

	// Recursively build tree
	var buildNode func(*backend.Task) *TaskNode
	buildNode = func(task *backend.Task) *TaskNode {
		node := &TaskNode{
			Task:     task,
			Children: []*TaskNode{},
		}

		// Add children recursively
		if children, exists := childrenMap[task.UID]; exists {
			for _, child := range children {
				node.Children = append(node.Children, buildNode(child))
			}
		}

		return node
	}

	// Build root nodes
	var roots []*TaskNode
	for _, rootTask := range rootTasks {
		roots = append(roots, buildNode(rootTask))
	}

	return roots
}

// FormatTaskTree formats a task tree with box-drawing characters for hierarchical display
func FormatTaskTree(nodes []*TaskNode, view string, taskManager backend.TaskManager, dateFormat string) string {
	var result strings.Builder
	formatNode(&result, nodes, "", true, view, taskManager, dateFormat)
	return result.String()
}

// formatNode recursively formats a task node with proper indentation
func formatNode(result *strings.Builder, nodes []*TaskNode, prefix string, isRoot bool, view string, taskManager backend.TaskManager, dateFormat string) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1

		// Determine the tree characters
		var nodePrefix, childPrefix string
		if isRoot {
			nodePrefix = ""
			childPrefix = ""
		} else {
			if isLast {
				nodePrefix = prefix + "└─ "
				childPrefix = prefix + "   "
			} else {
				nodePrefix = prefix + "├─ "
				childPrefix = prefix + "│  "
			}
		}

		// Format the task
		taskOutput := node.Task.FormatWithView(view, taskManager, dateFormat)

		// Add indentation to each line of the task output
		if nodePrefix != "" {
			lines := strings.Split(strings.TrimRight(taskOutput, "\n"), "\n")
			for j, line := range lines {
				if j == 0 {
					result.WriteString(nodePrefix)
				} else {
					// Continuation lines use the child prefix
					if isLast {
						result.WriteString(prefix + "   ")
					} else {
						result.WriteString(prefix + "│  ")
					}
				}
				result.WriteString(line)
				result.WriteString("\n")
			}
		} else {
			result.WriteString(taskOutput)
		}

		// Recursively format children
		if len(node.Children) > 0 {
			formatNode(result, node.Children, childPrefix, false, view, taskManager, dateFormat)
		}
	}
}
