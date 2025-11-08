package main

import (
	"gosynctasks/internal/app"
	"gosynctasks/internal/cli"
	"log"

	"github.com/spf13/cobra"
)

func main() {
	application, err := app.NewApp()
	if err != nil {
		log.Fatal("Failed to initialize app:", err)
	}

	rootCmd := &cobra.Command{
		Use:   "gosynctasks [list-name] [action] [task-summary]",
		Short: "Task synchronization tool",
		Long: `Task synchronization tool for managing tasks across different backends.

Actions (abbreviations in parentheses):
  get (g)       - List tasks from a task list (default action)
  add (a)       - Add a new task to a list
  update (u)    - Update an existing task by summary
  complete (c)  - Change task status by summary (defaults to DONE)

Examples:
  gosynctasks                           # Interactive list selection, show tasks
  gosynctasks MyList                    # Show tasks from "MyList"
  gosynctasks MyList get                # Show tasks from "MyList" (g also works)
  gosynctasks MyList -s TODO,PROCESSING # Filter tasks by status

  gosynctasks MyList add "New task"     # Add a task to "MyList"
  gosynctasks MyList a "New task"       # Same using abbreviation
  gosynctasks MyList add                # Add a task (will prompt for summary)
  gosynctasks MyList add "Task" -d "Details" -p 1 -S done  # Add with options

  gosynctasks MyList update "Buy groceries" -s DONE  # Update task status
  gosynctasks MyList u "groceries" --summary "Buy milk"  # Partial match + rename
  gosynctasks MyList update "task" -p 5              # Partial match + set priority

  gosynctasks MyList complete "Buy groceries"      # Mark as DONE (default)
  gosynctasks MyList c "groceries" -s TODO         # Mark as TODO
  gosynctasks MyList c "task" -s PROCESSING        # Mark as PROCESSING
  gosynctasks MyList c "old task" -s CANCELLED     # Mark as CANCELLED`,
		Args:              cobra.MaximumNArgs(3),
		ValidArgsFunction: cli.SmartCompletion(application.GetTaskLists()),
		RunE:              application.Run,
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
