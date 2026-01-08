package main

import (
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/internal/config"
	"gosynctasks/internal/operations"
	"gosynctasks/internal/utils"

	"github.com/spf13/cobra"
)

// newListCmd creates the list management command with all subcommands
func newListCmd() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Manage task lists",
		Long: `Manage task lists (create, delete, rename, info).

Task lists are collections/categories of tasks. In Nextcloud, these are
calendars that support VTODO components. In Git backend, these are
markdown headers (##).

Examples:
  gosynctasks list                                      # Show all lists (simple)
  gosynctasks list create "Work Tasks"                  # Create new list
  gosynctasks list create "Project" -d "Tasks for XYZ"  # With description
  gosynctasks list create "Urgent" --color "#ff0000"    # With color (Nextcloud)

  gosynctasks list delete "Old Tasks"                   # Delete list (with confirmation)
  gosynctasks list delete "Archive" --force             # Skip confirmation

  gosynctasks list rename "Old Name" "New Name"         # Rename list

  gosynctasks list info "Work Tasks"                    # Show list details
  gosynctasks list info --all                           # Show all lists with details
  gosynctasks list info "Work Tasks" --json             # JSON output`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Refresh task lists from remote (uses cache backend when sync disabled)
			application.RefreshTaskListsFromRemoteOrWarn()

			// Default action: show all lists (simple view)
			taskLists := application.GetTaskLists()
			if len(taskLists) == 0 {
				fmt.Println("No task lists found.")
				return nil
			}

			fmt.Println("\nAvailable task lists:")
			for _, list := range taskLists {
				if list.Description != "" {
					fmt.Printf("  • %s - %s\n", list.Name, list.Description)
				} else {
					fmt.Printf("  • %s\n", list.Name)
				}
			}
			fmt.Println()
			return nil
		},
	}

	// Add subcommands
	listCmd.AddCommand(newListCreateCmd())
	listCmd.AddCommand(newListDeleteCmd())
	listCmd.AddCommand(newListRenameCmd())
	listCmd.AddCommand(newListInfoCmd())
	listCmd.AddCommand(newListTrashCmd())

	return listCmd
}

