package main

import (
	"encoding/json"
	"fmt"
	"gosynctasks/backend"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
  gosynctasks list create "Work Tasks"                  # Create new list
  gosynctasks list create "Project" -d "Tasks for XYZ"  # With description
  gosynctasks list create "Urgent" --color "#ff0000"    # With color (Nextcloud)

  gosynctasks list delete "Old Tasks"                   # Delete list (with confirmation)
  gosynctasks list delete "Archive" --force             # Skip confirmation

  gosynctasks list rename "Old Name" "New Name"         # Rename list

  gosynctasks list info "Work Tasks"                    # Show list details
  gosynctasks list info --all                           # Show all lists
  gosynctasks list info "Work Tasks" --json             # JSON output`,
	}

	// Add subcommands
	listCmd.AddCommand(newListCreateCmd())
	listCmd.AddCommand(newListDeleteCmd())
	listCmd.AddCommand(newListRenameCmd())
	listCmd.AddCommand(newListInfoCmd())

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

			// Get task manager from application
			taskManager := application.GetTaskManager()
			if taskManager == nil {
				return fmt.Errorf("task manager not initialized")
			}

			// Create the list
			listID, err := taskManager.CreateTaskList(name, description, color)
			if err != nil {
				return fmt.Errorf("failed to create list: %w", err)
			}

			// Clear cache
			if err := application.RefreshTaskLists(); err != nil {
				// Non-fatal, just warn
				fmt.Printf("Warning: failed to refresh cache: %v\n", err)
			}

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

			// Get task manager from application
			taskManager := application.GetTaskManager()
			if taskManager == nil {
				return fmt.Errorf("task manager not initialized")
			}

			// Find the list by name
			taskLists := application.GetTaskLists()
			var listID string
			var taskCount int
			for _, list := range taskLists {
				if list.Name == name {
					listID = list.ID
					// Get task count
					tasks, err := taskManager.GetTasks(list.ID, nil)
					if err == nil {
						taskCount = len(tasks)
					}
					break
				}
			}

			if listID == "" {
				return fmt.Errorf("list '%s' not found", name)
			}

			// Confirm deletion unless --force
			if !force {
				fmt.Printf("This will delete the list '%s' and all %d tasks in it.\n", name, taskCount)
				fmt.Print("Are you sure? (y/n): ")
				var response string
				if _, err := fmt.Scanf("%s", &response); err != nil {
					return fmt.Errorf("invalid input: %w", err)
				}

				response = strings.ToLower(strings.TrimSpace(response))
				if response != "y" && response != "yes" {
					fmt.Println("Deletion cancelled.")
					return nil
				}
			}

			// Delete the list
			if err := taskManager.DeleteTaskList(listID); err != nil {
				return fmt.Errorf("failed to delete list: %w", err)
			}

			// Clear cache
			if err := application.RefreshTaskLists(); err != nil {
				// Non-fatal, just warn
				fmt.Printf("Warning: failed to refresh cache: %v\n", err)
			}

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

			// Get task manager from application
			taskManager := application.GetTaskManager()
			if taskManager == nil {
				return fmt.Errorf("task manager not initialized")
			}

			// Find the old list by name
			taskLists := application.GetTaskLists()
			var listID string
			for _, list := range taskLists {
				if list.Name == oldName {
					listID = list.ID
					break
				}
			}

			if listID == "" {
				return fmt.Errorf("list '%s' not found", oldName)
			}

			// Check if new name already exists
			for _, list := range taskLists {
				if list.Name == newName {
					return fmt.Errorf("list '%s' already exists", newName)
				}
			}

			// Confirm rename unless --force
			if !force {
				fmt.Printf("Rename list '%s' to '%s'? (y/n): ", oldName, newName)
				var response string
				if _, err := fmt.Scanf("%s", &response); err != nil {
					return fmt.Errorf("invalid input: %w", err)
				}

				response = strings.ToLower(strings.TrimSpace(response))
				if response != "y" && response != "yes" {
					fmt.Println("Rename cancelled.")
					return nil
				}
			}

			// Rename the list
			if err := taskManager.RenameTaskList(listID, newName); err != nil {
				return fmt.Errorf("failed to rename list: %w", err)
			}

			// Clear cache
			if err := application.RefreshTaskLists(); err != nil {
				// Non-fatal, just warn
				fmt.Printf("Warning: failed to refresh cache: %v\n", err)
			}

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
				var found bool
				for _, list := range taskLists {
					if list.Name == name {
						info := buildListInfo(taskManager, list)
						listsToShow = append(listsToShow, info)
						found = true
						break
					}
				}

				if !found {
					return fmt.Errorf("list '%s' not found", name)
				}
			}

			// Output in requested format
			if jsonOutput {
				data, err := json.MarshalIndent(listsToShow, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal JSON: %w", err)
				}
				fmt.Println(string(data))
			} else if yamlOutput {
				data, err := yaml.Marshal(listsToShow)
				if err != nil {
					return fmt.Errorf("failed to marshal YAML: %w", err)
				}
				fmt.Print(string(data))
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
