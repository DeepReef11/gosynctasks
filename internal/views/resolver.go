package views

import (
	"embed"
	"fmt"
	"path/filepath"
	"sync"
)

//go:embed builtin_views/*.yaml
var builtinViewFS embed.FS

// viewCache stores loaded views for performance
var viewCache = make(map[string]*View)
var cacheMutex sync.RWMutex

// ResolveView loads a view by name with the following priority:
// 1. User views (~/.config/gosynctasks/views/<name>.yaml)
// 2. Built-in views (basic, all, minimal, full, kanban, timeline, compact)
//
// Views are cached after first load for performance.
func ResolveView(name string) (*View, error) {
	// Check cache first
	cacheMutex.RLock()
	if cached, ok := viewCache[name]; ok {
		cacheMutex.RUnlock()
		return cached, nil
	}
	cacheMutex.RUnlock()

	// Try to load user view first
	viewsDir, err := GetViewsDir()
	if err == nil {
		// Try .yaml extension first, then .yml
		for _, ext := range []string{".yaml", ".yml"} {
			filePath := filepath.Join(viewsDir, name+ext)
			view, err := LoadView(filePath)
			if err == nil {
				// Cache the view
				cacheMutex.Lock()
				viewCache[name] = view
				cacheMutex.Unlock()
				return view, nil
			}
		}
	}

	// Fall back to built-in views
	view, err := getBuiltInView(name)
	if err != nil {
		return nil, fmt.Errorf("view '%s' not found (checked user views and built-in views)", name)
	}

	// Cache the built-in view
	cacheMutex.Lock()
	viewCache[name] = view
	cacheMutex.Unlock()

	return view, nil
}

// ClearViewCache clears the view cache (useful for testing or after view updates)
func ClearViewCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	viewCache = make(map[string]*View)
}

// InvalidateViewCache removes a specific view from the cache
func InvalidateViewCache(name string) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	delete(viewCache, name)
}

// getBuiltInView returns a built-in view by name from embedded YAML files
func getBuiltInView(name string) (*View, error) {
	// Try to read the embedded YAML file
	filePath := fmt.Sprintf("builtin_views/%s.yaml", name)
	data, err := builtinViewFS.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("built-in view '%s' not found", name)
	}

	// Load view from YAML bytes
	view, err := LoadViewFromBytes(data, name)
	if err != nil {
		return nil, fmt.Errorf("failed to load built-in view '%s': %w", name, err)
	}

	return view, nil
}

// GetBuiltInViews returns a list of built-in view names
func GetBuiltInViews() []string {
	return []string{"default", "all", "minimal", "full", "kanban", "timeline", "compact"}
}

// IsBuiltInView checks if a view name is a built-in view
func IsBuiltInView(name string) bool {
	builtInViews := GetBuiltInViews()
	for _, builtIn := range builtInViews {
		if name == builtIn {
			return true
		}
	}
	return false
}
