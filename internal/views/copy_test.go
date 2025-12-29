package views

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyBuiltInViewsToUserConfig(t *testing.T) {
	// Get views directory
	viewsDir, err := GetViewsDir()
	if err != nil {
		t.Fatalf("Failed to get views directory: %v", err)
	}

	// Clean up any existing views directory for a clean test
	os.RemoveAll(viewsDir)

	// Test copying built-in views
	copied, err := CopyBuiltInViewsToUserConfig()
	if err != nil {
		t.Fatalf("Failed to copy built-in views: %v", err)
	}

	if !copied {
		t.Error("Expected views to be copied, but they weren't")
	}

	// Verify all built-in views were copied
	builtInViews := GetBuiltInViews()
	for _, viewName := range builtInViews {
		viewPath := filepath.Join(viewsDir, viewName+".yaml")
		if _, err := os.Stat(viewPath); os.IsNotExist(err) {
			t.Errorf("Expected view file %s to exist, but it doesn't", viewPath)
		}
	}

	// Test that running copy again doesn't copy (views already exist)
	copied, err = CopyBuiltInViewsToUserConfig()
	if err != nil {
		t.Fatalf("Second copy failed: %v", err)
	}

	if copied {
		t.Error("Expected views not to be copied again, but they were")
	}

	// Verify basic view has filters
	basicView, err := ResolveView("basic")
	if err != nil {
		t.Fatalf("Failed to load basic view: %v", err)
	}

	if basicView.Filters == nil {
		t.Error("Basic view should have filters")
	} else if len(basicView.Filters.ExcludeStatuses) == 0 {
		t.Error("Basic view filters should exclude some statuses")
	} else {
		expectedStatuses := []string{"DONE", "COMPLETED", "CANCELLED"}
		for _, status := range expectedStatuses {
			found := false
			for _, excluded := range basicView.Filters.ExcludeStatuses {
				if excluded == status {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected basic view to exclude status %s", status)
			}
		}
	}
}
