package operations

import (
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/internal/cli"
	"strings"
)

// FindListByName performs case-insensitive search for a task list by name
func FindListByName(taskLists []backend.TaskList, name string) *backend.TaskList {
	for _, list := range taskLists {
		if strings.EqualFold(list.Name, name) {
			return &list
		}
	}
	return nil
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
		selectedList := FindListByName(taskLists, listName)
		if selectedList == nil {
			return nil, fmt.Errorf("list '%s' not found", listName)
		}
		return selectedList, nil
	}

	// No list name provided, use interactive selection
	return SelectListInteractively(taskLists, taskManager)
}
