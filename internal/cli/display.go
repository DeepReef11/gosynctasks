package cli

import (
	"fmt"
	"gosynctasks/backend"
	"os"
	"strings"

	"golang.org/x/term"
)

// GetTerminalWidth returns the current terminal width, defaulting to 80 if unable to detect
func GetTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Default to 80 if we can't detect terminal size
		return 80
	}
	return width
}

// ShowTaskLists displays a formatted list of task lists with borders, colors, and task counts
func ShowTaskLists(taskLists []backend.TaskList, taskManager backend.TaskManager) {
	termWidth := GetTerminalWidth()

	// Calculate border width (leave some padding)
	borderWidth := termWidth - 2
	if borderWidth < 40 {
		borderWidth = 40 // Minimum width
	}
	if borderWidth > 100 {
		borderWidth = 100 // Maximum width for readability
	}

	// Header - fixed to match footer width
	headerText := "─ Available Task Lists "
	headerPadding := borderWidth - len(headerText)
	if headerPadding < 0 {
		headerPadding = 0
	}
	fmt.Printf("\n\033[1;36m┌%s%s┐\033[0m\n", headerText, strings.Repeat("─", headerPadding))

	// List each task list with formatting
	for i, list := range taskLists {
		// Get task count for this list
		tasks, err := taskManager.GetTasks(list.ID, nil)
		taskCount := 0
		if err == nil {
			taskCount = len(tasks)
		}

		// Format the line
		nameColor := "\033[1;37m" // Bold white
		numColor := "\033[36m"     // Cyan
		countColor := "\033[90m"   // Gray
		reset := "\033[0m"

		fmt.Printf("  %s%2d.%s %s%-30s%s", numColor, i+1, reset, nameColor, list.Name, reset)

		// Show task count
		if taskCount > 0 {
			fmt.Printf(" %s(%d task", countColor, taskCount)
			if taskCount != 1 {
				fmt.Printf("s")
			}
			fmt.Printf(")%s", reset)
		}

		// Show description if available
		if list.Description != "" {
			fmt.Printf("\n      %s%s%s", countColor, list.Description, reset)
		}
		fmt.Println()
	}

	// Footer
	fmt.Printf("\033[1;36m└%s┘\033[0m\n", strings.Repeat("─", borderWidth))
}
