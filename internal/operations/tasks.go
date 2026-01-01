package operations

import (
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/internal/config"
	"gosynctasks/internal/utils"
	"strings"

	"github.com/spf13/cobra"
)

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
	fmt.Println()
	return utils.PromptConfirmation("Proceed with this task?")
}

// buildFlatTaskList recursively builds a flat list of tasks from the tree
// This is useful for numbered selection where we need sequential access
func buildFlatTaskList(nodes []*TaskNode, flatTasks *[]*backend.Task) {
	for _, node := range nodes {
		*flatTasks = append(*flatTasks, node.Task)
		if len(node.Children) > 0 {
			buildFlatTaskList(node.Children, flatTasks)
		}
	}
}

// formatTaskTreeNumbered recursively formats tasks with numbering and hierarchy
// Returns a string representation instead of printing directly (for testability)
func formatTaskTreeNumbered(nodes []*TaskNode, taskManager backend.TaskManager, dateFormat string, startNum int, prefix string, isRoot bool) (string, int) {
	var result strings.Builder
	currentNum := startNum

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

		// Format task number with color
		numColor := "\033[36m" // Cyan
		reset := "\033[0m"

		// Display task with number and hierarchy
		taskOutput := node.Task.FormatWithView("default", taskManager, dateFormat)
		lines := strings.Split(taskOutput, "\n")

		// First line with number and tree prefix
		if len(lines) > 0 && lines[0] != "" {
			result.WriteString(fmt.Sprintf("  %s%2d.%s %s%s\n", numColor, currentNum, reset, nodePrefix, lines[0]))
			currentNum++

			// Additional lines maintain indentation
			for j := 1; j < len(lines); j++ {
				if lines[j] != "" {
					indent := "     " // Space for number
					if !isRoot {
						indent += childPrefix
					}
					result.WriteString(fmt.Sprintf("%s%s\n", indent, lines[j]))
				}
			}
		}

		// Recursively format children
		if len(node.Children) > 0 {
			childOutput, newNum := formatTaskTreeNumbered(node.Children, taskManager, dateFormat, currentNum, childPrefix, false)
			result.WriteString(childOutput)
			currentNum = newNum
		}
	}

	return result.String(), currentNum
}

// displayTaskTreeNumbered recursively displays tasks with numbering and hierarchy
// This is the original function maintained for backward compatibility
func displayTaskTreeNumbered(nodes []*TaskNode, taskManager backend.TaskManager, dateFormat string, flatTasks *[]*backend.Task, prefix string, isRoot bool) {
	// Build flat list
	buildFlatTaskList(nodes, flatTasks)

	// Format and print the tree
	output, _ := formatTaskTreeNumbered(nodes, taskManager, dateFormat, 1, prefix, isRoot)
	fmt.Print(output)
}

// BuildFilter constructs a TaskFilter from cobra command flags
// Uses the backend's ParseStatusFlag to convert user input to backend-specific format
func BuildFilter(cmd *cobra.Command, taskManager backend.TaskManager) (*backend.TaskFilter, error) {
	filter := &backend.TaskFilter{}

	// Get status flags (errors ignored as flags are always defined by the command)
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
