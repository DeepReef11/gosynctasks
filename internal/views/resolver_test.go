package views

import (
	"strings"
	"testing"
)

func TestResolveView_BuiltIn(t *testing.T) {
	// Clear cache before test
	ClearViewCache()

	// Test resolving built-in view
	view, err := ResolveView("basic")
	if err != nil {
		t.Fatalf("Failed to resolve built-in view 'basic': %v", err)
	}

	if view.Name != "basic" {
		t.Errorf("Expected view name 'basic', got '%s'", view.Name)
	}

	if len(view.Fields) == 0 {
		t.Error("Expected basic view to have fields")
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

	// 'all' view should have more fields than 'basic'
	basicView, _ := ResolveView("basic")
	if len(view.Fields) <= len(basicView.Fields) {
		t.Errorf("Expected 'all' view to have more fields than 'basic', got %d vs %d",
			len(view.Fields), len(basicView.Fields))
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
	view1, err := ResolveView("basic")
	if err != nil {
		t.Fatalf("Failed to resolve view: %v", err)
	}

	// Second resolve (should come from cache)
	view2, err := ResolveView("basic")
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
	view1, err := ResolveView("basic")
	if err != nil {
		t.Fatalf("Failed to resolve view: %v", err)
	}

	// Clear cache
	ClearViewCache()

	// Resolve again
	view2, err := ResolveView("basic")
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
	basic1, err := ResolveView("basic")
	if err != nil {
		t.Fatalf("Failed to resolve basic: %v", err)
	}

	all1, err := ResolveView("all")
	if err != nil {
		t.Fatalf("Failed to resolve all: %v", err)
	}

	// Invalidate only basic
	InvalidateViewCache("basic")

	// Resolve both again
	basic2, _ := ResolveView("basic")
	all2, _ := ResolveView("all")

	// basic should be different (invalidated), all should be same (cached)
	if basic1 == basic2 {
		t.Error("Expected different pointer for invalidated view")
	}

	if all1 != all2 {
		t.Error("Expected same pointer for non-invalidated view")
	}
}

func TestGetBuiltInViews(t *testing.T) {
	views := GetBuiltInViews()

	if len(views) == 0 {
		t.Error("Expected at least one built-in view")
	}

	// Check for known built-in views
	hasBasic := false
	hasAll := false
	for _, name := range views {
		if name == "basic" {
			hasBasic = true
		}
		if name == "all" {
			hasAll = true
		}
	}

	if !hasBasic {
		t.Error("Expected 'basic' in built-in views list")
	}

	if !hasAll {
		t.Error("Expected 'all' in built-in views list")
	}
}

func TestIsBuiltInView(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"basic", true},
		{"all", true},
		{"custom", false},
		{"minimal", false},
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
	// Test basic view structure
	view, err := getBuiltInView("basic")
	if err != nil {
		t.Fatalf("Failed to get built-in 'basic' view: %v", err)
	}

	if view.Name != "basic" {
		t.Errorf("Expected name 'basic', got '%s'", view.Name)
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

	if !strings.Contains(err.Error(), "unknown built-in view") {
		t.Errorf("Expected 'unknown built-in view' error, got: %v", err)
	}
}
