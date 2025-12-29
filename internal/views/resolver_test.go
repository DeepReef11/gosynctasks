package views

import (
	"strings"
	"testing"
)

func TestResolveView_BuiltIn(t *testing.T) {
	// Clear cache before test
	ClearViewCache()

	// Test resolving built-in view
	view, err := ResolveView("default")
	if err != nil {
		t.Fatalf("Failed to resolve built-in view 'default': %v", err)
	}

	if view.Name != "default" {
		t.Errorf("Expected view name 'default', got '%s'", view.Name)
	}

	if len(view.Fields) == 0 {
		t.Error("Expected default view to have fields")
	}
}

func TestResolveView_AllBuiltIn(t *testing.T) {
	ClearViewCache()

	view, err := ResolveView("all")
	if err != nil {
		t.Fatalf("Failed to resolve built-in view 'all': %v", err)
	}

	if view.Name != "all" {
		t.Errorf("Expected view name 'all', got '%s'", view.Name)
	}

	// 'all' view should have more fields than 'default'
	defaultView, _ := ResolveView("default")
	if len(view.Fields) <= len(defaultView.Fields) {
		t.Errorf("Expected 'all' view to have more fields than 'default', got %d vs %d",
			len(view.Fields), len(defaultView.Fields))
	}
}

func TestResolveView_NotFound(t *testing.T) {
	ClearViewCache()

	_, err := ResolveView("non_existent_view")
	if err == nil {
		t.Error("Expected error for non-existent view, got nil")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestResolveView_Caching(t *testing.T) {
	ClearViewCache()

	// First resolve
	view1, err := ResolveView("default")
	if err != nil {
		t.Fatalf("Failed to resolve view: %v", err)
	}

	// Second resolve (should come from cache)
	view2, err := ResolveView("default")
	if err != nil {
		t.Fatalf("Failed to resolve view from cache: %v", err)
	}

	// Should be the same pointer (from cache)
	if view1 != view2 {
		t.Error("Expected cached view to return same pointer")
	}
}

func TestClearViewCache(t *testing.T) {
	// Load a view to populate cache
	view1, err := ResolveView("default")
	if err != nil {
		t.Fatalf("Failed to resolve view: %v", err)
	}

	// Clear cache
	ClearViewCache()

	// Resolve again
	view2, err := ResolveView("default")
	if err != nil {
		t.Fatalf("Failed to resolve view after cache clear: %v", err)
	}

	// Should be different pointers (cache was cleared)
	if view1 == view2 {
		t.Error("Expected different pointers after cache clear")
	}
}

func TestInvalidateViewCache(t *testing.T) {
	ClearViewCache()

	// Load two views
	default1, err := ResolveView("default")
	if err != nil {
		t.Fatalf("Failed to resolve default: %v", err)
	}

	all1, err := ResolveView("all")
	if err != nil {
		t.Fatalf("Failed to resolve all: %v", err)
	}

	// Invalidate only default
	InvalidateViewCache("default")

	// Resolve both again
	default2, _ := ResolveView("default")
	all2, _ := ResolveView("all")

	// default should be different (invalidated), all should be same (cached)
	if default1 == default2 {
		t.Error("Expected different pointer for invalidated view")
	}

	if all1 != all2 {
		t.Error("Expected same pointer for non-invalidated view")
	}
}

func TestGetBuiltInViews(t *testing.T) {
	views := GetBuiltInViews()

	// Should have 7 built-in views
	expectedCount := 7
	if len(views) != expectedCount {
		t.Errorf("Expected %d built-in views, got %d", expectedCount, len(views))
	}

	// Check for all expected built-in views
	expectedViews := []string{"default", "all", "minimal", "full", "kanban", "timeline", "compact"}
	for _, expected := range expectedViews {
		found := false
		for _, name := range views {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected '%s' in built-in views list", expected)
		}
	}
}

func TestIsBuiltInView(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"default", true},
		{"all", true},
		{"minimal", true},
		{"full", true},
		{"kanban", true},
		{"timeline", true},
		{"compact", true},
		{"custom", false},
		{"nonexistent", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBuiltInView(tt.name)
			if result != tt.expected {
				t.Errorf("IsBuiltInView(%q) = %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestGetBuiltInView(t *testing.T) {
	// Test default view structure
	view, err := getBuiltInView("default")
	if err != nil {
		t.Fatalf("Failed to get built-in 'default' view: %v", err)
	}

	if view.Name != "default" {
		t.Errorf("Expected name 'default', got '%s'", view.Name)
	}

	if view.Description == "" {
		t.Error("Expected description to be set")
	}

	if len(view.Fields) == 0 {
		t.Error("Expected fields to be populated")
	}

	// Check display options are set
	if view.Display.DateFormat == "" {
		t.Error("Expected date format to be set")
	}
}

func TestGetBuiltInView_Invalid(t *testing.T) {
	_, err := getBuiltInView("invalid_builtin")
	if err == nil {
		t.Error("Expected error for invalid built-in view name")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

// TestAllBuiltInViewsLoadable verifies that all built-in views can be loaded from embedded YAML
func TestAllBuiltInViewsLoadable(t *testing.T) {
	ClearViewCache()

	builtInViews := GetBuiltInViews()
	for _, viewName := range builtInViews {
		t.Run(viewName, func(t *testing.T) {
			view, err := getBuiltInView(viewName)
			if err != nil {
				t.Fatalf("Failed to load built-in view '%s': %v", viewName, err)
			}

			// Verify basic structure
			if view.Name != viewName {
				t.Errorf("Expected view name '%s', got '%s'", viewName, view.Name)
			}

			if view.Description == "" {
				t.Errorf("View '%s' has no description", viewName)
			}

			if len(view.Fields) == 0 {
				t.Errorf("View '%s' has no fields", viewName)
			}

			// Verify all fields have required properties
			for i, field := range view.Fields {
				if field.Name == "" {
					t.Errorf("View '%s' field %d has no name", viewName, i)
				}
				if field.Format == "" {
					t.Errorf("View '%s' field '%s' has no format", viewName, field.Name)
				}
				if field.Show == nil {
					t.Errorf("View '%s' field '%s' has nil Show property", viewName, field.Name)
				}
			}
		})
	}
}

// TestEmbeddedYAMLViewsIntegration tests that embedded YAML views work through ResolveView
func TestEmbeddedYAMLViewsIntegration(t *testing.T) {
	ClearViewCache()

	testCases := []string{"minimal", "full", "kanban", "timeline", "compact"}

	for _, viewName := range testCases {
		t.Run(viewName, func(t *testing.T) {
			view, err := ResolveView(viewName)
			if err != nil {
				t.Fatalf("Failed to resolve embedded view '%s': %v", viewName, err)
			}

			if view.Name != viewName {
				t.Errorf("Expected view name '%s', got '%s'", viewName, view.Name)
			}

			// These views should now be considered built-in
			if !IsBuiltInView(viewName) {
				t.Errorf("View '%s' should be a built-in view", viewName)
			}
		})
	}
}