// newListCreateCmd creates the 'list create' command
func newListCreateCmd() *cobra.Command {
	var description string
	var color string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new task list",
		Long: `Create a new task list with the given name.

For Nextcloud backend, this creates a new calendar that supports VTODO.
For Git backend, this creates a new header (##) in the markdown file.

The color parameter is Nextcloud-specific and will be ignored by other backends.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Get remote backend for list operations (bypasses cache when sync enabled)
			taskManager := application.GetRemoteBackend()
			if taskManager == nil {
				return fmt.Errorf("task manager not initialized")
			}

			// Create the list on remote backend
			listID, err := taskManager.CreateTaskList(name, description, color)
			if err != nil {
				return fmt.Errorf("failed to create list: %w", err)
			}

			// If sync is enabled, trigger a sync to pull the new list into the SQLite cache
			cfg := config.GetConfig()
			if cfg.Sync != nil && cfg.Sync.Enabled {
				// Get explicit backend from parent command's --backend flag
				explicitBackend, _ := cmd.Root().PersistentFlags().GetString("backend")

				// Perform sync to pull the new list into cache (quiet mode)
				if err := performSync(cfg, explicitBackend, true); err != nil {
					// Log warning but don't fail - the list was created successfully
					fmt.Printf("Warning: list created on remote but sync failed: %v\n", err)
					fmt.Println("Run 'gosynctasks sync' to manually sync the new list to your cache")
				}
			}

			// Refresh cache from remote to include the new list in memory
			application.RefreshTaskListsFromRemoteOrWarn()

			fmt.Printf("List '%s' created successfully (ID: %s)\n", name, listID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "List description")
	cmd.Flags().StringVar(&color, "color", "", "List color in hex format (e.g., #ff0000) - Nextcloud only")

	return cmd
}

// newListDeleteCmd creates the 'list delete' command
func newListDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a task list",
		Long: `Delete a task list and all tasks within it.

By default, prompts for confirmation showing the task count.
Use --force to skip the confirmation prompt.

WARNING: This permanently deletes the list and all its tasks.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Get remote backend for list operations (bypasses cache when sync enabled)
			taskManager := application.GetRemoteBackend()
			if taskManager == nil {
				return fmt.Errorf("task manager not initialized")
			}

			// Refresh task lists from backend
			application.RefreshTaskListsFromRemoteOrWarn()

			// Find the list by name
			taskLists := application.GetTaskLists()
			listID, err := operations.FindListByName(taskLists, name)
			if err != nil {
				return err
			}

			// Get task count
			var taskCount int
			tasks, err := taskManager.GetTasks(listID, nil)
			if err == nil {
				taskCount = len(tasks)
			}

			// Confirm deletion unless --force
			if !force {
				fmt.Printf("This will delete the list '%s' and all %d tasks in it.\n", name, taskCount)
				confirmed, err := utils.PromptConfirmation("Are you sure?")
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Println("Deletion cancelled.")
					return nil
				}
			}

			// Delete the list from remote backend
			if err := taskManager.DeleteTaskList(listID); err != nil {
				return fmt.Errorf("failed to delete list: %w", err)
			}

			// If sync is enabled, trigger a sync to update the SQLite cache
			cfg := config.GetConfig()
			if cfg.Sync != nil && cfg.Sync.Enabled {
				explicitBackend, _ := cmd.Root().PersistentFlags().GetString("backend")
				if err := performSync(cfg, explicitBackend, true); err != nil {
					fmt.Printf("Warning: list deleted on remote but sync failed: %v\n", err)
				}
			}

			// Clear cache
			application.RefreshTaskListsFromRemoteOrWarn()

			fmt.Printf("List '%s' deleted successfully.\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

// newListRenameCmd creates the 'list rename' command
func newListRenameCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "rename <old-name> <new-name>",
		Short: "Rename a task list",
		Long: `Rename a task list while preserving all tasks and metadata.

The new name must not already exist.
By default, prompts for confirmation.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldName := args[0]
			newName := args[1]

			// Get remote backend for list operations (bypasses cache when sync enabled)
			taskManager := application.GetRemoteBackend()
			if taskManager == nil {
				return fmt.Errorf("task manager not initialized")
			}

			// Refresh task lists from backend
			application.RefreshTaskListsFromRemoteOrWarn()

			// Find the old list by name
			taskLists := application.GetTaskLists()
			listID, err := operations.FindListByName(taskLists, oldName)
			if err != nil {
				return err
			}

			// Check if new name already exists
			_, err = operations.FindListByName(taskLists, newName)
			if err == nil {
				return fmt.Errorf("list '%s' already exists", newName)
			}

			// Confirm rename unless --force
			if !force {
				confirmed, err := utils.PromptConfirmation(fmt.Sprintf("Rename list '%s' to '%s'?", oldName, newName))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Println("Rename cancelled.")
					return nil
				}
			}

			// Rename the list on remote backend
			if err := taskManager.RenameTaskList(listID, newName); err != nil {
				return fmt.Errorf("failed to rename list: %w", err)
			}

			// If sync is enabled, trigger a sync to update the SQLite cache
			cfg := config.GetConfig()
			if cfg.Sync != nil && cfg.Sync.Enabled {
				explicitBackend, _ := cmd.Root().PersistentFlags().GetString("backend")
				if err := performSync(cfg, explicitBackend, true); err != nil {
					fmt.Printf("Warning: list renamed on remote but sync failed: %v\n", err)
				}
			}

			// Clear cache
			application.RefreshTaskListsFromRemoteOrWarn()

			fmt.Printf("List renamed from '%s' to '%s' successfully.\n", oldName, newName)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

// newListInfoCmd creates the 'list info' command
func newListInfoCmd() *cobra.Command {
	var showAll bool
	var jsonOutput bool
	var yamlOutput bool

	cmd := &cobra.Command{
		Use:   "info [name]",
		Short: "Show task list details",
		Long: `Display detailed information about a task list including:
- Name, description, and metadata
- Task count by status (TODO, DONE, PROCESSING, CANCELLED)
- Backend-specific information (URL, color, etc.)

Use --all to show info for all lists.
Use --json or --yaml for machine-readable output.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get task manager from application
			taskManager := application.GetTaskManager()
			if taskManager == nil {
				return fmt.Errorf("task manager not initialized")
			}

			// Refresh task lists from backend
			application.RefreshTaskListsFromRemoteOrWarn()

			taskLists := application.GetTaskLists()

			// Determine which lists to show
			var listsToShow []interface{}

			if showAll {
				// Show all lists
				for _, list := range taskLists {
					info := buildListInfo(taskManager, list)
					listsToShow = append(listsToShow, info)
				}
			} else {
				// Show specific list
				if len(args) == 0 {
					return fmt.Errorf("list name required (or use --all)")
				}

				name := args[0]
				list, err := operations.FindListByNameFull(taskLists, name)
				if err != nil {
					return err
				}
				info := buildListInfo(taskManager, *list)
				listsToShow = append(listsToShow, info)
			}

			// Output in requested format
			if jsonOutput {
				if err := utils.OutputJSON(listsToShow); err != nil {
					return err
				}
			} else if yamlOutput {
				if err := utils.OutputYAML(listsToShow); err != nil {
					return err
				}
			} else {
				// Human-readable format
				for i, item := range listsToShow {
					if i > 0 {
						fmt.Println()
					}
					printListInfo(item.(map[string]interface{}))
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&showAll, "all", false, "Show info for all lists")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&yamlOutput, "yaml", false, "Output in YAML format")

	return cmd
}

