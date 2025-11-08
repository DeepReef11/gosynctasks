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

	// Multiple matches (exact or partial) - prompt selection
	if len(exactMatches) > 1 {
		return selectTask(exactMatches, searchSummary, taskManager, cfg)
	}

	// Mix of exact and partial, or multiple partial
	return selectTask(matches, searchSummary, taskManager, cfg)
}

// selectTask shows a list of tasks and prompts user to select one
func selectTask(tasks []backend.Task, searchSummary string, taskManager backend.TaskManager, cfg *config.Config) (*backend.Task, error) {
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks found matching '%s'", searchSummary)
	}

	// Show tasks with "all" view
	fmt.Printf("\n%d tasks found matching '%s':\n", len(tasks), searchSummary)
	dateFormat := cfg.GetDateFormat()

	for i, task := range tasks {
		fmt.Printf("\n%d:", i+1)
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
func BuildFilter(cmd *cobra.Command) *backend.TaskFilter {
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

		// Convert to full status names and uppercase
		var upperStatuses []string
		for _, status := range allStatuses {
			upperStatus := strings.ToUpper(status)
			// Handle abbreviations
			switch upperStatus {
			case "P":
				upperStatus = "PROCESSING"
			case "D":
				upperStatus = "DONE"
			case "T":
				upperStatus = "TODO"
			case "C":
				upperStatus = "CANCELLED"
			}
			upperStatuses = append(upperStatuses, upperStatus)
		}
		filter.Statuses = &upperStatuses
	}

	return filter
}
