package operations

import (
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/internal/cli"
	"gosynctasks/internal/config"
	"gosynctasks/internal/utils"
	"gosynctasks/internal/views"
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
		// For update/complete/delete: arg[2] is summary to search for
		// For add: arg[2] is task summary to create
		if strings.ToLower(action) == "update" || strings.ToLower(action) == "u" ||
			strings.ToLower(action) == "complete" || strings.ToLower(action) == "c" ||
			strings.ToLower(action) == "delete" || strings.ToLower(action) == "d" {
			searchSummary = args[2]
		} else {
			taskSummary = args[2]
		}
	}

	// Normalize action (support abbreviations)
	action = NormalizeAction(action)

	selectedList, err := GetSelectedList(taskLists, taskManager, listName)
	if err != nil {
		return err
	}

	filter, err := BuildFilter(cmd, taskManager)
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

	case "delete":
		return HandleDeleteAction(cmd, taskManager, cfg, selectedList, searchSummary)

	default:
		return fmt.Errorf("unknown action: %s (supported: get/g, add/a, update/u, complete/c, delete/d)", action)
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
	case "d":
		return "delete"
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

	viewName, _ := cmd.Flags().GetString("view")
	dateFormat := cfg.GetDateFormat()
	termWidth := cli.GetTerminalWidth()

	// Try to use custom view rendering first
	// Note: Custom views currently don't support hierarchical display
	// This will be added in a future enhancement
	rendered, err := RenderWithCustomView(tasks, viewName, taskManager, dateFormat)
	if err == nil {
		// Custom view found and rendered successfully
		fmt.Print(selectedList.StringWithWidth(termWidth))
		fmt.Print(rendered)
		fmt.Print(selectedList.BottomBorderWithWidth(termWidth))
		return nil
	}

	// Fall back to tree-based hierarchical display
	fmt.Print(selectedList.StringWithWidth(termWidth))

	// Build task tree
	tree := BuildTaskTree(tasks)

	// Format and display tree
	treeOutput := FormatTaskTree(tree, viewName, taskManager, dateFormat)
	fmt.Print(treeOutput)

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
	dueDateStr, _ := cmd.Flags().GetString("due-date")
	startDateStr, _ := cmd.Flags().GetString("start-date")
	parentRef, _ := cmd.Flags().GetString("parent")
	literal, _ := cmd.Flags().GetBool("literal")

	// Default status: use backend's parser with "TODO" as default
	var taskStatus string
	var err error
	if statusFlag != "" {
		taskStatus, err = taskManager.ParseStatusFlag(statusFlag)
	} else {
		taskStatus, err = taskManager.ParseStatusFlag("TODO")
	}
	if err != nil {
		return err
	}

	// Validate priority
	if err := utils.ValidatePriority(priority); err != nil {
		return err
	}

	// Parse and validate dates
	dueDate, err := utils.ParseDateFlag(dueDateStr)
	if err != nil {
		return err
	}

	startDate, err := utils.ParseDateFlag(startDateStr)
	if err != nil {
		return err
	}

	if err := utils.ValidateDates(startDate, dueDate); err != nil {
		return err
	}

	cfg := config.GetConfig()
	var parentUID string
	var actualTaskName string

	// Handle path-based task creation or parent resolution
	if parentRef != "" {
		// Explicit parent provided via -P flag
		parentUID, err = ResolveParentTask(taskManager, cfg, selectedList.ID, parentRef, taskStatus)
		if err != nil {
			return fmt.Errorf("failed to resolve parent task: %w", err)
		}
		actualTaskName = taskSummary
	} else if !literal && strings.Contains(taskSummary, "/") {
		// Path-based shorthand: "parent/child/task" creates hierarchy automatically
		// Skip if --literal flag is set
		fmt.Printf("Detected path-based task creation: '%s'\n", taskSummary)
		parentUID, actualTaskName, err = CreateOrFindTaskPath(taskManager, cfg, selectedList.ID, taskSummary, taskStatus)
		if err != nil {
			return fmt.Errorf("failed to create task path: %w", err)
		}
	} else {
		// Simple task with no parent (or literal mode)
		actualTaskName = taskSummary
	}

	task := backend.Task{
		Summary:     actualTaskName,
		Description: description,
		Status:      taskStatus,
		Priority:    priority,
		DueDate:     dueDate,
		StartDate:   startDate,
		ParentUID:   parentUID,
	}

	if err := taskManager.AddTask(selectedList.ID, task); err != nil {
		return fmt.Errorf("error adding task: %w", err)
	}

	fmt.Printf("Task '%s' added successfully to list '%s'\n", actualTaskName, selectedList.Name)
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
	dueDateStr, _ := cmd.Flags().GetString("due-date")
	startDateStr, _ := cmd.Flags().GetString("start-date")

	// Update fields if provided
	// For update action, use first status value if provided
	if len(statusFlags) > 0 && statusFlags[0] != "" {
		newStatus, err := taskManager.ParseStatusFlag(statusFlags[0])
		if err != nil {
			return err
		}
		taskToUpdate.Status = newStatus
	}

	if summaryFlag != "" {
		taskToUpdate.Summary = summaryFlag
	}

	if cmd.Flags().Changed("description") {
		taskToUpdate.Description = description
	}

	if cmd.Flags().Changed("priority") {
		if err := utils.ValidatePriority(priority); err != nil {
			return err
		}
		taskToUpdate.Priority = priority
	}

	// Parse and update dates if changed
	if cmd.Flags().Changed("due-date") {
		dueDate, err := utils.ParseDateFlag(dueDateStr)
		if err != nil {
			return err
		}
		taskToUpdate.DueDate = dueDate
	}

	if cmd.Flags().Changed("start-date") {
		startDate, err := utils.ParseDateFlag(startDateStr)
		if err != nil {
			return err
		}
		taskToUpdate.StartDate = startDate
	}

	// Validate dates (after all updates applied)
	if err := utils.ValidateDates(taskToUpdate.StartDate, taskToUpdate.DueDate); err != nil {
		return err
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

	// Get status flag (if provided), otherwise default to DONE
	statusFlags, _ := cmd.Flags().GetStringArray("status")
	var newStatus string

	if len(statusFlags) > 0 && statusFlags[0] != "" {
		newStatus, err = taskManager.ParseStatusFlag(statusFlags[0])
	} else {
		newStatus, err = taskManager.ParseStatusFlag("DONE")
	}
	if err != nil {
		return err
	}

	// Get display name for user feedback
	statusName := taskManager.StatusToDisplayName(newStatus)

	// Set the new status
	taskToComplete.Status = newStatus

	// Update the task
	if err := taskManager.UpdateTask(selectedList.ID, *taskToComplete); err != nil {
		return fmt.Errorf("error updating task: %w", err)
	}

	fmt.Printf("Task '%s' marked as %s in list '%s'\n", taskToComplete.Summary, statusName, selectedList.Name)
	return nil
}