// buildListInfo builds a map of list information
func buildListInfo(tm backend.TaskManager, list backend.TaskList) map[string]interface{} {
	listMap := make(map[string]interface{})

	// Basic info
	listMap["name"] = list.Name
	listMap["id"] = list.ID
	listMap["description"] = list.Description
	listMap["url"] = list.URL
	listMap["color"] = list.Color
	listMap["ctag"] = list.CTags

	// Get tasks to count them
	tasks, err := tm.GetTasks(list.ID, nil)
	if err != nil {
		listMap["task_count"] = 0
		listMap["tasks_by_status"] = map[string]int{}
		listMap["error"] = err.Error()
		return listMap
	}

	// Count total and by status
	listMap["task_count"] = len(tasks)
	statusCounts := map[string]int{
		"TODO":       0,
		"DONE":       0,
		"PROCESSING": 0,
		"CANCELLED":  0,
	}

	for _, task := range tasks {
		// Convert backend status to display name
		displayStatus := tm.StatusToDisplayName(task.Status)
		statusCounts[displayStatus]++
	}

	listMap["tasks_by_status"] = statusCounts

	return listMap
}

// printListInfo prints list information in human-readable format
func printListInfo(info map[string]interface{}) {
	fmt.Printf("List: %s\n", info["name"])
	if id, ok := info["id"].(string); ok && id != "" {
		fmt.Printf("ID: %s\n", id)
	}
	if desc, ok := info["description"].(string); ok && desc != "" {
		fmt.Printf("Description: %s\n", desc)
	}

	if count, ok := info["task_count"].(int); ok {
		fmt.Printf("Total tasks: %d\n", count)
	}

	if statusCounts, ok := info["tasks_by_status"].(map[string]int); ok {
		fmt.Println("Tasks by status:")
		for status, count := range statusCounts {
			if count > 0 {
				fmt.Printf("  %s: %d\n", status, count)
			}
		}
	}

	// Print backend-specific info if available
	if url, ok := info["url"].(string); ok && url != "" {
		fmt.Printf("URL: %s\n", url)
	}
	if color, ok := info["color"].(string); ok && color != "" {
		fmt.Printf("Color: %s\n", color)
	}
	if ctag, ok := info["ctag"].(string); ok && ctag != "" {
		fmt.Printf("CTag: %s\n", ctag)
	}
}

// newListTrashCmd creates the 'list trash' command with subcommands
func newListTrashCmd() *cobra.Command {
	trashCmd := &cobra.Command{
		Use:   "trash",
		Short: "Manage deleted task lists (trash)",
		Long: `View and manage task lists that have been deleted (moved to trash).

Subcommands:
  gosynctasks list trash              # Show all deleted lists
  gosynctasks list trash restore <name>  # Restore a deleted list
  gosynctasks list trash empty <name>    # Permanently delete a list from trash
  gosynctasks list trash empty --all     # Empty entire trash (dangerous!)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default action: show deleted lists
			taskManager := application.GetTaskManager()
			if taskManager == nil {
				return fmt.Errorf("task manager not initialized")
			}

			// Get deleted lists
			deletedLists, err := taskManager.GetDeletedTaskLists()
			if err != nil {
				return fmt.Errorf("failed to get deleted lists: %w", err)
			}

			if len(deletedLists) == 0 {
				fmt.Println("No deleted lists in trash.")
				return nil
			}

			fmt.Println("\nDeleted task lists (in trash):")
			for _, list := range deletedLists {
				deletedInfo := ""
				if list.DeletedAt != "" {
					deletedInfo = fmt.Sprintf(" (deleted: %s)", list.DeletedAt)
				}

				if list.Description != "" {
					fmt.Printf("  • %s - %s%s\n", list.Name, list.Description, deletedInfo)
				} else {
					fmt.Printf("  • %s%s\n", list.Name, deletedInfo)
				}
			}
			fmt.Println()
			return nil
		},
	}

	// Add subcommands
	trashCmd.AddCommand(newListTrashRestoreCmd())
	trashCmd.AddCommand(newListTrashEmptyCmd())

	return trashCmd
}

// newListTrashRestoreCmd creates the 'list trash restore' command
func newListTrashRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <name>",
		Short: "Restore a deleted list from trash",
		Long: `Restore a task list from trash, bringing back all its tasks.

The list will be restored to its original state before deletion.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Get remote backend for list operations (bypasses cache when sync enabled)
			taskManager := application.GetRemoteBackend()
			if taskManager == nil {
				return fmt.Errorf("task manager not initialized")
			}

			// Refresh task lists from backend (needed to ensure cache is updated)
			application.RefreshTaskListsFromRemoteOrWarn()

			// Find the list in trash by name
			deletedLists, err := taskManager.GetDeletedTaskLists()
			if err != nil {
				return fmt.Errorf("failed to get deleted lists: %w", err)
			}

			listID, err := operations.FindListByName(deletedLists, name)
			if err != nil {
				return fmt.Errorf("list '%s' not found in trash", name)
			}

			// Restore the list on remote backend
			if err := taskManager.RestoreTaskList(listID); err != nil {
				return fmt.Errorf("failed to restore list: %w", err)
			}

			// If sync is enabled, trigger a sync to update the SQLite cache
			cfg := config.GetConfig()
			if cfg.Sync != nil && cfg.Sync.Enabled {
				explicitBackend, _ := cmd.Root().PersistentFlags().GetString("backend")
				if err := performSync(cfg, explicitBackend, true); err != nil {
					fmt.Printf("Warning: list restored on remote but sync failed: %v\n", err)
				}
			}

			// Clear cache
			application.RefreshTaskListsFromRemoteOrWarn()

			fmt.Printf("List '%s' restored successfully.\n", name)
			return nil
		},
	}

	return cmd
}

