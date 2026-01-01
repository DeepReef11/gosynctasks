package main

import (
	"gosynctasks/internal/app"
	"gosynctasks/internal/cli"
	"gosynctasks/internal/config"
	"gosynctasks/internal/utils"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	configPath     string
	backendName    string
	listBackends   bool
	detectBackends bool
	verbose        bool
	application    *app.App
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gosynctasks [list-name] [action] [task-summary]",
		Short: "Task synchronization tool",
		Long: `Task synchronization tool for managing tasks across different backends.

Actions (abbreviations in parentheses):
  get (g)       - List tasks from a task list (default action)
  add (a)       - Add a new task to a list
  update (u)    - Update an existing task by summary
  complete (c)  - Change task status by summary (defaults to DONE)
  delete (d)    - Delete a task by summary

Examples:
  gosynctasks                           # Interactive list selection, show tasks
  gosynctasks MyList                    # Show tasks from "MyList"
  gosynctasks MyList get                # Show tasks from "MyList" (g also works)
  gosynctasks MyList -s TODO,PROCESSING # Filter tasks by status

  gosynctasks MyList add "New task"     # Add a task to "MyList"
  gosynctasks MyList a "New task"       # Same using abbreviation
  gosynctasks MyList add                # Add a task (will prompt for summary)
  gosynctasks MyList add "Task" -d "Details" -p 1 -S done  # Add with options
  gosynctasks MyList add "Report" --due-date 2025-01-31 --start-date 2025-01-15  # With dates
  gosynctasks MyList add "Subtask" -P "Parent Task"  # Add subtask under parent
  gosynctasks MyList add "Fix bug" -P "Feature/Code"  # Path-based parent reference
  gosynctasks MyList add "parent/child/grandchild"  # Shorthand: auto-creates hierarchy
  gosynctasks MyList add -l "be a good/generous person"  # Use -l to disable path parsing

  gosynctasks MyList update "Buy groceries" -s DONE  # Update task status
  gosynctasks MyList u "groceries" --summary "Buy milk"  # Partial match + rename
  gosynctasks MyList update "task" -p 5              # Partial match + set priority
  gosynctasks MyList update "task" --due-date 2025-02-15  # Update due date

  gosynctasks MyList complete "Buy groceries"      # Mark as DONE (default)
  gosynctasks MyList c "groceries"

  gosynctasks MyList delete "Buy groceries"        # Delete a task
  gosynctasks MyList d "groceries"                 # Same using abbreviation

Config:
  --config .                            # Use ./gosynctasks/config.json
  --config /path/to/config.json         # Use specific config file
  --config /path/to/dir                 # Use /path/to/dir/config.json
`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Set verbose mode first
			if verbose {
				utils.SetVerboseMode(true)
				utils.Debugf("Verbose mode enabled")
			}

			// Set custom config path if specified
			if configPath != "" {
				config.SetCustomConfigPath(configPath)
				utils.Debugf("Using custom config path: %s", configPath)
			}

			// Initialize app after config path is set
			var err error
			application, err = app.NewApp(backendName)
			if err != nil {
				return err
			}
			if backendName != "" {
				utils.Debugf("Application initialized with backend argument: %s", backendName)
			}

			// Handle --list-backends flag
			if listBackends {
				return application.ListBackends()
			}

			// Handle --detect-backend flag
			if detectBackends {
				return application.DetectBackends()
			}

			return nil
		},
		Args: cobra.MaximumNArgs(3),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if application == nil {
				return []string{}, cobra.ShellCompDirectiveNoFileComp
			}
			return cli.SmartCompletion(application.GetTaskLists())(cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return application.Run(cmd, args)
		},
	}

	// Persistent flags (available to all commands)
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path (default: $XDG_CONFIG_HOME/gosynctasks/config.json, use '.' for ./gosynctasks/config.json)")
	rootCmd.PersistentFlags().StringVar(&backendName, "backend", "", "backend to use (overrides config default and auto-detection)")
	rootCmd.PersistentFlags().BoolVar(&listBackends, "list-backends", false, "list all configured backends and exit")
	rootCmd.PersistentFlags().BoolVar(&detectBackends, "detect-backend", false, "show auto-detected backends and exit")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "enable verbose/debug logging")

	// Command flags
	rootCmd.Flags().StringArrayP("status", "s", []string{}, "filter by status (for get) or set status (for update): [T]ODO, [D]ONE, [P]ROCESSING, [C]ANCELLED")
	rootCmd.Flags().StringP("view", "v", "default", "view mode (default, all, or custom view name)")
	rootCmd.Flags().StringP("description", "d", "", "task description (for add/update)")
	rootCmd.Flags().IntP("priority", "p", 0, "task priority (for add/update, 0-9: 0=undefined, 1=highest, 9=lowest)")
	rootCmd.Flags().StringP("add-status", "S", "", "task status when adding (TODO/T, DONE/D, PROCESSING/P, CANCELLED/C)")
	rootCmd.Flags().String("summary", "", "task summary (for update)")
	rootCmd.Flags().String("due-date", "", "task due date (for add/update, format: YYYY-MM-DD, empty string to clear)")
	rootCmd.Flags().String("start-date", "", "task start date (for add/update, format: YYYY-MM-DD, empty string to clear)")
	rootCmd.Flags().StringP("parent", "P", "", "parent task reference (for add): task summary or path like 'Parent/Child'")
	rootCmd.Flags().BoolP("literal", "l", false, "treat task summary literally (for add): disable automatic path-based hierarchy creation")

	// Register flag value completion for status flags
	_ = rootCmd.RegisterFlagCompletionFunc("status", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"TODO", "DONE", "PROCESSING", "CANCELLED"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = rootCmd.RegisterFlagCompletionFunc("add-status", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"TODO", "DONE", "PROCESSING", "CANCELLED"}, cobra.ShellCompDirectiveNoFileComp
	})

	// Register view flag completion
	_ = rootCmd.RegisterFlagCompletionFunc("view", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if application == nil {
			return []string{"default", "all"}, cobra.ShellCompDirectiveNoFileComp
		}
		viewNames, err := cli.ListViewNames()
		if err != nil {
			return []string{"default", "all"}, cobra.ShellCompDirectiveNoFileComp
		}
		return viewNames, cobra.ShellCompDirectiveNoFileComp
	})

	// Add subcommands
	rootCmd.AddCommand(newViewCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newSyncCmd())
	rootCmd.AddCommand(newCredentialsCmd())

	// Set up graceful shutdown on Ctrl+C / SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		if application != nil {
			application.Shutdown()
		}
		os.Exit(0)
	}()

	// Execute command
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}

	// Exit immediately - background sync runs in detached process
	// Operations are queued in sqlite and synced by background daemon
}
