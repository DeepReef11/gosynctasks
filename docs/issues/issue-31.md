# Issue #31: Reduce duplication between built-in views and view templates

**Status**: Open
**Created**: 2025-11-12
**Labels**: None
**GitHub**: https://github.com/DeepReef11/gosynctasks/issues/31

## Problem

Built-in views are defined in two places with similar but not identical configurations:

1. **internal/views/resolver.go** - Built-in views (basic, all)
2. **cmd/gosynctasks/view.go** - View templates (minimal, full, kanban, timeline, compact)

This creates maintenance burden and potential inconsistency.

## Current Duplication

**resolver.go:73-112 (getBuiltInView)**
```go
case "basic":
    return &View{
        Name: "basic",
        Fields: []FieldConfig{
            {Name: "status", Format: "symbol", Show: true},
            {Name: "summary", Format: "full", Show: true},
            // ...
        },
    }
```

**view.go:417-432 (getViewTemplate "minimal")**
```go
case "minimal":
    return &View{
        Name: "minimal",
        Fields: []FieldConfig{
            {Name: "status", Format: "symbol", Show: true},
            {Name: "summary", Format: "full", Show: true},
            // ...
        },
    }
```

## Issues

1. **Similar views with different names:**
   - "basic" (built-in) ≈ "minimal" (template)
   - "all" (built-in) ≈ "full" (template)

2. **Inconsistent configurations:**
   - Some templates have features not in built-ins (sort_by, sort_order)
   - Different date formats
   - Different field selections

3. **No way to create built-in views from templates:**
   - `gosynctasks view create myview --template minimal` creates user view
   - No way to install template as built-in

## Proposed Solutions

### Option 1: Embed Templates as Built-ins
Make all templates available as built-in views:

```go
// internal/views/resolver.go
func getBuiltInView(name string) (*View, error) {
    // Check legacy built-ins first (basic, all)
    if view, ok := legacyViews[name]; ok {
        return view, nil
    }

    // Check templates (minimal, full, kanban, timeline, compact)
    return getViewTemplate(name)
}
```

**Benefits:**
- Users can use `--view kanban` without creating custom view
- Single source of truth for view configurations
- Templates become discoverable via `gosynctasks view list`

**Drawbacks:**
- Many built-in views (7 total)
- Might be confusing which are "core" vs "convenience"

### Option 2: Consolidate Definitions
Move all view definitions to a single location:

```go
// internal/views/builtin_views.go
var BuiltInViews = map[string]*View{
    "basic": {...},
    "all": {...},
    "minimal": {...},
    "full": {...},
    "kanban": {...},
    "timeline": {...},
    "compact": {...},
}

// Mark which are templates vs core
var TemplateViews = []string{"minimal", "full", "kanban", "timeline", "compact"}
```

### Option 3: Embed YAML Files (Most Flexible)
Store built-in views as embedded YAML:

```go
//go:embed builtin_views/*.yaml
var builtinViewFS embed.FS

func getBuiltInView(name string) (*View, error) {
    data, err := builtinViewFS.ReadFile("builtin_views/" + name + ".yaml")
    if err != nil {
        return nil, err
    }
    return LoadViewFromBytes(data, name)
}
```

**Benefits:**
- YAML is easier to read/edit than Go structs
- Can document views with comments
- Same format as user views
- Easy to add new built-in views without code changes

**Example: builtin_views/kanban.yaml**
```yaml
# Kanban-style view grouped by status
name: kanban
description: Kanban-style view grouped by status
fields:
  - name: status
    format: emoji
    show: true
  - name: summary
    format: truncate
    width: 50
    show: true
  # ...
```

## Recommendation

**Use Option 3 (Embedded YAML):**
- Most maintainable
- Consistent with user view format
- Easy to add new views
- Self-documenting

## Implementation Steps

1. Create `internal/views/builtin_views/` directory
2. Convert all built-in views to YAML files
3. Update resolver.go to read from embedded FS
4. Deprecate getViewTemplate() in cmd/gosynctasks/view.go
5. Update view create command to use built-in views as templates
6. Add tests for embedded view loading

## Migration

- Keep legacy "basic" and "all" as-is for backward compatibility
- Mark cmd/gosynctasks/view.go templates as deprecated
- Provide migration path: `gosynctasks view migrate-templates`

## Priority

Low - Code quality improvement, no user-facing issues

## Related

- Issue #28 (error messages) - would benefit from single definition source
- Issue #30 (ViewFilters) - easier to test with YAML-based views
