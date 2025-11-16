package main

import (
	"fmt"
	"gosynctasks/internal/utils"
	"gosynctasks/internal/views"
	"gosynctasks/internal/views/builder"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// newViewCmd creates the view management command with all subcommands
func newViewCmd() *cobra.Command {
	viewCmd := &cobra.Command{
		Use:   "view",
		Short: "Manage custom views",
		Long: `Manage custom view configurations for task display.

Views control how tasks are formatted and displayed, including which fields
to show, their order, formatting options, and colors.

Examples:
  gosynctasks view list                    # List all views
  gosynctasks view show minimal            # Show view configuration
  gosynctasks view create myview           # Create from editor
  gosynctasks view create myview --template minimal  # Create from template
  gosynctasks view edit myview             # Edit in $EDITOR
  gosynctasks view delete myview           # Delete user view
  gosynctasks view copy minimal custom     # Copy view
  gosynctasks view validate myview         # Validate configuration`,
	}

	// Add subcommands
	viewCmd.AddCommand(newViewListCmd())
	viewCmd.AddCommand(newViewShowCmd())
	viewCmd.AddCommand(newViewCreateCmd())
	viewCmd.AddCommand(newViewEditCmd())
	viewCmd.AddCommand(newViewDeleteCmd())
	viewCmd.AddCommand(newViewCopyCmd())
	viewCmd.AddCommand(newViewValidateCmd())

	return viewCmd
}

// newViewListCmd creates the 'view list' command
func newViewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available views",
		Long:  "List all available views including built-in and user-created views.",
		RunE: func(cmd *cobra.Command, args []string) error {
			viewNames, err := views.ListViews()
			if err != nil {
				return fmt.Errorf("failed to list views: %w", err)
			}

			if len(viewNames) == 0 {
				fmt.Println("No views found.")
				return nil
			}

			fmt.Println("Available views:")
			fmt.Println()

			for _, name := range viewNames {
				// Try to load view to get description
				view, err := views.ResolveView(name)
				if err != nil {
					fmt.Printf("  %-20s (error loading)\n", name)
					continue
				}

				// Mark built-in views
				marker := ""
				if views.IsBuiltInView(name) {
					marker = " [built-in]"
				}

				desc := view.Description
				if desc == "" {
					desc = "No description"
				}

				fmt.Printf("  %-20s %s%s\n", name, desc, marker)
			}

			return nil
		},
	}
}

// newViewShowCmd creates the 'view show' command
func newViewShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <view-name>",
		Short: "Show view configuration",
		Long:  "Display the complete configuration of a view including fields, order, and display options.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			viewName := args[0]

			view, err := views.ResolveView(viewName)
			if err != nil {
				return fmt.Errorf("view '%s' not found: %w", viewName, err)
			}

			// Marshal to YAML for display
			data, err := yaml.Marshal(view)
			if err != nil {
				return fmt.Errorf("failed to format view: %w", err)
			}

			fmt.Printf("View: %s\n", view.Name)
			if views.IsBuiltInView(viewName) {
				fmt.Println("Type: Built-in")
			} else {
				fmt.Println("Type: User-defined")
			}
			fmt.Println()
			fmt.Println(string(data))

			return nil
		},
	}
}

