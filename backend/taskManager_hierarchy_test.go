package backend

import (
	"testing"
	"time"
)

func TestOrganizeTasksHierarchically(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		tasks    []Task
		expected []TaskWithLevel // Expected order and levels
	}{
		{
			name:     "empty task list",
			tasks:    []Task{},
			expected: nil,
		},
		{
			name: "single task without parent",
			tasks: []Task{
				{UID: "task1", Summary: "Task 1", Created: now},
			},
			expected: []TaskWithLevel{
				{Task: Task{UID: "task1", Summary: "Task 1", Created: now}, Level: 0},
			},
		},
		{
			name: "parent with one child",
			tasks: []Task{
				{UID: "parent1", Summary: "Parent Task", Created: now},
				{UID: "child1", Summary: "Child Task", ParentUID: "parent1", Created: now},
			},
			expected: []TaskWithLevel{
				{Task: Task{UID: "parent1", Summary: "Parent Task", Created: now}, Level: 0},
				{Task: Task{UID: "child1", Summary: "Child Task", ParentUID: "parent1", Created: now}, Level: 1},
			},
		},
		{
			name: "parent with multiple children",
			tasks: []Task{
				{UID: "parent1", Summary: "Parent Task", Created: now},
				{UID: "child1", Summary: "Child 1", ParentUID: "parent1", Created: now},
				{UID: "child2", Summary: "Child 2", ParentUID: "parent1", Created: now},
			},
			expected: []TaskWithLevel{
				{Task: Task{UID: "parent1", Summary: "Parent Task", Created: now}, Level: 0},
				{Task: Task{UID: "child1", Summary: "Child 1", ParentUID: "parent1", Created: now}, Level: 1},
				{Task: Task{UID: "child2", Summary: "Child 2", ParentUID: "parent1", Created: now}, Level: 1},
			},
		},
		{
			name: "nested hierarchy (grandchildren)",
			tasks: []Task{
				{UID: "parent1", Summary: "Parent Task", Created: now},
				{UID: "child1", Summary: "Child Task", ParentUID: "parent1", Created: now},
				{UID: "grandchild1", Summary: "Grandchild Task", ParentUID: "child1", Created: now},
			},
			expected: []TaskWithLevel{
				{Task: Task{UID: "parent1", Summary: "Parent Task", Created: now}, Level: 0},
				{Task: Task{UID: "child1", Summary: "Child Task", ParentUID: "parent1", Created: now}, Level: 1},
				{Task: Task{UID: "grandchild1", Summary: "Grandchild Task", ParentUID: "child1", Created: now}, Level: 2},
			},
		},
		{
			name: "multiple independent trees",
			tasks: []Task{
				{UID: "parent1", Summary: "Parent 1", Created: now},
				{UID: "child1", Summary: "Child 1", ParentUID: "parent1", Created: now},
				{UID: "parent2", Summary: "Parent 2", Created: now},
				{UID: "child2", Summary: "Child 2", ParentUID: "parent2", Created: now},
			},
			expected: []TaskWithLevel{
				{Task: Task{UID: "parent1", Summary: "Parent 1", Created: now}, Level: 0},
				{Task: Task{UID: "child1", Summary: "Child 1", ParentUID: "parent1", Created: now}, Level: 1},
				{Task: Task{UID: "parent2", Summary: "Parent 2", Created: now}, Level: 0},
				{Task: Task{UID: "child2", Summary: "Child 2", ParentUID: "parent2", Created: now}, Level: 1},
			},
		},
		{
			name: "orphaned child (parent doesn't exist)",
			tasks: []Task{
				{UID: "child1", Summary: "Orphaned Child", ParentUID: "nonexistent", Created: now},
			},
			expected: []TaskWithLevel{
				{Task: Task{UID: "child1", Summary: "Orphaned Child", ParentUID: "nonexistent", Created: now}, Level: 0},
			},
		},
		{
			name: "mixed root and child tasks",
			tasks: []Task{
				{UID: "root1", Summary: "Root 1", Created: now},
				{UID: "parent1", Summary: "Parent 1", Created: now},
				{UID: "child1", Summary: "Child 1", ParentUID: "parent1", Created: now},
				{UID: "root2", Summary: "Root 2", Created: now},
			},
			expected: []TaskWithLevel{
				{Task: Task{UID: "root1", Summary: "Root 1", Created: now}, Level: 0},
				{Task: Task{UID: "parent1", Summary: "Parent 1", Created: now}, Level: 0},
				{Task: Task{UID: "child1", Summary: "Child 1", ParentUID: "parent1", Created: now}, Level: 1},
				{Task: Task{UID: "root2", Summary: "Root 2", Created: now}, Level: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := OrganizeTasksHierarchically(tt.tasks)

			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d tasks, got %d", len(tt.expected), len(result))
				return
			}

			// Check each task
			for i, expected := range tt.expected {
				if i >= len(result) {
					t.Errorf("missing task at index %d", i)
					continue
				}

				actual := result[i]

				if actual.Task.UID != expected.Task.UID {
					t.Errorf("index %d: expected UID %s, got %s", i, expected.Task.UID, actual.Task.UID)
				}

				if actual.Level != expected.Level {
					t.Errorf("task %s: expected level %d, got %d", actual.Task.UID, expected.Level, actual.Level)
				}

				if actual.Task.Summary != expected.Task.Summary {
					t.Errorf("task %s: expected summary %s, got %s", actual.Task.UID, expected.Task.Summary, actual.Task.Summary)
				}

				if actual.Task.ParentUID != expected.Task.ParentUID {
					t.Errorf("task %s: expected parent %s, got %s", actual.Task.UID, expected.Task.ParentUID, actual.Task.ParentUID)
				}
			}
		})
	}
}

func TestOrganizeTasksHierarchically_CircularReference(t *testing.T) {
	// Test that circular references don't cause infinite loops
	now := time.Now()
	tasks := []Task{
		{UID: "task1", Summary: "Task 1", ParentUID: "task2", Created: now},
		{UID: "task2", Summary: "Task 2", ParentUID: "task1", Created: now},
	}

	result := OrganizeTasksHierarchically(tasks)

	// Circular references result in no tasks being displayed since neither is a root task
	// This is expected behavior - circular references in data should be fixed at source
	if len(result) != 0 {
		t.Errorf("expected 0 tasks (circular reference), got %d", len(result))
	}
}
