package operations

import (
	"gosynctasks/backend"
	"testing"
	"time"
)

// TestSortTaskTree_PreservesHierarchy verifies that sorting maintains parent-child relationships
func TestSortTaskTree_PreservesHierarchy(t *testing.T) {
	now := time.Now()

	// Create tasks with parent-child relationships
	// Alphabetically: Apple < Banana < Cherry < Zebra
	// Hierarchy: Zebra -> Apple -> Banana, Cherry (root)
	tasks := []backend.Task{
		{
			UID:      "zebra",
			Summary:  "Zebra Task",
			Status:   "NEEDS-ACTION",
			Priority: 1,
			Created:  now,
			Modified: now,
		},
		{
			UID:       "apple",
			Summary:   "Apple Task",
			Status:    "NEEDS-ACTION",
			Priority:  2,
			ParentUID: "zebra",
			Created:   now,
			Modified:  now,
		},
		{
			UID:       "banana",
			Summary:   "Banana Task",
			Status:    "NEEDS-ACTION",
			Priority:  3,
			ParentUID: "apple",
			Created:   now,
			Modified:  now,
		},
		{
			UID:      "cherry",
			Summary:  "Cherry Task",
			Status:   "NEEDS-ACTION",
			Priority: 4,
			Created:  now,
			Modified: now,
		},
	}

	// Build tree
	tree := BuildTaskTree(tasks)

	// Sort by summary (alphabetically ascending)
	SortTaskTree(tree, "summary", "asc")

	// Verify tree structure
	if len(tree) != 2 {
		t.Fatalf("Expected 2 root nodes, got %d", len(tree))
	}

	// Root nodes should be sorted: Cherry, Zebra
	if tree[0].Task.Summary != "Cherry Task" {
		t.Errorf("Expected first root to be 'Cherry Task', got '%s'", tree[0].Task.Summary)
	}
	if tree[1].Task.Summary != "Zebra Task" {
		t.Errorf("Expected second root to be 'Zebra Task', got '%s'", tree[1].Task.Summary)
	}

	// Cherry should have no children
	if len(tree[0].Children) != 0 {
		t.Errorf("Expected Cherry to have 0 children, got %d", len(tree[0].Children))
	}

	// Zebra should have Apple as child
	if len(tree[1].Children) != 1 {
		t.Fatalf("Expected Zebra to have 1 child, got %d", len(tree[1].Children))
	}
	if tree[1].Children[0].Task.Summary != "Apple Task" {
		t.Errorf("Expected Zebra's child to be 'Apple Task', got '%s'", tree[1].Children[0].Task.Summary)
	}

	// Apple should have Banana as child
	appleNode := tree[1].Children[0]
	if len(appleNode.Children) != 1 {
		t.Fatalf("Expected Apple to have 1 child, got %d", len(appleNode.Children))
	}
	if appleNode.Children[0].Task.Summary != "Banana Task" {
		t.Errorf("Expected Apple's child to be 'Banana Task', got '%s'", appleNode.Children[0].Task.Summary)
	}
}

// TestSortTaskTree_SortsChildrenSeparately verifies that children are sorted within their parent
func TestSortTaskTree_SortsChildrenSeparately(t *testing.T) {
	now := time.Now()

	// Create a parent with multiple children
	tasks := []backend.Task{
		{
			UID:      "parent",
			Summary:  "Parent Task",
			Status:   "NEEDS-ACTION",
			Priority: 1,
			Created:  now,
			Modified: now,
		},
		{
			UID:       "child-z",
			Summary:   "Zebra Child",
			Status:    "NEEDS-ACTION",
			Priority:  2,
			ParentUID: "parent",
			Created:   now,
			Modified:  now,
		},
		{
			UID:       "child-a",
			Summary:   "Apple Child",
			Status:    "NEEDS-ACTION",
			Priority:  3,
			ParentUID: "parent",
			Created:   now,
			Modified:  now,
		},
		{
			UID:       "child-m",
			Summary:   "Mango Child",
			Status:    "NEEDS-ACTION",
			Priority:  4,
			ParentUID: "parent",
			Created:   now,
			Modified:  now,
		},
	}

	// Build tree
	tree := BuildTaskTree(tasks)

	// Sort by summary (alphabetically ascending)
	SortTaskTree(tree, "summary", "asc")

	// Verify parent exists
	if len(tree) != 1 {
		t.Fatalf("Expected 1 root node, got %d", len(tree))
	}

	parent := tree[0]
	if parent.Task.Summary != "Parent Task" {
		t.Errorf("Expected root to be 'Parent Task', got '%s'", parent.Task.Summary)
	}

	// Verify children are sorted alphabetically
	if len(parent.Children) != 3 {
		t.Fatalf("Expected parent to have 3 children, got %d", len(parent.Children))
	}

	expectedOrder := []string{"Apple Child", "Mango Child", "Zebra Child"}
	for i, expected := range expectedOrder {
		if parent.Children[i].Task.Summary != expected {
			t.Errorf("Expected child %d to be '%s', got '%s'", i, expected, parent.Children[i].Task.Summary)
		}
	}
}

