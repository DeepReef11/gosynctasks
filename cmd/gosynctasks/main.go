package main

import (
	// "context"
	"encoding/json"
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/internal/config"
	"log"
	"os"
	"path/filepath"
	"time"
	"database/sql"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
	"strings"
)

type App struct {
	taskLists   []backend.TaskList
	taskManager backend.TaskManager
	config      *config.Config
}

func NewApp() (*App, error) {
	cfg := config.GetConfig()
	taskManager, err := cfg.Connector.TaskManager()
	if err != nil {
		return nil, err
	}

	app := &App{
		config:      cfg,
		taskManager: taskManager,
	}

	if err := app.loadTaskLists(); err != nil {
		log.Printf("Warning: Could not load task lists: %v", err)
	}

	return app, nil
}

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "tasks.db")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS tasks (
            id TEXT PRIMARY KEY,
            content TEXT,
            status TEXT,
            created_at INTEGER,
            updated_at INTEGER
        )
    `)
	return db, err
}

func getCacheDir() (string, error) {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		cacheDir = filepath.Join(home, ".cache")
	}
	cacheDir = filepath.Join(cacheDir, "gosynctasks")
	return cacheDir, os.MkdirAll(cacheDir, 0755)
}

func getCacheFile() (string, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "lists.json"), nil
}

type cachedData struct {
	Lists     []backend.TaskList `json:"lists"`
	Timestamp int64              `json:"timestamp"`
}

func (a *App) loadTaskListsFromCache() error {
	cacheFile, err := getCacheFile()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return err
	}

	var cached cachedData
	if err := json.Unmarshal(data, &cached); err != nil {
		return err
	}

	a.taskLists = cached.Lists
	return nil
}

func (a *App) saveTaskListsToCache() error {
	cacheFile, err := getCacheFile()
	if err != nil {
		return err
	}

	cached := cachedData{
		Lists:     a.taskLists,
		Timestamp: time.Now().Unix(),
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

func (a *App) loadTaskLists() error {
	// Try cache first
	if err := a.loadTaskListsFromCache(); err == nil {
		return nil
	}

	// Fetch from remote
	var err error
	a.taskLists, err = a.taskManager.GetTaskLists()
	if err != nil {
		return err
	}

	// Save to cache for next time
	_ = a.saveTaskListsToCache()
	return nil
}

func (a *App) refreshTaskLists() error {
	var err error
	a.taskLists, err = a.taskManager.GetTaskLists()
	if err != nil {
		return err
	}
	_ = a.saveTaskListsToCache()
	return nil
}

func (a *App) smartCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var completions []string

	// First argument: suggest actions OR list names
	if len(args) == 0 {
		actions := []string{"get", "g", "add", "a", "update", "u", "complete", "c"}
		for _, action := range actions {
			if strings.HasPrefix(action, strings.ToLower(toComplete)) {
				completions = append(completions, action)
			}
		}

		// Also suggest list names for direct access
		for _, list := range a.taskLists {
			if strings.HasPrefix(strings.ToLower(list.Name), strings.ToLower(toComplete)) {
				completions = append(completions, list.Name)
			}
		}
	}

	// Second argument (after action): suggest list names only
	if len(args) == 1 {
		for _, list := range a.taskLists {
			if strings.HasPrefix(strings.ToLower(list.Name), strings.ToLower(toComplete)) {
				completions = append(completions, list.Name)
			}
		}
	}

	// Third argument (after "add <list>"): no completion, user enters task summary
	// Return directive to stop completion
	if len(args) >= 2 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

func (a *App) findListByName(name string) *backend.TaskList {
	for _, list := range a.taskLists {
		if strings.EqualFold(list.Name, name) {
			return &list
		}
	}
	return nil
}

func (a *App) cli_showList() {
	for i, list := range a.taskLists {
		desc := ""
		if list.Description != "" {
			desc = " - " + list.Description
		}
		fmt.Printf("%d: %s%s\n", i+1, list.Name, desc)
	}
}

func (a *App) selectListInteractively() (*backend.TaskList, error) {
	a.cli_showList()

	fmt.Printf("Enter selection (1-%d): ", len(a.taskLists))
	var choice int
	if _, err := fmt.Scanf("%d", &choice); err != nil || choice < 1 || choice > len(a.taskLists) {
		return nil, fmt.Errorf("invalid choice: %d", choice)
	}
	return &a.taskLists[choice-1], nil
}

func (a *App) getSelectedList(listName string) (*backend.TaskList, error) {
	if listName != "" {
		selectedList := a.findListByName(listName)
		if selectedList == nil {
			return nil, fmt.Errorf("list '%s' not found", listName)
		}
		return selectedList, nil
	}

	// No list name provided, use interactive selection
	return a.selectListInteractively()
}

func normalizeAction(action string) string {
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

// findTaskBySummary searches for a task by summary and handles UX for exact/partial/multiple matches
func (a *App) findTaskBySummary(listID string, searchSummary string) (*backend.Task, error) {
	// Use backend's search method
	matches, err := a.taskManager.FindTasksBySummary(listID, searchSummary)
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
		confirmed, err := a.confirmTask(task)
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
		return a.selectTask(exactMatches, searchSummary)
	}

	// Mix of exact and partial, or multiple partial
	return a.selectTask(matches, searchSummary)
}

// selectTask shows a list of tasks and prompts user to select one
func (a *App) selectTask(tasks []backend.Task, searchSummary string) (*backend.Task, error) {
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks found matching '%s'", searchSummary)
	}

	// Show tasks with "all" view
	fmt.Printf("\n%d tasks found matching '%s':\n", len(tasks), searchSummary)
	dateFormat := a.config.GetDateFormat()

	for i, task := range tasks {
		fmt.Printf("\n%d:", i+1)
		fmt.Print(task.FormatWithView("all", a.taskManager, dateFormat))
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
func (a *App) confirmTask(task *backend.Task) (bool, error) {
	dateFormat := a.config.GetDateFormat()
	fmt.Println("\nTask found:")
	fmt.Print(task.FormatWithView("all", a.taskManager, dateFormat))
	fmt.Print("\nProceed with this task? (y/n): ")

	var response string
	if _, err := fmt.Scanf("%s", &response); err != nil {
		return false, fmt.Errorf("invalid input: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}

func (a *App) buildFilter(cmd *cobra.Command) *backend.TaskFilter {
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

func (a *App) run(cmd *cobra.Command, args []string) error {
	// Refresh task lists from remote for actual operations
	if err := a.refreshTaskLists(); err != nil {
		log.Printf("Warning: Could not refresh task lists: %v", err)
	}

	var listName string
	var taskSummary string
	var searchSummary string
	action := "get"

	if len(args) == 1 {
		listName = args[0]
	}
	if len(args) >= 2 {
		action = args[0]
		listName = args[1]
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
	action = normalizeAction(action)

	filter := a.buildFilter(cmd)

	selectedList, err := a.getSelectedList(listName)
	if err != nil {
		return err
	}

	switch action {
	case "get":
		tasks, err := a.taskManager.GetTasks(selectedList.ID, filter)
		if err != nil {
			return fmt.Errorf("error retrieving tasks: %w", err)
		}

		// Sort using backend-specific sorting
		a.taskManager.SortTasks(tasks)

		view, _ := cmd.Flags().GetString("view")
		dateFormat := a.config.GetDateFormat()

		fmt.Println(selectedList)
		for _, task := range tasks {
			fmt.Print(task.FormatWithView(view, a.taskManager, dateFormat))
		}
		fmt.Print(selectedList.BottomBorder())
		return nil

	case "add":
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

		if err := a.taskManager.AddTask(selectedList.ID, task); err != nil {
			return fmt.Errorf("error adding task: %w", err)
		}

		fmt.Printf("Task '%s' added successfully to list '%s'\n", taskSummary, selectedList.Name)
		return nil

	case "update":
		// Get the task summary to search for
		if searchSummary == "" {
			return fmt.Errorf("task summary is required for update (usage: gosynctasks update <list> <task-summary>)")
		}

		// Find the task by summary (handles exact/partial/multiple matches)
		taskToUpdate, err := a.findTaskBySummary(selectedList.ID, searchSummary)
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
		if err := a.taskManager.UpdateTask(selectedList.ID, *taskToUpdate); err != nil {
			return fmt.Errorf("error updating task: %w", err)
		}

		fmt.Printf("Task '%s' updated successfully in list '%s'\n", taskToUpdate.Summary, selectedList.Name)
		return nil

	case "complete":
		// Get the task summary to search for
		if searchSummary == "" {
			return fmt.Errorf("task summary is required for complete (usage: gosynctasks complete <list> <task-summary>)")
		}

		// Find the task by summary (handles exact/partial/multiple matches)
		taskToComplete, err := a.findTaskBySummary(selectedList.ID, searchSummary)
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
		if err := a.taskManager.UpdateTask(selectedList.ID, *taskToComplete); err != nil {
			return fmt.Errorf("error updating task: %w", err)
		}

		fmt.Printf("Task '%s' marked as %s in list '%s'\n", taskToComplete.Summary, statusName, selectedList.Name)
		return nil

	default:
		return fmt.Errorf("unknown action: %s (supported: get/g, add/a, update/u, complete/c)", action)
	}
}

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal("Failed to initialize app:", err)
	}

	rootCmd := &cobra.Command{
		Use:               "gosynctasks [action] [list-name] [task-summary]",
		Short:             "Task synchronization tool",
		Long: `Task synchronization tool for managing tasks across different backends.

