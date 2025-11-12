package views

import (
	"fmt"
	"path/filepath"
	"sync"
)

// viewCache stores loaded views for performance
var viewCache = make(map[string]*View)
var cacheMutex sync.RWMutex

// ResolveView loads a view by name with the following priority:
// 1. User views (~/.config/gosynctasks/views/<name>.yaml)
// 2. Built-in views (basic, all)
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

// getBuiltInView returns a built-in view by name
func getBuiltInView(name string) (*View, error) {
	switch name {
	case "basic":
		return &View{
			Name:        "basic",
			Description: "Basic view showing status, summary, and dates",
			Fields: []FieldConfig{
				{Name: "status", Format: "symbol", Show: true},
				{Name: "summary", Format: "full", Show: true},
				{Name: "start_date", Format: "full", Show: true, Color: true},
				{Name: "due_date", Format: "full", Show: true, Color: true},
				{Name: "description", Format: "truncate", Width: 70, Show: true},
			},
			FieldOrder: []string{"status", "summary", "start_date", "due_date", "description"},
			Display: DisplayOptions{
				ShowHeader:  true,
				ShowBorder:  true,
				CompactMode: false,
				DateFormat:  "2006-01-02",
			},
		}, nil

	case "all":
		return &View{
			Name:        "all",
			Description: "Complete view showing all task metadata",
			Fields: []FieldConfig{
				{Name: "status", Format: "symbol", Show: true},
				{Name: "summary", Format: "full", Show: true},
				{Name: "start_date", Format: "full", Show: true, Color: true},
				{Name: "due_date", Format: "full", Show: true, Color: true},
				{Name: "description", Format: "truncate", Width: 70, Show: true},
				{Name: "created", Format: "full", Show: true},
				{Name: "modified", Format: "full", Show: true},
				{Name: "priority", Format: "number", Show: true, Color: true},
			},
			FieldOrder: []string{"status", "summary", "start_date", "due_date", "description", "created", "modified", "priority"},
			Display: DisplayOptions{
				ShowHeader:  true,
				ShowBorder:  true,
				CompactMode: false,
				DateFormat:  "2006-01-02",
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown built-in view: %s", name)
	}
}

// GetBuiltInViews returns a list of built-in view names
func GetBuiltInViews() []string {
	return []string{"basic", "all"}
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