// newListTrashEmptyCmd creates the 'list trash empty' command
func newListTrashEmptyCmd() *cobra.Command {
	var emptyAll bool
	var force bool

	cmd := &cobra.Command{
		Use:   "empty [name]",
		Short: "Permanently delete a list from trash",
		Long: `Permanently delete a task list from trash. This operation is irreversible.

By default, prompts for confirmation.
Use --all to empty the entire trash (WARNING: very dangerous!).
Use --force to skip the confirmation prompt.

WARNING: This permanently and irreversibly deletes the list and all its tasks.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get remote backend for list operations (bypasses cache when sync enabled)
			taskManager := application.GetRemoteBackend()
			if taskManager == nil {
				return fmt.Errorf("task manager not initialized")
			}

			// Refresh task lists from backend (needed to ensure cache is updated)
			application.RefreshTaskListsFromRemoteOrWarn()

			// Get deleted lists
			deletedLists, err := taskManager.GetDeletedTaskLists()
			if err != nil {
				return fmt.Errorf("failed to get deleted lists: %w", err)
			}

			if len(deletedLists) == 0 {
				fmt.Println("Trash is already empty.")
				return nil
			}

			// Determine which lists to delete
			var listsToDelete []backend.TaskList

			if emptyAll {
				// Delete all lists in trash
				listsToDelete = deletedLists
			} else {
				// Delete specific list
				if len(args) == 0 {
					return fmt.Errorf("list name required (or use --all)")
				}

				name := args[0]
				list, err := operations.FindListByNameFull(deletedLists, name)
				if err != nil {
					return fmt.Errorf("list '%s' not found in trash", name)
				}
				listsToDelete = append(listsToDelete, *list)
			}

			// Confirm deletion unless --force
			if !force {
				if emptyAll {
					fmt.Printf("This will PERMANENTLY delete ALL %d lists in trash.\n", len(listsToDelete))
					fmt.Println("This operation is IRREVERSIBLE and will delete all tasks in these lists.")
				} else {
					fmt.Printf("This will PERMANENTLY delete the list '%s' from trash.\n", listsToDelete[0].Name)
					fmt.Println("This operation is IRREVERSIBLE and will delete all tasks in this list.")
				}
				confirmed, err := utils.PromptConfirmation("Are you sure?")
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Println("Operation cancelled.")
					return nil
				}
			}

			// Delete the lists from remote backend
			deletedCount := 0
			for _, list := range listsToDelete {
				if err := taskManager.PermanentlyDeleteTaskList(list.ID); err != nil {
					fmt.Printf("Warning: failed to delete '%s': %v\n", list.Name, err)
					continue
				}
				deletedCount++
			}

			// Clear cache
			application.RefreshTaskListsFromRemoteOrWarn()

			if emptyAll {
				fmt.Printf("Successfully permanently deleted %d lists from trash.\n", deletedCount)
			} else {
				fmt.Printf("List '%s' permanently deleted from trash.\n", listsToDelete[0].Name)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&emptyAll, "all", false, "Empty entire trash (delete all lists permanently)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}
