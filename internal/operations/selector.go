package operations

import (
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/internal/cli"
	"gosynctasks/internal/config"
	"gosynctasks/internal/utils"
	"strings"
)

// TaskSelector provides a unified interface for task selection across the application.
// It consolidates the logic from FindTaskBySummary, selectTask, selectTaskSimple,
// SelectTaskInteractively, and selectTaskWithPath into a single configurable service.
type TaskSelector struct {
	taskManager backend.TaskManager
	config      *config.Config
}

// SelectionOptions configures how task selection should behave.
type SelectionOptions struct {
	// ShowHierarchy determines whether to display task hierarchical paths
	ShowHierarchy bool

	// DisplayFormat controls the display style: "list" or "tree"
	DisplayFormat string

	// CancelText is the text shown in the cancel prompt (e.g., "skip", "cancel", "create new")
	CancelText string

	// Filter excludes tasks based on status or other criteria
	Filter *backend.TaskFilter

	// ConfirmSinglePartial asks for confirmation when only one partial match is found
	ConfirmSinglePartial bool

	// AllowEmpty allows returning nil when no tasks are found (instead of error)
	AllowEmpty bool
}

// NewTaskSelector creates a new TaskSelector instance.
func NewTaskSelector(taskManager backend.TaskManager, cfg *config.Config) *TaskSelector {
	return &TaskSelector{
		taskManager: taskManager,
		config:      cfg,
	}
}

// Select finds and selects a task based on the search term and options.
// This is the unified entry point replacing all the individual selection functions.
func (ts *TaskSelector) Select(listID string, searchTerm string, opts SelectionOptions) (*backend.Task, error) {
	// If no search term and we're in interactive mode, show all tasks
	if searchTerm == "" && opts.DisplayFormat == "tree" {
		return ts.selectFromAll(listID, opts)
	}

	// Search for matching tasks
	matches, err := ts.taskManager.FindTasksBySummary(listID, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("error searching for tasks: %w", err)
	}

	// Apply filter to exclude certain tasks
	if opts.Filter != nil {
		matches = ts.applyFilter(matches, opts.Filter)
	}

	// Handle no matches
	if len(matches) == 0 {
		if opts.AllowEmpty {
			return nil, nil
		}
		return nil, fmt.Errorf("no tasks found matching '%s'", searchTerm)
	}

	// Handle exact and partial matches
	if task, done, err := ts.handleMatches(matches, searchTerm, listID, opts); done {
		return task, err
	}

	// Multiple matches - prompt for selection
	return ts.promptSelection(matches, searchTerm, listID, opts)
}

// selectFromAll shows all tasks in the list and prompts for selection (interactive mode).
func (ts *TaskSelector) selectFromAll(listID string, opts SelectionOptions) (*backend.Task, error) {
	tasks, err := ts.taskManager.GetTasks(listID, opts.Filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks in list")
	}

	// Always use tree format for interactive selection
	return ts.displayTreeAndSelect(tasks, listID, opts.CancelText)
}

// handleMatches processes exact and partial matches, returning early if a single match is found.
func (ts *TaskSelector) handleMatches(matches []backend.Task, searchTerm string, listID string, opts SelectionOptions) (*backend.Task, bool, error) {
	// Separate exact and partial matches
	var exactMatches []backend.Task
	var partialMatches []backend.Task
	searchLower := strings.ToLower(searchTerm)

	for _, task := range matches {
		if strings.ToLower(task.Summary) == searchLower {
			exactMatches = append(exactMatches, task)
		} else {
			partialMatches = append(partialMatches, task)
		}
	}

	// Single exact match - proceed without confirmation
	if len(exactMatches) == 1 && len(partialMatches) == 0 {
		return &exactMatches[0], true, nil
	}

	// Single partial match - ask for confirmation if configured
	if len(exactMatches) == 0 && len(partialMatches) == 1 && opts.ConfirmSinglePartial {
		task := &partialMatches[0]
		confirmed, err := confirmTask(task, ts.taskManager, ts.config)
		if err != nil {
			return nil, true, err
		}
		if !confirmed {
			return nil, true, fmt.Errorf("operation cancelled")
		}
		return task, true, nil
	}

	// Single partial match without confirmation required
	if len(exactMatches) == 0 && len(partialMatches) == 1 {
		return &partialMatches[0], true, nil
	}

	// Multiple matches - caller should prompt for selection
	return nil, false, nil
}

// promptSelection displays tasks and prompts the user to select one.
func (ts *TaskSelector) promptSelection(tasks []backend.Task, searchTerm string, listID string, opts SelectionOptions) (*backend.Task, error) {
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks to select from")
	}

	// Choose display format
	if opts.DisplayFormat == "tree" {
		return ts.displayTreeAndSelect(tasks, listID, opts.CancelText)
	}

	// List format
	if opts.ShowHierarchy {
		return ts.displayListWithHierarchy(tasks, searchTerm, listID, opts.CancelText)
	}

	return ts.displayListSimple(tasks, searchTerm, opts.CancelText)
}

