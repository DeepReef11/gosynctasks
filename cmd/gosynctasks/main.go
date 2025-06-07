package main

import (
    // "context"
    "fmt"
    "gosynctasks/backend"
    "gosynctasks/internal/config"
    "log"
    // "os"
    "database/sql"
    "strings"
    "github.com/spf13/cobra"
    _ "modernc.org/sqlite"
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


func (a *App) loadTaskLists() error {
    var err error
    a.taskLists, err = a.taskManager.GetTaskLists()
    return err
}

func (a *App) listNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    var completions []string
    for _, list := range a.taskLists {
        if strings.HasPrefix(strings.ToLower(list.Name), strings.ToLower(toComplete)) {
            completions = append(completions, list.Name)
        }
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

func (a *App) selectListInteractively() (*backend.TaskList, error) {
    for i, list := range a.taskLists {
        desc := ""
        if list.Description != "" {
            desc = " - " + list.Description
        }
        fmt.Printf("%d: %s%s\n", i+1, list.Name, desc)
    }

    fmt.Printf("Enter selection (1-%d): ", len(a.taskLists))
    var choice int
    if _, err := fmt.Scanf("%d", &choice); err != nil || choice < 1 || choice > len(a.taskLists) {
        return nil, fmt.Errorf("invalid choice: %d", choice)
    }
    return &a.taskLists[choice-1], nil
}


func (a *App) buildFilter(cmd *cobra.Command) *backend.TaskFilter {
    filter := &backend.TaskFilter{}
    var statuses []string

    if done, _ := cmd.Flags().GetBool("done"); done {
        statuses = append(statuses, "DONE")
    }
    if todo, _ := cmd.Flags().GetBool("todo"); todo {
        statuses = append(statuses, "TODO")
    }
    if processing, _ := cmd.Flags().GetBool("processing"); processing {
        statuses = append(statuses, "PROCESSING")
    }
    if cancelled, _ := cmd.Flags().GetBool("cancelled"); cancelled {
        statuses = append(statuses, "CANCELLED")
    }

    if len(statuses) > 0 {
        filter.Statuses = &statuses
    }
    return filter
}

func (a *App) run(cmd *cobra.Command, args []string) error {
    var listName string
    if len(args) > 0 {
        listName = args[0]
    }

    filter := a.buildFilter(cmd)

    var selectedList *backend.TaskList
    var err error

    if listName != "" {
        selectedList = a.findListByName(listName)
        if selectedList == nil {
            return fmt.Errorf("list '%s' not found", listName)
        }
    } else {
        selectedList, err = a.selectListInteractively()
        if err != nil {
            return err
        }
    }

    tasks, err := a.taskManager.GetTasks(selectedList.ID, filter)
    if err != nil {
        return fmt.Errorf("error retrieving tasks: %w", err)
    }

    fmt.Println(selectedList)
    fmt.Println(tasks)
    return nil
}

func main() {
    app, err := NewApp()
    if err != nil {
        log.Fatal("Failed to initialize app:", err)
    }

    rootCmd := &cobra.Command{
        Use:               "gosynctasks [list-name]",
        Short:             "Task synchronization tool",
        Args:              cobra.MaximumNArgs(1),
        ValidArgsFunction: app.listNameCompletion,
        RunE:              app.run,
    }

    rootCmd.Flags().BoolP("done", "d", false, "filter done tasks")
    rootCmd.Flags().BoolP("todo", "t", false, "filter todo tasks")
    rootCmd.Flags().BoolP("processing", "p", false, "filter processing tasks")
    rootCmd.Flags().BoolP("cancelled", "c", false, "filter cancelled tasks")

    if err := rootCmd.Execute(); err != nil {
        log.Fatal(err)
    }
}