// newViewCreateCmd creates the 'view create' command
func newViewCreateCmd() *cobra.Command {
	var templateName string
	var interactive bool

	cmd := &cobra.Command{
		Use:   "create <view-name>",
		Short: "Create a new view",
		Long: `Create a new view configuration.

By default, opens your editor ($EDITOR) to create the view.
Use --template to create from a built-in template.
Use --interactive to use the interactive builder.

Available templates:
  minimal  - Minimalist view (status, summary, due date)
  full     - Complete view with all fields
  kanban   - Kanban-style view
  timeline - Timeline view focused on dates
  compact  - Single-line compact view`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			viewName := args[0]

			// Check if view already exists
			if views.ViewExists(viewName) {
				return fmt.Errorf("view '%s' already exists (use 'edit' to modify)", viewName)
			}

			var view *views.View

			if interactive {
				// Use interactive builder
				built, err := builder.Run(viewName)
				if err != nil {
					return err
				}
				view = built
			} else if templateName != "" {
				// Create from template
				template, err := getViewTemplate(templateName)
				if err != nil {
					return err
				}

				// Update name
				template.Name = viewName
				view = template
			} else {
				// Create from editor
				trueVal := true
				falseVal := false
				view = &views.View{
					Name:        viewName,
					Description: "New view",
					Fields: []views.FieldConfig{
						{Name: "summary", Format: "full", Show: &trueVal},
						{Name: "description", Format: "full", Show: &falseVal, Width: 70},
						{Name: "status", Format: "symbol", Show: &trueVal, Color: true},
						{Name: "priority", Format: "number", Show: &trueVal, Color: true},
						{Name: "due_date", Format: "relative", Show: &trueVal, Color: true},
						{Name: "start_date", Format: "short", Show: &falseVal, Color: true},
						{Name: "created", Format: "relative", Show: &falseVal},
						{Name: "modified", Format: "relative", Show: &falseVal},
						{Name: "completed", Format: "short", Show: &falseVal},
						{Name: "tags", Format: "comma", Show: &falseVal},
					},
					Filters: &views.ViewFilters{
						Status: []string{"NEEDS-ACTION", "IN-PROCESS"},
					},
					Display: views.DisplayOptions{
						ShowHeader:  true,
						ShowBorder:  true,
						CompactMode: false,
						DateFormat:  "2006-01-02",
						SortBy:      "priority",
						SortOrder:   "asc",
					},
				}

				// Open in editor
				edited, err := editViewInEditor(view)
				if err != nil {
					return err
				}
				view = edited
			}

			// Save view
			if err := views.SaveView(view); err != nil {
				return fmt.Errorf("failed to save view: %w", err)
			}

			fmt.Printf("View '%s' created successfully.\n", viewName)
			return nil
		},
	}

	cmd.Flags().StringVarP(&templateName, "template", "t", "", "Create from template (minimal, full, kanban, timeline, compact)")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Use interactive builder")

	return cmd
}

// newViewEditCmd creates the 'view edit' command
func newViewEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <view-name>",
		Short: "Edit a view",
		Long:  "Edit a view configuration in your editor ($EDITOR).\nBuilt-in views cannot be edited.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			viewName := args[0]

			// Prevent editing built-in views
			if views.IsBuiltInView(viewName) {
				return fmt.Errorf("cannot edit built-in view '%s' (use 'copy' to create an editable version)", viewName)
			}

			// Load view
			view, err := views.ResolveView(viewName)
			if err != nil {
				return fmt.Errorf("view '%s' not found: %w", viewName, err)
			}

			// Edit in editor
			edited, err := editViewInEditor(view)
			if err != nil {
				return err
			}

			// Save
			if err := views.SaveView(edited); err != nil {
				return fmt.Errorf("failed to save view: %w", err)
			}

			// Clear cache
			views.InvalidateViewCache(viewName)

			fmt.Printf("View '%s' updated successfully.\n", viewName)
			return nil
		},
	}
}

// newViewDeleteCmd creates the 'view delete' command
func newViewDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <view-name>",
		Short: "Delete a view",
		Long:  "Delete a user-defined view.\nBuilt-in views cannot be deleted.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			viewName := args[0]

			// Confirm deletion unless --force
			if !force {
				confirmed, err := utils.PromptConfirmation(fmt.Sprintf("Delete view '%s'?", viewName))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Println("Deletion cancelled.")
					return nil
				}
			}

			// Delete view
			if err := views.DeleteView(viewName); err != nil {
				return err
			}

			// Clear cache
			views.InvalidateViewCache(viewName)

			fmt.Printf("View '%s' deleted successfully.\n", viewName)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

// newViewCopyCmd creates the 'view copy' command
func newViewCopyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "copy <source> <destination>",
		Short: "Copy a view",
		Long:  "Create a copy of an existing view with a new name.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceName := args[0]
			destName := args[1]

			// Check if destination exists
			if views.ViewExists(destName) {
				return fmt.Errorf("view '%s' already exists", destName)
			}

			// Load source view
			sourceView, err := views.ResolveView(sourceName)
			if err != nil {
				return fmt.Errorf("source view '%s' not found: %w", sourceName, err)
			}

			// Create copy with new name
			destView := *sourceView
			destView.Name = destName

			// Save
			if err := views.SaveView(&destView); err != nil {
				return fmt.Errorf("failed to save view: %w", err)
			}

			fmt.Printf("View '%s' copied to '%s' successfully.\n", sourceName, destName)
			return nil
		},
	}
}