// HandleDeleteAction deletes a task by summary
func HandleDeleteAction(cmd *cobra.Command, taskManager backend.TaskManager, cfg *config.Config, selectedList *backend.TaskList, searchSummary string) error {
	// Get the task summary to search for
	if searchSummary == "" {
		return fmt.Errorf("task summary is required for delete (usage: gosynctasks <list> delete <task-summary>)")
	}

	// Find the task by summary (handles exact/partial/multiple matches)
	taskToDelete, err := FindTaskBySummary(taskManager, cfg, selectedList.ID, searchSummary)
	if err != nil {
		return err
	}

	// Show a final confirmation before deletion
	fmt.Printf("\nAre you sure you want to delete task '%s'? This action cannot be undone. (y/n): ", taskToDelete.Summary)
	var response string
	if _, err := fmt.Scanf("%s", &response); err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" && response != "yes" {
		return fmt.Errorf("deletion cancelled")
	}

	// Delete the task
	if err := taskManager.DeleteTask(selectedList.ID, taskToDelete.UID); err != nil {
		return fmt.Errorf("error deleting task: %w", err)
	}

	fmt.Printf("Task '%s' deleted successfully from list '%s'\n", taskToDelete.Summary, selectedList.Name)
	return nil
}

// RenderWithCustomView attempts to render tasks using a custom view
// Returns the rendered output or an error if the view cannot be loaded
// This version supports hierarchical display with tree structure
func RenderWithCustomView(tasks []backend.Task, viewName string, taskManager backend.TaskManager, dateFormat string) (string, error) {
	// Don't use custom rendering for built-in legacy views
	if viewName == "basic" || viewName == "all" {
		return "", fmt.Errorf("using legacy view")
	}

	// Try to resolve the view
	view, err := views.ResolveView(viewName)
	if err != nil {
		return "", err
	}

	// Create renderer
	renderer := views.NewViewRenderer(view, taskManager, dateFormat)

	// Apply view-specific filters
	filteredTasks := tasks
	if filters := renderer.GetFilters(); filters != nil {
		filteredTasks = views.ApplyFilters(tasks, filters)
	}

	// Build task tree BEFORE sorting
	// This preserves parent-child relationships
	tree := BuildTaskTree(filteredTasks)

	// Apply view-specific sorting hierarchically
	// This sorts root tasks and recursively sorts children within each parent
	sortBy, sortOrder := renderer.GetSortConfig()
	if sortBy != "" {
		SortTaskTree(tree, sortBy, sortOrder)
	}

	// Render tasks with hierarchy
	return RenderTaskTreeWithCustomView(tree, renderer), nil
}

// RenderTaskTreeWithCustomView formats a task tree using a custom view renderer
func RenderTaskTreeWithCustomView(nodes []*TaskNode, renderer *views.ViewRenderer) string {
	var result strings.Builder
	formatNodeWithCustomView(&result, nodes, "", true, renderer)
	return result.String()
}

// formatNodeWithCustomView recursively formats a task node with proper indentation using custom view
func formatNodeWithCustomView(result *strings.Builder, nodes []*TaskNode, prefix string, isRoot bool, renderer *views.ViewRenderer) {
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

		// Render the task with hierarchy using the custom view
		taskOutput := renderer.RenderTaskHierarchical(*node.Task, nodePrefix, childPrefix)
		result.WriteString(taskOutput)

		// Recursively format children
		if len(node.Children) > 0 {
			formatNodeWithCustomView(result, node.Children, childPrefix, false, renderer)
		}
	}
}