// displayListWithHierarchy shows tasks with their hierarchical paths.
func (ts *TaskSelector) displayListWithHierarchy(tasks []backend.Task, searchTerm string, listID string, cancelText string) (*backend.Task, error) {
	// Get all tasks to build hierarchy
	allTasks, err := ts.taskManager.GetTasks(listID, nil)
	if err != nil {
		// Fall back to simple display if we can't get all tasks
		return ts.displayListSimple(tasks, searchTerm, cancelText)
	}

	// Build UID to task map for path resolution
	taskMap := make(map[string]*backend.Task)
	for i := range allTasks {
		taskMap[allTasks[i].UID] = &allTasks[i]
	}

	// Show tasks with hierarchical paths
	fmt.Printf("\n%d tasks found matching '%s':\n", len(tasks), searchTerm)
	dateFormat := ts.config.GetDateFormat()

	for i, task := range tasks {
		// Get hierarchical path
		path := GetTaskPath(&task, taskMap)
		fmt.Printf("\n%d: [%s]", i+1, path)
		fmt.Print(task.FormatWithView("all", ts.taskManager, dateFormat))
	}

	fmt.Printf("\nSelect task (1-%d) or 0 to %s: ", len(tasks), cancelText)
	choice, err := utils.ReadInt()
	if err != nil {
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

// displayListSimple shows tasks without hierarchical paths (fallback).
func (ts *TaskSelector) displayListSimple(tasks []backend.Task, searchTerm string, cancelText string) (*backend.Task, error) {
	fmt.Printf("\n%d tasks found matching '%s':\n", len(tasks), searchTerm)
	dateFormat := ts.config.GetDateFormat()

	for i, task := range tasks {
		fmt.Printf("\n%d:", i+1)
		fmt.Print(task.FormatWithView("all", ts.taskManager, dateFormat))
	}

	fmt.Printf("\nSelect task (1-%d) or 0 to %s: ", len(tasks), cancelText)
	choice, err := utils.ReadInt()
	if err != nil {
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

// displayTreeAndSelect shows tasks in a tree format and prompts for selection.
func (ts *TaskSelector) displayTreeAndSelect(tasks []backend.Task, listID string, cancelText string) (*backend.Task, error) {
	// Build task tree
	tree := BuildTaskTree(tasks)
	if len(tree) == 0 {
		return nil, fmt.Errorf("no tasks to display")
	}

	// Flatten for numbered selection
	var flatTasks []*backend.Task
	buildFlatTaskList(tree, &flatTasks)

	if len(flatTasks) == 0 {
		return nil, fmt.Errorf("no tasks available")
	}

	// Calculate border width
	borderWidth := 80
	if termWidth := cli.GetTerminalWidth(); termWidth > 0 {
		borderWidth = termWidth - 2
	}
	if borderWidth > 100 {
		borderWidth = 100
	}

	// Display tasks with numbering
	headerText := "─ Available Tasks "
	headerPadding := borderWidth - len(headerText)
	if headerPadding < 0 {
		headerPadding = 0
	}
	fmt.Printf("\n\033[1;36m┌%s%s┐\033[0m\n", headerText, strings.Repeat("─", headerPadding))

	// Format and print the tree
	output, _ := formatTaskTreeNumbered(tree, ts.taskManager, ts.config.GetDateFormat(), 1, "", true)
	fmt.Print(output)

	// Display footer
	fmt.Printf("\033[1;36m└%s┘\033[0m\n", strings.Repeat("─", borderWidth))

	// Prompt for selection
	fmt.Printf("\n\033[1mSelect task (1-%d, or 0 to %s):\033[0m ", len(flatTasks), cancelText)
	choice, err := utils.ReadInt()
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	if choice == 0 {
		return nil, fmt.Errorf("cancelled")
	}

	if choice < 1 || choice > len(flatTasks) {
		return nil, fmt.Errorf("invalid choice: %d (must be 1-%d)", choice, len(flatTasks))
	}

	return flatTasks[choice-1], nil
}

// applyFilter filters tasks based on the provided filter criteria.
func (ts *TaskSelector) applyFilter(tasks []backend.Task, filter *backend.TaskFilter) []backend.Task {
	if filter == nil {
		return tasks
	}

	// Apply exclude statuses filter
	if filter.ExcludeStatuses != nil && len(*filter.ExcludeStatuses) > 0 {
		excludeMap := make(map[string]bool)
		for _, status := range *filter.ExcludeStatuses {
			excludeMap[status] = true
		}

		var filtered []backend.Task
		for _, task := range tasks {
			if !excludeMap[task.Status] {
				filtered = append(filtered, task)
			}
		}
		return filtered
	}

	return tasks
}

// DefaultOptions returns sensible default selection options.
func DefaultOptions() SelectionOptions {
	return SelectionOptions{
		ShowHierarchy:        true,
		DisplayFormat:        "list",
		CancelText:           "cancel",
		ConfirmSinglePartial: true,
		AllowEmpty:           false,
	}
}
