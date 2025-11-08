package operations

import (
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/internal/cli"
	"gosynctasks/internal/config"
	"strings"

	"github.com/spf13/cobra"
)

// ExecuteAction parses arguments and routes to the appropriate action handler
func ExecuteAction(taskManager backend.TaskManager, cfg *config.Config, taskLists []backend.TaskList, cmd *cobra.Command, args []string) error {
	var listName string
	var taskSummary string
	var searchSummary string
	action := "get"

	// Argument order: <list> [action] [task-summary]
	if len(args) >= 1 {
		listName = args[0]
	}
	if len(args) >= 2 {
		action = args[1]
	}
	if len(args) >= 3 {
		// For update/complete: arg[2] is summary to search for
		// For add: arg[2] is task summary to create
		if strings.ToLower(action) == "update" || strings.ToLower(action) == "u" ||
			strings.ToLower(action) == "complete" || strings.ToLower(action) == "c" {
			searchSummary = args[2]
		} else {
			taskSummary = args[2]
		}
	}

	// Normalize action (support abbreviations)
	action = NormalizeAction(action)

	filter := BuildFilter(cmd)

	selectedList, err := GetSelectedList(taskLists, taskManager, listName)
	if err != nil {
		return err
	}

	switch action {
	case "get":
		return HandleGetAction(cmd, taskManager, cfg, selectedList, filter)

	case "add":
		return HandleAddAction(cmd, taskManager, selectedList, taskSummary)

	case "update":
		return HandleUpdateAction(cmd, taskManager, cfg, selectedList, searchSummary)

	case "complete":
		return HandleCompleteAction(cmd, taskManager, cfg, selectedList, searchSummary)

	default:
		return fmt.Errorf("unknown action: %s (supported: get/g, add/a, update/u, complete/c)", action)
	}
}

// NormalizeAction converts action abbreviations to full action names
func NormalizeAction(action string) string {
	action = strings.ToLower(action)
	switch action {
	case "g":
		return "get"
	case "a":
		return "add"
	case "u":
		return "update"
	case "c":
		return "complete"
	default:
		return action
	}
}

// HandleGetAction lists tasks from a task list
func HandleGetAction(cmd *cobra.Command, taskManager backend.TaskManager, cfg *config.Config, selectedList *backend.TaskList, filter *backend.TaskFilter) error {
	tasks, err := taskManager.GetTasks(selectedList.ID, filter)
	if err != nil {
		return fmt.Errorf("error retrieving tasks: %w", err)
	}

	// Sort using backend-specific sorting
	taskManager.SortTasks(tasks)

	view, _ := cmd.Flags().GetString("view")
	dateFormat := cfg.GetDateFormat()
	termWidth := cli.GetTerminalWidth()

	fmt.Print(selectedList.StringWithWidth(termWidth))
	for _, task := range tasks {
		fmt.Print(task.FormatWithView(view, taskManager, dateFormat))
	}
	fmt.Print(selectedList.BottomBorderWithWidth(termWidth))
	return nil
}

// HandleAddAction adds a new task to a list
func HandleAddAction(cmd *cobra.Command, taskManager backend.TaskManager, selectedList *backend.TaskList, taskSummary string) error {
	// If no task summary provided in args, prompt for it
	if taskSummary == "" {
		fmt.Print("Enter task summary: ")
		var input string
		if _, err := fmt.Scanln(&input); err != nil {
			return fmt.Errorf("failed to read task summary: %w", err)
		}
		taskSummary = input
	}

	if taskSummary == "" {
		return fmt.Errorf("task summary cannot be empty")
	}

	// Get optional flags
	description, _ := cmd.Flags().GetString("description")
	priority, _ := cmd.Flags().GetInt("priority")
	statusFlag, _ := cmd.Flags().GetString("add-status")

	// Default status
	taskStatus := "NEEDS-ACTION"

	// Handle status flag (case insensitive, translate app statuses)
	if statusFlag != "" {
		upperStatus := strings.ToUpper(statusFlag)
		// Handle app status names and abbreviations
		switch upperStatus {
		case "T", "TODO":
			taskStatus = "NEEDS-ACTION"
		case "D", "DONE":
			taskStatus = "COMPLETED"
		case "P", "PROCESSING":
			taskStatus = "IN-PROCESS"
		case "C", "CANCELLED":
			taskStatus = "CANCELLED"
		// Also accept backend status names directly
		case "NEEDS-ACTION", "COMPLETED", "IN-PROCESS":
			taskStatus = upperStatus
		default:
			return fmt.Errorf("invalid status: %s (valid: TODO/T, DONE/D, PROCESSING/P, CANCELLED/C, or backend statuses)", statusFlag)
		}
	}

	// Validate priority
	if priority < 0 || priority > 9 {
		return fmt.Errorf("priority must be between 0-9 (0=undefined, 1=highest, 9=lowest)")
	}

	task := backend.Task{
		Summary:     taskSummary,
		Description: description,
		Status:      taskStatus,
		Priority:    priority,
	}

	if err := taskManager.AddTask(selectedList.ID, task); err != nil {
		return fmt.Errorf("error adding task: %w", err)
	}

	fmt.Printf("Task '%s' added successfully to list '%s'\n", taskSummary, selectedList.Name)
	return nil
}

