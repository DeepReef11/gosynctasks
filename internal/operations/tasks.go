package operations

import (
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/internal/config"
	"strings"

	"github.com/spf13/cobra"
)

// FindTaskBySummary searches for a task by summary and handles UX for exact/partial/multiple matches
func FindTaskBySummary(taskManager backend.TaskManager, cfg *config.Config, listID string, searchSummary string) (*backend.Task, error) {
	// Use backend's search method
	matches, err := taskManager.FindTasksBySummary(listID, searchSummary)
	if err != nil {
		return nil, fmt.Errorf("error searching for tasks: %w", err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no tasks found matching '%s'", searchSummary)
	}

	// Separate exact and partial matches
	var exactMatches []backend.Task
	var partialMatches []backend.Task
	searchLower := strings.ToLower(searchSummary)

	for _, task := range matches {
		if strings.ToLower(task.Summary) == searchLower {
			exactMatches = append(exactMatches, task)
		} else {
			partialMatches = append(partialMatches, task)
		}
	}

	// Single exact match - proceed without confirmation
	if len(exactMatches) == 1 && len(partialMatches) == 0 {
		return &exactMatches[0], nil
	}

	// Single partial match - ask for confirmation
	if len(exactMatches) == 0 && len(partialMatches) == 1 {
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

	// Multiple matches (exact or partial) - prompt selection with hierarchical paths
	if len(exactMatches) > 1 {
		return selectTask(exactMatches, searchSummary, taskManager, cfg, listID)
	}

	// Mix of exact and partial, or multiple partial
	return selectTask(matches, searchSummary, taskManager, cfg, listID)
}

// selectTask shows a list of tasks and prompts user to select one
// Now includes hierarchical paths for disambiguation
func selectTask(tasks []backend.Task, searchSummary string, taskManager backend.TaskManager, cfg *config.Config, listID string) (*backend.Task, error) {
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks found matching '%s'", searchSummary)
	}

	// Get all tasks to build hierarchy
	allTasks, err := taskManager.GetTasks(listID, nil)
	if err != nil {
		// Fall back to simple display if we can't get all tasks
		return selectTaskSimple(tasks, searchSummary, taskManager, cfg)
	}

	// Build UID to task map for path resolution
	taskMap := make(map[string]*backend.Task)
	for i := range allTasks {
		taskMap[allTasks[i].UID] = &allTasks[i]
	}

	// Show tasks with hierarchical paths and "all" view
	fmt.Printf("\n%d tasks found matching '%s':\n", len(tasks), searchSummary)
	dateFormat := cfg.GetDateFormat()

	for i, task := range tasks {
		// Get hierarchical path
		path := GetTaskPath(&task, taskMap)
		fmt.Printf("\n%d: [%s]", i+1, path)
		fmt.Print(task.FormatWithView("all", taskManager, dateFormat))
	}

	fmt.Printf("\nSelect task (1-%d) or 0 to skip: ", len(tasks))
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

// selectTaskSimple is a fallback that doesn't show hierarchical paths
func selectTaskSimple(tasks []backend.Task, searchSummary string, taskManager backend.TaskManager, cfg *config.Config) (*backend.Task, error) {
	fmt.Printf("\n%d tasks found matching '%s':\n", len(tasks), searchSummary)
	dateFormat := cfg.GetDateFormat()

	for i, task := range tasks {
		fmt.Printf("\n%d:", i+1)
		fmt.Print(task.FormatWithView("all", taskManager, dateFormat))
	}

	fmt.Printf("\nSelect task (1-%d) or 0 to skip: ", len(tasks))
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

// GetTaskPath returns the hierarchical path of a task (e.g., "Feature X / Write code / Fix bug")
// This is exported so it can be used by other packages
func GetTaskPath(task *backend.Task, taskMap map[string]*backend.Task) string {
	var parts []string
	current := task

	// Walk up the parent chain
	for current != nil {
		parts = append([]string{current.Summary}, parts...)
		if current.ParentUID == "" {
			break
		}
		current = taskMap[current.ParentUID]
	}

	return strings.Join(parts, " / ")
}

// confirmTask shows task details and asks for confirmation
func confirmTask(task *backend.Task, taskManager backend.TaskManager, cfg *config.Config) (bool, error) {
	dateFormat := cfg.GetDateFormat()
	fmt.Println("\nTask found:")
	fmt.Print(task.FormatWithView("all", taskManager, dateFormat))
	fmt.Print("\nProceed with this task? (y/n): ")

	var response string
	if _, err := fmt.Scanf("%s", &response); err != nil {
		return false, fmt.Errorf("invalid input: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}

// BuildFilter constructs a TaskFilter from cobra command flags
// Uses the backend's ParseStatusFlag to convert user input to backend-specific format
func BuildFilter(cmd *cobra.Command, taskManager backend.TaskManager) (*backend.TaskFilter, error) {
	filter := &backend.TaskFilter{}

	statuses, _ := cmd.Flags().GetStringArray("status")
	if len(statuses) > 0 {
		var allStatuses []string
		for _, status := range statuses {
			// Split by comma and trim spaces
			parts := strings.SplitSeq(status, ",")
			for part := range parts {
				allStatuses = append(allStatuses, strings.TrimSpace(part))
			}
		}

		// Parse each status using backend's parser
		var parsedStatuses []string
		for _, status := range allStatuses {
			parsed, err := taskManager.ParseStatusFlag(status)
			if err != nil {
				return nil, fmt.Errorf("invalid status '%s': %w", status, err)
			}
			parsedStatuses = append(parsedStatuses, parsed)
		}
		filter.Statuses = &parsedStatuses
	}

	return filter, nil
}
