package operations

import (
	"gosynctasks/backend"
	"strings"
	"testing"
	"time"
)

// TestParentIndicator_NestedHierarchy tests that parent indicators appear for all tasks with children,
// including intermediate tasks that are both parents and children themselves.
// This addresses issue #123: UI: Parent indicators not working for nested subtasks (grandchildren)
func TestParentIndicator_NestedHierarchy(t *testing.T) {
	now := time.Now()

	// Create a 3-level hierarchy:
	// Project (2 children)
	//   ├─ Phase 1 (3 children) ← This is an intermediate parent (both parent and child)
	//   │  ├─ Task 1.1
	//   │  ├─ Task 1.2
	//   │  └─ Task 1.3
	//   └─ Phase 2 (no children)
	tasks := []backend.Task{
		{
			UID:      "project",
			Summary:  "Project",
			Status:   "NEEDS-ACTION",
			Created:  now,
			Modified: now,
		},
		{
			UID:       "phase1",
			Summary:   "Phase 1",
			Status:    "NEEDS-ACTION",
			ParentUID: "project",
			Created:   now,
			Modified:  now,
		},
		{
			UID:       "phase2",
			Summary:   "Phase 2",
			Status:    "NEEDS-ACTION",
			ParentUID: "project",
			Created:   now,
			Modified:  now,
		},
		{
			UID:       "task11",
			Summary:   "Task 1.1",
			Status:    "NEEDS-ACTION",
			ParentUID: "phase1",
			Created:   now,
			Modified:  now,
		},
		{
			UID:       "task12",
			Summary:   "Task 1.2",
			Status:    "NEEDS-ACTION",
			ParentUID: "phase1",
			Created:   now,
			Modified:  now,
		},
		{
			UID:       "task13",
			Summary:   "Task 1.3",
			Status:    "NEEDS-ACTION",
			ParentUID: "phase1",
			Created:   now,
			Modified:  now,
		},
	}

	// Build task tree
	tree := BuildTaskTree(tasks)

	// Format the tree
	mockBackend := &backend.NextcloudBackend{}
	output := FormatTaskTree(tree, "basic", mockBackend, "2006-01-02")

	// Verify Project has parent indicator with count (2)
	if !strings.Contains(output, "▶") {
		t.Error("Expected parent indicator ▶ in output")
	}
	if !strings.Contains(output, "Project") {
		t.Error("Expected 'Project' in output")
	}

	// Extract lines for detailed checking
	lines := strings.Split(output, "\n")

	// Find and verify Project line (should have ▶ and (2))
	projectFound := false
	phase1Found := false
	phase2Found := false

	for _, line := range lines {
		if strings.Contains(line, "Project") && !strings.Contains(line, "Phase") {
			projectFound = true
			if !strings.Contains(line, "▶") {
				t.Error("Project line should contain parent indicator ▶")
			}
			if !strings.Contains(line, "(2)") {
				t.Errorf("Project should show child count (2), got: %s", line)
			}
		}

		if strings.Contains(line, "Phase 1") {
			phase1Found = true
			// Phase 1 is an intermediate parent - it should have the indicator!
			if !strings.Contains(line, "▶") {
				t.Errorf("Phase 1 (intermediate parent) should contain parent indicator ▶, got: %s", line)
			}
			if !strings.Contains(line, "(3)") {
				t.Errorf("Phase 1 should show child count (3), got: %s", line)
			}
		}

		if strings.Contains(line, "Phase 2") {
			phase2Found = true
			// Phase 2 has no children - should NOT have the indicator
			if strings.Contains(line, "▶") {
				t.Errorf("Phase 2 (no children) should NOT contain parent indicator ▶, got: %s", line)
			}
		}
	}

	if !projectFound {
		t.Error("Project not found in output")
	}
	if !phase1Found {
		t.Error("Phase 1 not found in output")
	}
	if !phase2Found {
		t.Error("Phase 2 not found in output")
	}

	// Print output for manual verification during development
	t.Logf("Tree output:\n%s", output)
}

// TestAddParentIndicator tests the addParentIndicator function directly
func TestAddParentIndicator(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		childCount  int
		expected    string
	}{
		{
			name:       "simple task",
			input:      "  ○ My Task\n",
			childCount: 3,
			expected:   "  ▶ ○ My Task (3)\n",
		},
		{
			name:       "task with metadata",
			input:      "  ○ My Task (due: 2025-01-20)\n     Description\n",
			childCount: 2,
			expected:   "  ▶ ○ My Task (due: 2025-01-20) (2)\n     Description\n",
		},
		{
			name:       "task with leading spaces",
			input:      "    ○ Indented Task\n",
			childCount: 1,
			expected:   "    ▶ ○ Indented Task (1)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addParentIndicator(tt.input, tt.childCount)
			if result != tt.expected {
				t.Errorf("addParentIndicator() =\n%q\nwant:\n%q", result, tt.expected)
			}
		})
	}
}