// HandleUpdateAction updates an existing task
func HandleUpdateAction(cmd *cobra.Command, taskManager backend.TaskManager, cfg *config.Config, selectedList *backend.TaskList, searchSummary string) error {
	// Get the task summary to search for
	if searchSummary == "" {
		return fmt.Errorf("task summary is required for update (usage: gosynctasks <list> update <task-summary>)")
	}

	// Find the task by summary (handles exact/partial/multiple matches)
	taskToUpdate, err := FindTaskBySummary(taskManager, cfg, selectedList.ID, searchSummary)
	if err != nil {
		return err
	}

	// Get update flags
	statusFlags, _ := cmd.Flags().GetStringArray("status")
	description, _ := cmd.Flags().GetString("description")
	priority, _ := cmd.Flags().GetInt("priority")
	summaryFlag, _ := cmd.Flags().GetString("summary")

	// Update fields if provided
	// For update action, use first status value if provided
	if len(statusFlags) > 0 && statusFlags[0] != "" {
		statusFlag := statusFlags[0]
		upperStatus := strings.ToUpper(statusFlag)
		switch upperStatus {
		case "T", "TODO":
			taskToUpdate.Status = "NEEDS-ACTION"
		case "D", "DONE":
			taskToUpdate.Status = "COMPLETED"
		case "P", "PROCESSING":
			taskToUpdate.Status = "IN-PROCESS"
		case "C", "CANCELLED":
			taskToUpdate.Status = "CANCELLED"
		case "NEEDS-ACTION", "COMPLETED", "IN-PROCESS":
			taskToUpdate.Status = upperStatus
		default:
			return fmt.Errorf("invalid status: %s (valid: TODO/T, DONE/D, PROCESSING/P, CANCELLED/C)", statusFlag)
		}
	}

	if summaryFlag != "" {
		taskToUpdate.Summary = summaryFlag
	}

	if cmd.Flags().Changed("description") {
		taskToUpdate.Description = description
	}

	if cmd.Flags().Changed("priority") {
		if priority < 0 || priority > 9 {
			return fmt.Errorf("priority must be between 0-9 (0=undefined, 1=highest, 9=lowest)")
		}
		taskToUpdate.Priority = priority
	}

	// Update the task
	if err := taskManager.UpdateTask(selectedList.ID, *taskToUpdate); err != nil {
		return fmt.Errorf("error updating task: %w", err)
	}

	fmt.Printf("Task '%s' updated successfully in list '%s'\n", taskToUpdate.Summary, selectedList.Name)
	return nil
}

// HandleCompleteAction marks a task with a status (defaults to COMPLETED)
func HandleCompleteAction(cmd *cobra.Command, taskManager backend.TaskManager, cfg *config.Config, selectedList *backend.TaskList, searchSummary string) error {
	// Get the task summary to search for
	if searchSummary == "" {
		return fmt.Errorf("task summary is required for complete (usage: gosynctasks <list> complete <task-summary>)")
	}

	// Find the task by summary (handles exact/partial/multiple matches)
	taskToComplete, err := FindTaskBySummary(taskManager, cfg, selectedList.ID, searchSummary)
	if err != nil {
		return err
	}

	// Get status flag (if provided), otherwise default to COMPLETED
	statusFlags, _ := cmd.Flags().GetStringArray("status")
	newStatus := "COMPLETED" // Default
	statusName := "completed"

	if len(statusFlags) > 0 && statusFlags[0] != "" {
		statusFlag := statusFlags[0]
		upperStatus := strings.ToUpper(statusFlag)
		switch upperStatus {
		case "T", "TODO":
			newStatus = "NEEDS-ACTION"
			statusName = "TODO"
		case "D", "DONE":
			newStatus = "COMPLETED"
			statusName = "DONE"
		case "P", "PROCESSING":
			newStatus = "IN-PROCESS"
			statusName = "PROCESSING"
		case "C", "CANCELLED":
			newStatus = "CANCELLED"
			statusName = "CANCELLED"
		case "NEEDS-ACTION", "COMPLETED", "IN-PROCESS":
			newStatus = upperStatus
			statusName = upperStatus
		default:
			return fmt.Errorf("invalid status: %s (valid: TODO/T, DONE/D, PROCESSING/P, CANCELLED/C)", statusFlag)
		}
	}

	// Set the new status
	taskToComplete.Status = newStatus

	// Update the task
	if err := taskManager.UpdateTask(selectedList.ID, *taskToComplete); err != nil {
		return fmt.Errorf("error updating task: %w", err)
	}

	fmt.Printf("Task '%s' marked as %s in list '%s'\n", taskToComplete.Summary, statusName, selectedList.Name)
	return nil
}