// newViewValidateCmd creates the 'view validate' command
func newViewValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <view-name>",
		Short: "Validate a view",
		Long:  "Check if a view configuration is valid and can be loaded.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			viewName := args[0]

			// Try to load view
			view, err := views.ResolveView(viewName)
			if err != nil {
				fmt.Printf("❌ View '%s' is INVALID:\n", viewName)
				fmt.Printf("   %v\n", err)
				return nil // Don't return error, we want to show validation result
			}

			fmt.Printf("✓ View '%s' is valid\n", viewName)
			fmt.Printf("  Name: %s\n", view.Name)
			fmt.Printf("  Fields: %d\n", len(view.Fields))
			if view.Description != "" {
				fmt.Printf("  Description: %s\n", view.Description)
			}

			return nil
		},
	}
}

// editViewInEditor opens a view in the user's editor with validation loop
func editViewInEditor(view *views.View) (*views.View, error) {
	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default to vi
	}

	// Create temp file
	tmpfile, err := os.CreateTemp("", "gosynctasks-view-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpfile.Name())

	// Marshal initial view to YAML
	data, err := yaml.Marshal(view)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal view: %w", err)
	}

	currentContent := data

	// Validation loop
	for {
		// Write current content to temp file
		if err := os.WriteFile(tmpfile.Name(), currentContent, 0644); err != nil {
			return nil, fmt.Errorf("failed to write temp file: %w", err)
		}

		// Open in editor
		cmd := exec.Command(editor, tmpfile.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("editor failed: %w", err)
		}

		// Read edited content
		editedData, err := os.ReadFile(tmpfile.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read edited file: %w", err)
		}

		// Try to parse the YAML first
		var parsedView views.View
		if err := yaml.Unmarshal(editedData, &parsedView); err != nil {
			fmt.Printf("\n❌ YAML parsing failed: %v\n\n", err)
			fmt.Print("The YAML syntax is invalid. Would you like to:\n")
			fmt.Print("  [r] Retry - reopen editor to fix errors\n")
			fmt.Print("  [c] Cancel - discard changes\n")
			fmt.Print("Choice (r/c): ")

			var choice string
			if _, err := fmt.Scanf("%s", &choice); err != nil {
				return nil, fmt.Errorf("failed to read choice: %w", err)
			}

			choice = strings.ToLower(strings.TrimSpace(choice))
			if choice == "c" || choice == "cancel" {
				return nil, fmt.Errorf("edit cancelled by user")
			}

			// User wants to retry - keep current content with YAML error at top
			errorComment := fmt.Sprintf("# YAML SYNTAX ERROR:\n# %s\n# Please fix the YAML syntax above and try again.\n\n", err.Error())
			currentContent = append([]byte(errorComment), editedData...)
			continue
		}

		// Set the view name if not in YAML
		if parsedView.Name == "" {
			parsedView.Name = view.Name
		}

		// Perform comprehensive validation
		validationErrors := views.ValidateViewComprehensive(&parsedView)

		if validationErrors != nil && len(validationErrors.Errors) > 0 {
			// Validation failed - show errors
			fmt.Printf("\n❌ Validation failed with %d error(s):\n", len(validationErrors.Errors))
			for i, err := range validationErrors.Errors {
				fmt.Printf("  %d. %s: %s\n", i+1, err.Field, err.Message)
				if err.Hint != "" {
					fmt.Printf("     Hint: %s\n", err.Hint)
				}
			}
			fmt.Println()

			// Ask user what to do
			fmt.Print("Would you like to:\n")
			fmt.Print("  [r] Retry - reopen editor with inline error comments\n")
			fmt.Print("  [c] Cancel - discard changes\n")
			fmt.Print("Choice (r/c): ")

			var choice string
			if _, err := fmt.Scanf("%s", &choice); err != nil {
				return nil, fmt.Errorf("failed to read choice: %w", err)
			}

			choice = strings.ToLower(strings.TrimSpace(choice))
			if choice == "c" || choice == "cancel" {
				return nil, fmt.Errorf("edit cancelled by user")
			}

			// User wants to retry - annotate YAML with errors
			annotatedYAML := views.AnnotateYAMLWithErrors(string(editedData), validationErrors)
			currentContent = []byte(annotatedYAML)
			continue
		}

		// Validation passed! Apply defaults and return
		edited, err := views.LoadViewFromBytes(editedData, view.Name)
		if err != nil {
			// This shouldn't happen since we already validated, but handle it
			return nil, fmt.Errorf("failed to load view: %w", err)
		}

		return edited, nil
	}
}