Actions (abbreviations in parentheses):
  get (g)       - List tasks from a task list
  add (a)       - Add a new task to a list
  update (u)    - Update an existing task by summary
  complete (c)  - Change task status by summary (defaults to DONE)

Examples:
  gosynctasks                           # Interactive list selection, show tasks
  gosynctasks MyList                    # Show tasks from "MyList"
  gosynctasks get MyList                # Show tasks from "MyList" (g also works)
  gosynctasks -s TODO,PROCESSING MyList # Filter tasks by status

  gosynctasks add MyList "New task"     # Add a task to "MyList"
  gosynctasks a MyList "New task"       # Same using abbreviation
  gosynctasks add MyList                # Add a task (will prompt for summary)
  gosynctasks add MyList "Task" -d "Details" -p 1 -S done  # Add with options

  gosynctasks update MyList "Buy groceries" -s DONE  # Update task status
  gosynctasks u MyList "groceries" --summary "Buy milk"  # Partial match + rename
  gosynctasks update MyList "task" -p 5              # Partial match + set priority

  gosynctasks complete MyList "Buy groceries"      # Mark as DONE (default)
  gosynctasks c MyList "groceries" -s TODO         # Mark as TODO
  gosynctasks c MyList "task" -s PROCESSING        # Mark as PROCESSING
  gosynctasks c MyList "old task" -s CANCELLED     # Mark as CANCELLED`,
		Args:              cobra.MaximumNArgs(3),
		ValidArgsFunction: app.smartCompletion,
		RunE:              app.run,
	}

	rootCmd.Flags().StringArrayP("status", "s", []string{}, "filter by status (for get) or set status (for update): [T]ODO, [D]ONE, [P]ROCESSING, [C]ANCELLED")
	rootCmd.Flags().StringP("view", "v", "basic", "view mode (basic, all)")
	rootCmd.Flags().StringP("description", "d", "", "task description (for add/update)")
	rootCmd.Flags().IntP("priority", "p", 0, "task priority (for add/update, 0-9: 0=undefined, 1=highest, 9=lowest)")
	rootCmd.Flags().StringP("add-status", "S", "", "task status when adding (TODO/T, DONE/D, PROCESSING/P, CANCELLED/C)")
	rootCmd.Flags().String("summary", "", "task summary (for update)")

	// Register flag value completion for status flags
	rootCmd.RegisterFlagCompletionFunc("status", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"TODO", "DONE", "PROCESSING", "CANCELLED"}, cobra.ShellCompDirectiveNoFileComp
	})
	rootCmd.RegisterFlagCompletionFunc("add-status", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"TODO", "DONE", "PROCESSING", "CANCELLED"}, cobra.ShellCompDirectiveNoFileComp
	})

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
