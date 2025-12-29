package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"slices"
)

// GetViewsDir returns the directory where view configurations are stored
// Default: ~/.config/gosynctasks/views/
func GetViewsDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	viewsDir := filepath.Join(configDir, "gosynctasks", "views")
	return viewsDir, nil
}

// EnsureViewsDir creates the views directory if it doesn't exist
func EnsureViewsDir() error {
	viewsDir, err := GetViewsDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(viewsDir, 0755); err != nil {
		return fmt.Errorf("failed to create views directory: %w", err)
	}

	return nil
}

// ListViews returns a list of all available view names (user + built-in)
// User views take precedence over built-in views with the same name
func ListViews() ([]string, error) {
	viewsMap := make(map[string]bool)

	// Get built-in views
	builtInViews := []string{"default", "all"}
	for _, name := range builtInViews {
		viewsMap[name] = true
	}

	// Get user views
	viewsDir, err := GetViewsDir()
	if err != nil {
		return nil, err
	}

	// Check if directory exists
	if _, err := os.Stat(viewsDir); os.IsNotExist(err) {
		// Directory doesn't exist yet, just return built-in views
		return builtInViews, nil
	}

	// Read directory
	entries, err := os.ReadDir(viewsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read views directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Only include .yaml and .yml files
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			// Remove extension
			viewName := name[:len(name)-len(filepath.Ext(name))]
			viewsMap[viewName] = true
		}
	}

	// Convert map to sorted list
	views := make([]string, 0, len(viewsMap))
	for name := range viewsMap {
		views = append(views, name)
	}

	return views, nil
}

// SaveView saves a view configuration to disk
func SaveView(view *View) error {
	if err := EnsureViewsDir(); err != nil {
		return err
	}

	viewsDir, err := GetViewsDir()
	if err != nil {
		return err
	}

	// Validate view before saving
	if err := validate.Struct(view); err != nil {
		return fmt.Errorf("validation failed: %w", formatValidationError(err))
	}

	// Marshal to YAML
	data, err := yaml.Marshal(view)
	if err != nil {
		return fmt.Errorf("failed to marshal view to YAML: %w", err)
	}

	// Write to file
	filePath := filepath.Join(viewsDir, view.Name+".yaml")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write view file: %w", err)
	}

	return nil
}

// DeleteView deletes a user view configuration
// Built-in views cannot be deleted
func DeleteView(name string) error {
	// Prevent deletion of built-in views
	builtInViews := []string{"default", "all"}
	if slices.Contains(builtInViews, name) {
			return fmt.Errorf("cannot delete built-in view '%s'", name)
		}

	viewsDir, err := GetViewsDir()
	if err != nil {
		return err
	}

	// Try both .yaml and .yml extensions
	for _, ext := range []string{".yaml", ".yml"} {
		filePath := filepath.Join(viewsDir, name+ext)
		if _, err := os.Stat(filePath); err == nil {
			if err := os.Remove(filePath); err != nil {
				return fmt.Errorf("failed to delete view file: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("view '%s' not found", name)
}

// ViewExists checks if a view exists (user or built-in)
func ViewExists(name string) bool {
	// Check built-in views
	builtInViews := []string{"default", "all"}
	for _, builtIn := range builtInViews {
		if name == builtIn {
			return true
		}
	}

	// Check user views
	viewsDir, err := GetViewsDir()
	if err != nil {
		return false
	}

	// Try both .yaml and .yml extensions
	for _, ext := range []string{".yaml", ".yml"} {
		filePath := filepath.Join(viewsDir, name+ext)
		if _, err := os.Stat(filePath); err == nil {
			return true
		}
	}

	return false
}

// CopyBuiltInViewsToUserConfig copies all built-in views to the user's config directory
// This is typically called on first run to allow users to customize views
// Returns true if views were copied, false if they already existed
func CopyBuiltInViewsToUserConfig() (bool, error) {
	viewsDir, err := GetViewsDir()
	if err != nil {
		return false, fmt.Errorf("failed to get views directory: %w", err)
	}

	// Check if views directory exists and has any .yaml files
	if info, err := os.Stat(viewsDir); err == nil && info.IsDir() {
		entries, err := os.ReadDir(viewsDir)
		if err == nil && len(entries) > 0 {
			// Check if there are any .yaml or .yml files
			for _, entry := range entries {
				if !entry.IsDir() {
					name := entry.Name()
					if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
						// Views already exist, skip copying
						return false, nil
					}
				}
			}
		}
	}

	// Ensure views directory exists
	if err := EnsureViewsDir(); err != nil {
		return false, fmt.Errorf("failed to create views directory: %w", err)
	}

	// Copy each built-in view
	builtInViews := GetBuiltInViews()
	for _, viewName := range builtInViews {
		// Read built-in view from embedded FS
		filePath := fmt.Sprintf("builtin_views/%s.yaml", viewName)
		data, err := builtinViewFS.ReadFile(filePath)
		if err != nil {
			return false, fmt.Errorf("failed to read built-in view '%s': %w", viewName, err)
		}

		// Write to user's views directory
		destPath := filepath.Join(viewsDir, viewName+".yaml")
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return false, fmt.Errorf("failed to write view '%s' to %s: %w", viewName, destPath, err)
		}
	}

	return true, nil
}