// getViewTemplate returns a built-in view template
func getViewTemplate(name string) (*views.View, error) {
	trueVal := true

	switch name {
	case "minimal":
		return &views.View{
			Name:        "minimal",
			Description: "Minimalist view showing only essential information",
			Fields: []views.FieldConfig{
				{Name: "status", Format: "symbol", Show: &trueVal},
				{Name: "summary", Format: "full", Show: &trueVal},
				{Name: "due_date", Format: "short", Color: true, Show: &trueVal},
			},
			Display: views.DisplayOptions{
				ShowHeader:  true,
				ShowBorder:  false,
				CompactMode: true,
				DateFormat:  "01/02",
			},
		}, nil

	case "full":
		return &views.View{
			Name:        "full",
			Description: "Comprehensive view with all task metadata",
			Fields: []views.FieldConfig{
				{Name: "status", Format: "text", Show: &trueVal},
				{Name: "priority", Format: "stars", Color: true, Show: &trueVal},
				{Name: "summary", Format: "full", Show: &trueVal},
				{Name: "description", Format: "first_line", Width: 80, Show: &trueVal},
				{Name: "start_date", Format: "full", Color: true, Show: &trueVal},
				{Name: "due_date", Format: "full", Color: true, Show: &trueVal},
				{Name: "tags", Format: "hash", Show: &trueVal},
				{Name: "created", Format: "relative", Show: &trueVal},
				{Name: "modified", Format: "relative", Show: &trueVal},
				{Name: "uid", Format: "short", Show: &trueVal},
			},
			FieldOrder: []string{"status", "priority", "summary", "description", "start_date", "due_date", "tags", "created", "modified", "uid"},
			Display: views.DisplayOptions{
				ShowHeader:  true,
				ShowBorder:  true,
				CompactMode: false,
				DateFormat:  "2006-01-02 15:04",
				SortBy:      "priority",
				SortOrder:   "asc",
			},
		}, nil

	case "kanban":
		return &views.View{
			Name:        "kanban",
			Description: "Kanban-style view grouped by status",
			Fields: []views.FieldConfig{
				{Name: "status", Format: "emoji", Show: &trueVal},
				{Name: "summary", Format: "truncate", Width: 50, Show: &trueVal},
				{Name: "priority", Format: "color", Color: true, Show: &trueVal},
				{Name: "due_date", Format: "relative", Color: true, Show: &trueVal},
				{Name: "tags", Format: "comma", Show: &trueVal},
			},
			FieldOrder: []string{"status", "priority", "summary", "due_date", "tags"},
			Display: views.DisplayOptions{
				ShowHeader:  true,
				ShowBorder:  true,
				CompactMode: true,
				DateFormat:  "Jan 02",
				SortBy:      "status",
				SortOrder:   "asc",
			},
		}, nil

	case "timeline":
		return &views.View{
			Name:        "timeline",
			Description: "Timeline view focusing on dates and scheduling",
			Fields: []views.FieldConfig{
				{Name: "start_date", Format: "full", Color: true, Label: "Starts", Show: &trueVal},
				{Name: "due_date", Format: "full", Color: true, Label: "Due", Show: &trueVal},
				{Name: "status", Format: "short", Show: &trueVal},
				{Name: "summary", Format: "full", Show: &trueVal},
				{Name: "priority", Format: "number", Color: true, Show: &trueVal},
				{Name: "description", Format: "truncate", Width: 60, Show: &trueVal},
			},
			FieldOrder: []string{"start_date", "due_date", "status", "priority", "summary", "description"},
			Display: views.DisplayOptions{
				ShowHeader:  true,
				ShowBorder:  true,
				CompactMode: false,
				DateFormat:  "Mon 01/02",
				SortBy:      "start_date",
				SortOrder:   "asc",
			},
		}, nil

	case "compact":
		return &views.View{
			Name:        "compact",
			Description: "Single-line compact view",
			Fields: []views.FieldConfig{
				{Name: "status", Format: "short", Show: &trueVal},
				{Name: "priority", Format: "number", Show: &trueVal},
				{Name: "summary", Format: "truncate", Width: 40, Show: &trueVal},
				{Name: "due_date", Format: "short", Color: true, Show: &trueVal},
			},
			FieldOrder: []string{"status", "priority", "summary", "due_date"},
			Display: views.DisplayOptions{
				ShowHeader:  false,
				ShowBorder:  false,
				CompactMode: true,
				DateFormat:  "01/02",
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown template: %s (available: minimal, full, kanban, timeline, compact)", name)
	}
}
