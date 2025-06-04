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

var taskLists []backend.TaskList

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

func loadTaskLists() error {
    config := config.GetConfig()
    taskManager, err := config.Connector.TaskManager()
    if err != nil {
        return err
    }

    taskLists, err = taskManager.GetTaskLists()
    return err
}

func listNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    var completions []string
    for _, list := range taskLists {
        if strings.HasPrefix(strings.ToLower(list.Name), strings.ToLower(toComplete)) {
            completions = append(completions, list.Name)
        }
    }
    return completions, cobra.ShellCompDirectiveNoFileComp
}

func findListByName(name string) *backend.TaskList {
    for _, list := range taskLists {
        if strings.EqualFold(list.Name, name) {
            return &list
        }
    }
    return nil
}

func cli_ListSelection(lists *[]backend.TaskList) (*backend.TaskList, error) {
    for i, list := range *lists {
        desc := ""
        if list.Description != "" {
            desc = " - " + list.Description
        }
        fmt.Printf("%d: %s%s\n", i+1, list.Name, desc)
    }

    fmt.Printf("Enter selection (1-%d):\n", len(*lists))
    var choice int
    _, err := fmt.Scanf("%d", &choice)
    if err != nil || choice < 1 || choice > len(*lists) {
        return nil, fmt.Errorf("invalid choice: %d", choice)
    }
    return &(*lists)[choice-1], err
}

func main() {
	// trying to add autocomplete TODO: doesn't seem to work
    if err := loadTaskLists(); err != nil {
        log.Printf("Warning: Could not load task lists for autocomplete: %v", err)
    }

    filter := &backend.TaskFilter{}
    var listName string

    var rootCmd = &cobra.Command{
        Use:   "gosynctasks",
        Short: "Task synchronization tool",
    }

    var listCmd = &cobra.Command{
        Use:   "list [list-name]",
        Aliases: []string{"l"},
        Short: "select task list",
        Args:  cobra.ExactArgs(1),
        ValidArgsFunction: listNameCompletion,
        Run: func(cmd *cobra.Command, args []string) {
            listName = args[0]
        },
    }

    var statusCmd = &cobra.Command{
        Use:     "status",
        Aliases: []string{"s"},
        Short:   "filter task by status",
        Run: func(cmd *cobra.Command, args []string) {
            var statuses []string

            done, _ := cmd.Flags().GetBool("done")
            todo, _ := cmd.Flags().GetBool("todo")
            processing, _ := cmd.Flags().GetBool("processing")
            cancelled, _ := cmd.Flags().GetBool("cancelled")

            if done {
                statuses = append(statuses, "DONE")
            }
            if todo {
                statuses = append(statuses, "TODO")
            }
            if processing {
                statuses = append(statuses, "PROCESSING")
            }
            if cancelled {
                statuses = append(statuses, "CANCELLED")
            }

            if len(statuses) > 0 {
                filter.Statuses = &statuses
            }
        },
    }

    statusCmd.Flags().BoolP("done", "d", false, "filter done tasks")
    statusCmd.Flags().BoolP("todo", "t", false, "filter todo tasks")
    statusCmd.Flags().BoolP("processing", "p", false, "filter processing tasks")
    statusCmd.Flags().BoolP("cancelled", "c", false, "filter cancelled tasks")

    rootCmd.AddCommand(listCmd)
    rootCmd.AddCommand(statusCmd)

    if err := rootCmd.Execute(); err != nil {
        log.Fatal(err)
    }

    // Main logic
    config := config.GetConfig()
    taskManager, err := config.Connector.TaskManager()
    if err != nil {
        log.Fatalln("There was a problem with connector's task manager.")
    }

    // If taskLists not loaded yet, load them now
    if len(taskLists) == 0 {
        taskLists, err = taskManager.GetTaskLists()
        if err != nil {
            fmt.Printf("there was an error while retrieving task lists: %s", err)
            return
        }
    }

    var selectedList *backend.TaskList

    if listName != "" {
        selectedList = findListByName(listName)
        if selectedList == nil {
            fmt.Printf("List '%s' not found\n", listName)
            return
        }
    } else {
        selectedList, err = cli_ListSelection(&taskLists)
        if err != nil {
            fmt.Printf("error: %s\n", err)
            return
        }
    }

    fmt.Println(selectedList)
    tasks, err := taskManager.GetTasks(selectedList.ID, filter)
    if err != nil {
        fmt.Println("There was an error while retrieving tasks:")
        fmt.Println(err)
    }
    fmt.Println(tasks)

    fmt.Println(*config)
    fmt.Println("End.")
}
