# Parent Task Visual Indicators

**Issue**: #119
**Status**: Completed
**Implementation**: PR #121 (commit 2093505), PR #128 (commit a7a9e1d)
**Date**: November 17, 2025

## Overview

Parent task visual indicators enhance the task display by adding clear visual markers to tasks that have children (subtasks). This makes hierarchical task relationships immediately apparent when viewing task lists.

## Features

- **▶ Symbol**: All parent tasks display a right-pointing triangle (▶) prefix
- **Child Count**: Shows the number of direct children in parentheses, e.g., "(3)"
- **Multi-level Support**: Works for all hierarchy levels:
  - Root parents (top-level tasks with children)
  - Intermediate parents (tasks that are both children AND have their own children)
  - Deeply nested hierarchies (grandchildren, great-grandchildren, etc.)
- **View Integration**: Works with both legacy views (`basic`, `all`) and custom view system

## Before and After Examples

### Example 1: Simple Two-Level Hierarchy

**Before** (without parent indicators):
```
Project
  ├─ Phase 1
  ├─ Phase 2
  └─ Phase 3
```

**After** (with parent indicators):
```
▶ Project (3)
  ├─ Phase 1
  ├─ Phase 2
  └─ Phase 3
```

The `▶` symbol immediately shows "Project" is a parent task, and `(3)` indicates it has 3 children.

---

### Example 2: Multi-Level Hierarchy (Intermediate Parents)

**Before** (without parent indicators):
```
Project
  ├─ Phase 1
  │  ├─ Task 1.1
  │  ├─ Task 1.2
  │  └─ Task 1.3
  ├─ Phase 2
  │  ├─ Task 2.1
  │  └─ Task 2.2
  └─ Phase 3
```

**After** (with parent indicators):
```
▶ Project (3)
  ├─ ▶ Phase 1 (3)
  │  ├─ Task 1.1
  │  ├─ Task 1.2
  │  └─ Task 1.3
  ├─ ▶ Phase 2 (2)
  │  ├─ Task 2.1
  │  └─ Task 2.2
  └─ Phase 3
```

Key improvements:
- **Project** shows `▶` with `(3)` - has 3 direct children
- **Phase 1** shows `▶` with `(3)` - intermediate parent with 3 children
- **Phase 2** shows `▶` with `(2)` - intermediate parent with 2 children
- **Phase 3** has no indicator - it's a child but has no children of its own

---

### Example 3: Deep Nesting (4+ Levels)

**Before** (without parent indicators):
```
Epic
  └─ Feature
      └─ Story
          ├─ Subtask 1
          └─ Subtask 2
```

**After** (with parent indicators):
```
▶ Epic (1)
  └─ ▶ Feature (1)
      └─ ▶ Story (2)
          ├─ Subtask 1
          └─ Subtask 2
```

Even at deep nesting levels, every parent task correctly shows the indicator and child count.

---

### Example 4: Real-World Task List

**Before**:
```
  ○ Implement sync feature
  ├─ ○ Design database schema
  ├─ ○ Write backend code
  │  ├─ ○ Add SQLite support
  │  ├─ ○ Add CalDAV support
  │  └─ ○ Add conflict resolution
  ├─ ○ Write tests
  └─ ● Documentation
```

**After**:
```
  ▶ ○ Implement sync feature (4)
  ├─ ○ Design database schema
  ├─ ▶ ○ Write backend code (3)
  │  ├─ ○ Add SQLite support
  │  ├─ ○ Add CalDAV support
  │  └─ ○ Add conflict resolution
  ├─ ○ Write tests
  └─ ● Documentation
```

The hierarchical structure is now crystal clear:
- "Implement sync feature" is a parent with 4 direct children
- "Write backend code" is an intermediate parent with 3 children
- Other tasks are leaf tasks (no children)

---

## Technical Implementation

### Core Function: `addParentIndicator()`

Located in `internal/operations/subtasks.go`:

```go
// addParentIndicator adds a visual indicator to parent tasks showing they have children.
// It adds a prefix symbol (▶) and child count to the first line of the task output.
func addParentIndicator(taskOutput string, childCount int) string {
    lines := strings.Split(taskOutput, "\n")
    if len(lines) == 0 {
        return taskOutput
    }

    // Add the parent indicator to the first line
    // Format: "▶ [original first line] (N)"
    firstLine := lines[0]

    // Insert the indicator at the beginning (after any leading spaces)
    trimmed := strings.TrimLeft(firstLine, " ")
    leadingSpaces := firstLine[:len(firstLine)-len(trimmed)]

    lines[0] = leadingSpaces + "▶ " + trimmed + fmt.Sprintf(" (%d)", childCount)

    return strings.Join(lines, "\n")
}
```

### Integration Points

1. **Legacy Views** (`internal/operations/subtasks.go`):
   - `FormatTaskTree()` calls `addParentIndicator()` for any task with children
   - Works with `basic` and `all` views

2. **Custom Views** (`internal/operations/actions.go`):
   - `formatNodeWithCustomView()` calls `addParentIndicator()` for hierarchical rendering
   - Preserves all custom view formatting while adding parent indicators

### How It Works

1. **Tree Building**: Tasks are organized into a tree structure using `BuildTaskTree()`
2. **Child Detection**: For each node, check `len(node.Children) > 0`
3. **Indicator Application**: If children exist, call `addParentIndicator()` with child count
4. **Output Formatting**: Apply tree prefixes (├─, └─, │) and render

This approach ensures parent indicators work correctly regardless of:
- Hierarchy depth (works at any level)
- Task position (root, intermediate, or leaf)
- View type (legacy or custom)

---

## Testing

Comprehensive tests verify the functionality:

### Test File: `internal/operations/parent_indicator_test.go`

**Key Test Cases:**

1. **`TestParentIndicator_NestedHierarchy`**
   - Tests 3-level hierarchy with intermediate parents
   - Verifies `▶` symbol appears for all parent tasks
   - Validates child counts are correct
   - Ensures non-parents don't get indicators

2. **`TestAddParentIndicator`**
   - Tests the core function directly
   - Verifies formatting with different input patterns
   - Handles edge cases (metadata, indentation, etc.)

### Running Tests

```bash
go test ./internal/operations -run TestParentIndicator -v
```

Expected output shows the tree structure with indicators correctly applied.

---

## User Impact

**Benefits:**
- ✅ **Clarity**: Immediately see which tasks are parents
- ✅ **Planning**: Understand task structure at a glance
- ✅ **Navigation**: Quickly identify complex tasks with many subtasks
- ✅ **Consistency**: Works identically across all view types

**No Breaking Changes:**
- Existing functionality preserved
- Tree characters (├─, └─, │) unchanged
- All views continue to work as before
- No configuration required - automatically enabled

---

## Related Issues

- **Issue #119**: Original feature request (completed)
- **Issue #123**: Bug fix for nested parent indicators (completed)
- **PR #121**: Initial implementation (commit 2093505)
- **PR #128**: Documentation and tests (commit a7a9e1d)

---

## Future Enhancements

Potential improvements for future consideration:

1. **Configurable Symbols**: Allow users to customize the parent indicator symbol
2. **Color Coding**: Different colors for different hierarchy levels
3. **Expand/Collapse**: Interactive toggling of parent task children
4. **Statistics**: Show completion percentage for parent tasks (e.g., "3/5 done")

These would be tracked as separate feature requests.