// TestSortTaskTree_SortByPriority verifies sorting by priority maintains hierarchy
func TestSortTaskTree_SortByPriority(t *testing.T) {
	now := time.Now()

	tasks := []backend.Task{
		{
			UID:      "parent",
			Summary:  "Parent Task",
			Status:   "NEEDS-ACTION",
			Priority: 5,
			Created:  now,
			Modified: now,
		},
		{
			UID:       "child-low",
			Summary:   "Low Priority Child",
			Status:    "NEEDS-ACTION",
			Priority:  9,
			ParentUID: "parent",
			Created:   now,
			Modified:  now,
		},
		{
			UID:       "child-high",
			Summary:   "High Priority Child",
			Status:    "NEEDS-ACTION",
			Priority:  1,
			ParentUID: "parent",
			Created:   now,
			Modified:  now,
		},
		{
			UID:       "child-medium",
			Summary:   "Medium Priority Child",
			Status:    "NEEDS-ACTION",
			Priority:  5,
			ParentUID: "parent",
			Created:   now,
			Modified:  now,
		},
	}

	// Build tree
	tree := BuildTaskTree(tasks)

	// Sort by priority (ascending - 1 is highest)
	SortTaskTree(tree, "priority", "asc")

	// Verify parent exists
	if len(tree) != 1 {
		t.Fatalf("Expected 1 root node, got %d", len(tree))
	}

	parent := tree[0]
	if len(parent.Children) != 3 {
		t.Fatalf("Expected parent to have 3 children, got %d", len(parent.Children))
	}

	// Children should be sorted by priority: 1, 5, 9
	expectedPriorities := []int{1, 5, 9}
	for i, expected := range expectedPriorities {
		if parent.Children[i].Task.Priority != expected {
			t.Errorf("Expected child %d to have priority %d, got %d", i, expected, parent.Children[i].Task.Priority)
		}
	}
}

// TestSortTaskTree_EmptyTree verifies sorting empty tree doesn't crash
func TestSortTaskTree_EmptyTree(t *testing.T) {
	tree := []*TaskNode{}

	// Should not panic
	SortTaskTree(tree, "summary", "asc")

	if len(tree) != 0 {
		t.Errorf("Expected tree to remain empty, got %d nodes", len(tree))
	}
}

// TestSortTaskTree_NoSortField verifies that empty sortBy does nothing
func TestSortTaskTree_NoSortField(t *testing.T) {
	now := time.Now()

	tasks := []backend.Task{
		{UID: "z", Summary: "Zebra", Created: now, Modified: now},
		{UID: "a", Summary: "Apple", Created: now, Modified: now},
	}

	tree := BuildTaskTree(tasks)
	originalOrder := tree[0].Task.Summary

	// Sort with empty sortBy
	SortTaskTree(tree, "", "asc")

	// Order should remain unchanged
	if tree[0].Task.Summary != originalOrder {
		t.Errorf("Expected order to remain unchanged when sortBy is empty")
	}
}

// TestSortTaskTree_DescendingOrder verifies descending sort works
func TestSortTaskTree_DescendingOrder(t *testing.T) {
	now := time.Now()

	tasks := []backend.Task{
		{UID: "a", Summary: "Apple", Created: now, Modified: now},
		{UID: "b", Summary: "Banana", Created: now, Modified: now},
		{UID: "c", Summary: "Cherry", Created: now, Modified: now},
	}

	tree := BuildTaskTree(tasks)
	SortTaskTree(tree, "summary", "desc")

	// Should be sorted Z-A: Cherry, Banana, Apple
	expected := []string{"Cherry", "Banana", "Apple"}
	for i, exp := range expected {
		if tree[i].Task.Summary != exp {
			t.Errorf("Expected node %d to be '%s', got '%s'", i, exp, tree[i].Task.Summary)
		}
	}
}

// TestSortTaskTree_MultiLevelHierarchy verifies deep nesting is preserved
func TestSortTaskTree_MultiLevelHierarchy(t *testing.T) {
	now := time.Now()

	// Create 3-level hierarchy: Root -> Child -> Grandchild
	tasks := []backend.Task{
		{
			UID:      "root",
			Summary:  "Root Task",
			Created:  now,
			Modified: now,
		},
		{
			UID:       "child",
			Summary:   "Child Task",
			ParentUID: "root",
			Created:   now,
			Modified:  now,
		},
		{
			UID:       "grandchild",
			Summary:   "Grandchild Task",
			ParentUID: "child",
			Created:   now,
			Modified:  now,
		},
	}

	tree := BuildTaskTree(tasks)
	SortTaskTree(tree, "summary", "asc")

	// Verify 3-level structure is preserved
	if len(tree) != 1 {
		t.Fatalf("Expected 1 root node, got %d", len(tree))
	}

	root := tree[0]
	if root.Task.Summary != "Root Task" {
		t.Errorf("Expected root to be 'Root Task', got '%s'", root.Task.Summary)
	}

	if len(root.Children) != 1 {
		t.Fatalf("Expected root to have 1 child, got %d", len(root.Children))
	}

	child := root.Children[0]
	if child.Task.Summary != "Child Task" {
		t.Errorf("Expected child to be 'Child Task', got '%s'", child.Task.Summary)
	}

	if len(child.Children) != 1 {
		t.Fatalf("Expected child to have 1 grandchild, got %d", len(child.Children))
	}

	grandchild := child.Children[0]
	if grandchild.Task.Summary != "Grandchild Task" {
		t.Errorf("Expected grandchild to be 'Grandchild Task', got '%s'", grandchild.Task.Summary)
	}
}
