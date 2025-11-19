# Issue #123: UI: Parent indicators not working for nested subtasks (grandchildren)

**Status**: Open
**Created**: 2025-11-17
**Priority**: Medium
**GitHub**: https://github.com/DeepReef11/gosynctasks/issues/123

## Problem Description

The parent task indicator feature (▶ symbol with child count) implemented in PR #121 does not work correctly for nested subtasks (subtasks of subtasks/grandchildren). These still show the same character/formatting as their parent tasks.

### Current Behavior

- Top-level parent tasks: ▶ Parent (2) ✅ Works
- Subtasks that are also parents: Same character as regular subtasks ❌ Broken
- Grandchildren and deeper: Display correctly as children

### Expected Behavior

Subtasks that have their own children should also display the ▶ indicator with their child count, showing the full hierarchy clearly.

**Example:**
```
▶ Project (2)
  ├─ ▶ Phase 1 (3)
  │  ├─ Task 1.1
  │  ├─ Task 1.2
  │  └─ Task 1.3
  └─ Phase 2
```

## Technical Context

### Related Code
- PR #121 (original implementation)
- `internal/operations/subtasks.go` - Contains `addParentIndicator()` function
- `internal/operations/actions.go` - Contains `applyHierarchicalFormatting()` helper

### Root Cause

The `addParentIndicator()` function may only be checking for direct children at the top level, not accounting for tasks that are both children (have a parent_uid) and parents (have their own children).

### Proposed Fix

The hierarchical formatting logic needs to:
1. Identify all tasks that have children (not just top-level parents)
2. Apply the ▶ indicator to ANY task that has children, regardless of its depth
3. Count children correctly for nested parent tasks
4. Preserve the tree structure characters (├─, └─, │) while adding parent indicators

### Implementation Notes

The fix likely involves:
1. Building a parent-child map for all tasks (not just top-level)
2. Checking each task to see if it appears as a parent in the map
3. Applying the indicator based on this check rather than just checking for `parent_uid == ""`

### Testing Scenarios

Test with a 3-level hierarchy:
```
Task A (parent)
  └─ Task B (child of A, parent of C)
      └─ Task C (grandchild)
```

Expected output:
```
▶ Task A (1)
  └─ ▶ Task B (1)
      └─ Task C
```

## Impact

**Severity**: Medium
**Affects**: Users with complex hierarchical task structures
**Workaround**: None - users cannot see which nested tasks have children

## Related Issues

- Issue #119: Original parent indicator feature request (closed by PR #121)
- PR #121: Original implementation that introduced this bug
