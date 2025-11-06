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
		actions := []string{"get", "add"}
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
	action := "get"

	if len(args) == 1 {
		listName = args[0]
	}
	if len(args) >= 2 {
		action = args[0]
		listName = args[1]
	}
	if len(args) >= 3 {
		taskSummary = args[2]
	}

	filter := a.buildFilter(cmd)

	selectedList, err := a.getSelectedList(listName)
	if err != nil {
		return err
	}

	switch strings.ToLower(action) {
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

		task := backend.Task{
			Summary: taskSummary,
			Status:  "NEEDS-ACTION",
		}

		if err := a.taskManager.AddTask(selectedList.ID, task); err != nil {
			return fmt.Errorf("error adding task: %w", err)
		}

		fmt.Printf("Task '%s' added successfully to list '%s'\n", taskSummary, selectedList.Name)
		return nil

	default:
		return fmt.Errorf("unknown action: %s (supported: get, add)", action)
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

Examples:
  gosynctasks                           # Interactive list selection, show tasks
  gosynctasks MyList                    # Show tasks from "MyList"
  gosynctasks get MyList                # Show tasks from "MyList"
  gosynctasks add MyList "New task"     # Add a task to "MyList"
  gosynctasks add MyList                # Add a task (will prompt for summary)
  gosynctasks -s TODO,PROCESSING MyList # Filter tasks by status`,
		Args:              cobra.MaximumNArgs(3),
		ValidArgsFunction: app.smartCompletion,
		RunE:              app.run,
	}

	rootCmd.Flags().StringArrayP("status", "s", []string{}, "filter by status ([T]ODO, [D]ONE, [P]ROCESSING, [C]ANCELLED)")
	rootCmd.Flags().StringP("view", "v", "basic", "view mode (basic, all)")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
