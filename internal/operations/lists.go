package operations

import (
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/internal/cli"
	"strings"
)

// FindListByName searches for a task list by name and returns its ID.
// Performs case-insensitive search. Returns an error if the list is not found.
func FindListByName(taskLists []backend.TaskList, name string) (string, error) {
	for _, list := range taskLists {
		if strings.EqualFold(list.Name, name) {
			return list.ID, nil
		}
	}
	return "", fmt.Errorf("list '%s' not found", name)
}

// FindListByNameFull searches for a task list by name and returns the complete TaskList struct.
// Performs case-insensitive search. Returns an error if the list is not found.
func FindListByNameFull(taskLists []backend.TaskList, name string) (*backend.TaskList, error) {
	for _, list := range taskLists {
		if strings.EqualFold(list.Name, name) {
			return &list, nil
		}
	}
	return nil, fmt.Errorf("list '%s' not found", name)
}

// SelectListInteractively displays task lists and prompts user to select one
func SelectListInteractively(taskLists []backend.TaskList, taskManager backend.TaskManager) (*backend.TaskList, error) {
	cli.ShowTaskLists(taskLists, taskManager)

	fmt.Printf("\n\033[1mSelect list (1-%d, or 0 to cancel):\033[0m ", len(taskLists))
	var choice int
	if _, err := fmt.Scanf("%d", &choice); err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	if choice == 0 {
		return nil, fmt.Errorf("cancelled")
	}

	if choice < 1 || choice > len(taskLists) {
		return nil, fmt.Errorf("invalid choice: %d (must be 1-%d)", choice, len(taskLists))
	}

	return &taskLists[choice-1], nil
}

// GetSelectedList returns a list by name or prompts for interactive selection
func GetSelectedList(taskLists []backend.TaskList, taskManager backend.TaskManager, listName string) (*backend.TaskList, error) {
	if listName != "" {
		selectedList, err := FindListByNameFull(taskLists, listName)
		if err != nil {
			// If no task lists were loaded at all, suggest checking connection
			if len(taskLists) == 0 {
				return nil, fmt.Errorf("list '%s' not found - no task lists could be loaded. This usually means a connection or authentication failure. Please check your connection URL, username, and password in the config file", listName)
			}
			return nil, fmt.Errorf("list '%s' not found. Available lists: %s", listName, formatAvailableLists(taskLists))
		}
		return selectedList, nil
	}

	// No list name provided, use interactive selection
	if len(taskLists) == 0 {
		return nil, fmt.Errorf("no task lists available - failed to connect to backend. Please check your connection URL, username, and password in the config file")
	}
	return SelectListInteractively(taskLists, taskManager)
}

// formatAvailableLists creates a comma-separated list of available task list names
func formatAvailableLists(taskLists []backend.TaskList) string {
	if len(taskLists) == 0 {
		return "(none)"
	}
	names := make([]string, len(taskLists))
	for i, list := range taskLists {
		names[i] = list.Name
	}
	return strings.Join(names, ", ")
}
