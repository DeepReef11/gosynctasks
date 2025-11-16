# GoSyncTasks Architecture Diagrams

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLI Layer (Cobra)                        │
│  main.go: root command, flags, arg parsing                      │
│  list.go: list subcommands                                      │
│  sync.go: sync operations                                       │
│  view.go: view management                                       │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────┴──────────────────────────────────────┐
│                     Operations Layer                            │
│  ExecuteAction → action handlers (Get/Add/Update/Complete)     │
│  FindTaskBySummary → intelligent task search                   │
│  CreateOrFindTaskPath → subtask hierarchy                      │
│  ListOperations → list management                              │
└──────────────────┬──────────────────────────────────────────────┘
                   │
         ┌─────────┴──────────────┐
         │                        │
    ┌────▼─────┐         ┌────────▼────────┐
    │  Config  │         │  Cache System   │
    │  (464)   │         │  (cache.go)     │
    └──────────┘         └─────────────────┘
         │
    ┌────▼──────────────────────────────────────────────────────┐
    │            Backend Abstraction Layer                      │
    │                                                            │
    │  TaskManager Interface (20+ methods)                      │
    │  ├─ CRUD: GetTasks, AddTask, UpdateTask, DeleteTask      │
    │  ├─ Lists: GetTaskLists, CreateTaskList, DeleteTaskList  │
    │  ├─ Status: ParseStatusFlag, StatusToDisplayName         │
    │  ├─ Display: GetBackendDisplayName, GetPriorityColor     │
    │  └─ Hierarchy: SortTasks (for parents before children)   │
    │                                                            │
    │  DetectableBackend (optional): CanDetect, DetectionInfo  │
    └────┬─────────────────────────────────────────────────────┘
         │
    ┌────┴──────────────────────────────────────────────┐
    │       Backend Registry & Selector                  │
    │  ├─ BackendRegistry: Initialize & manage backends │
    │  └─ BackendSelector: Priority-based selection     │
    │     1. Explicit (--backend flag)                   │
    │     2. Auto-detect (if enabled)                    │
    │     3. Default backend                             │
    │     4. First enabled backend                       │
    └────┬──────────────────────────────────────────────┘
         │
    ┌────┴──────────────────────────────────────────────────────┐
    │              Concrete Backend Implementations              │
    │                                                             │
    │  ┌─────────────────────────────────────────────┐           │
    │  │   NextcloudBackend (825 lines)              │           │
    │  │  CalDAV/VTODO protocol for Nextcloud Tasks │           │
    │  │  - PROPFIND (list discovery)                │           │
    │  │  - REPORT (task queries)                    │           │
    │  │  - PUT/DELETE (mutations)                   │           │
    │  │  - ETags & CTag sync tracking               │           │
    │  └─────────────────────────────────────────────┘           │
    │                                                             │
    │  ┌─────────────────────────────────────────────┐           │
    │  │   SQLiteBackend (969 lines)                 │           │
    │  │  Local task storage with sync support       │           │
    │  │  - Schema: tasks, sync_metadata, queue      │           │
    │  │  - Transactional operations                 │           │
    │  │  - Sync flags & queuing                     │           │
    │  └──────────┬──────────────────────────────────┘           │
    │             │                                              │
    │             │         ┌──────────────────────┐             │
    │             ├────────▶│  SyncManager         │             │
    │             │         │  (765 lines)         │             │
    │  ┌──────────┤         │  ├─ Pull phase       │             │
    │  │          │         │  ├─ Push phase       │             │
    │  │          │         │  ├─ Conflict res.    │             │
    │  │          │         │  └─ Retry logic      │             │
    │  │          │         └──────────────────────┘             │
    │  │          │                                              │
    │  │     ┌────▼──────────────────────────────┐               │
    │  │     │  Remote Backend                   │               │
    │  │     │  (Nextcloud, File, Git, etc.)     │               │
    │  │     └───────────────────────────────────┘               │
    │  │                                                          │
    │  └─────────────────┬─────────────────────────┐             │
    │                    │                         │             │
    │  ┌─────────────────▼──────┐   ┌─────────────▼──────────┐   │
    │  │  GitBackend (621)       │   │  FileBackend (stub)    │   │
    │  │  Git repo + Markdown    │   │  (non-functional)      │   │
    │  │  - TODO.md parsing      │   │  - Placeholder only    │   │
    │  │  - Metadata annotations │   │  - Future file system  │   │
    │  │  - Auto-commit support  │   │    storage             │   │
    │  └─────────────────────────┘   └────────────────────────┘   │
    │                                                             │
    └─────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                        Views System                              │
│  ├─ View Definition (types.go)                                  │
│  ├─ ViewRenderer: Format tasks according to view config         │
│  ├─ Field Formatters: status, priority, dates, text, tags      │
│  ├─ Filtering & Sorting: Apply view-defined filters            │
│  └─ View Builder (TUI): Interactive custom view creation       │
└──────────────────────────────────────────────────────────────────┘
```

---

## Data Flow: Getting Tasks

```
User Command: gosynctasks MyList get --view basic
     │
     ▼
main.go: Parse args & flags
     │
     ▼
App.Run() in app.go
     │
     ├─ Refresh cache from backend
     │
     ▼
operations.ExecuteAction()
     │
     ├─ Resolve list name → TaskList
     │
     ├─ Build filters from flags
     │    └─ status, priority, date ranges
     │
     ├─ taskManager.GetTasks(listID, filter)
     │    └─ Backend-specific query
     │
     ├─ Load view configuration (basic/all/custom)
     │
     ├─ ViewRenderer.RenderTask() for each task
     │    ├─ Initialize field formatters
     │    ├─ Apply filters
     │    ├─ Format each field
     │    └─ Return formatted string
     │
     ├─ OrganizeTasksHierarchically() if needed
     │    └─ Build tree: parents before children
     │
     ▼
Display tasks with colors, status indicators, dates
```

---

## Data Flow: Adding Task with Subtask

```
User Command: gosynctasks MyList add "parent/child" -P "Parent"
     │
     ▼
main.go: Parse args
     │
     ▼
operations.HandleAddAction()
     │
     ├─ Parse summary: "parent/child"
     │
     ├─ Check for path-based creation (contains "/")
     │
     ├─ CreateOrFindTaskPath()
     │    │
     │    ├─ For "parent":
     │    │    └─ FindTaskBySummary() → exists? yes → get UID
     │    │
     │    └─ Return (parentUID, "child")
     │
     ├─ Get flags: status, priority, description, etc.
     │
     ├─ Create Task object:
     │    Task{
     │        UID: auto-generated
     │        Summary: "child"
     │        ParentUID: <parent-uid>
     │        Status: TODO (or from flag)
     │        Created/Modified: now
     │    }
     │
     ▼
taskManager.AddTask(listID, task)
     │
     ├─ NextcloudBackend:
     │    ├─ Generate VTODO iCalendar format
     │    └─ HTTP PUT to Nextcloud server
     │
     ├─ SQLiteBackend:
     │    ├─ Insert into tasks table
     │    ├─ Mark locally_modified = 0 (already created)
     │    └─ Return success
     │
     ▼
Display: Task created successfully
```

---

## Data Flow: Sync Operation

```
User Command: gosynctasks sync
     │
     ▼
sync.go: newSyncCmd()
     │
     ├─ Check if sync enabled in config
     ├─ Get local backend (SQLiteBackend)
     ├─ Get remote backend (NextcloudBackend, GitBackend, etc.)
     │
     ▼
SyncManager.Sync()
     │
     ├─ Phase 1: Pull (Remote → Local)
     │    │
     │    ├─ remote.GetTaskLists()
     │    │    └─ For each remote list
     │    │
     │    ├─ Check if local list exists
     │    │    ├─ If not: create locally
     │    │    └─ If yes: proceed
     │    │
     │    ├─ remote.GetTasks(listID)
     │    │    └─ All remote tasks
     │    │
     │    ├─ For each remote task:
     │    │    ├─ Check CTag (cache tag) for changes
     │    │    ├─ Look up local sync_metadata
     │    │    ├─ Detect conflict (if both modified)
     │    │    │    └─ Apply conflict_resolution strategy
     │    │    │       • server_wins: use remote
     │    │    │       • local_wins: keep local
     │    │    │       • merge: combine fields
     │    │    │       • keep_both: create duplicate
     │    │    │
     │    │    ├─ Insert/update in local SQLite
     │    │    └─ Update sync_metadata (etag, timestamps)
     │    │
     │    └─ Clear locally_modified flag for pulled tasks
     │
     ├─ Phase 2: Push (Local → Remote)
     │    │
     │    ├─ local.GetSyncQueue()
     │    │    └─ Pending CREATE/UPDATE/DELETE operations
     │    │
     │    ├─ Sort tasks: parents before children
     │    │    └─ Prevent FK violations when creating
     │    │
     │    ├─ For each queued operation:
     │    │    ├─ Retry up to 5 times
     │    │    │    └─ Exponential backoff: 2^attempt seconds
     │    │    │
     │    │    ├─ Switch operation:
     │    │    │    ├─ CREATE: remote.AddTask()
     │    │    │    ├─ UPDATE: remote.UpdateTask()
     │    │    │    └─ DELETE: remote.DeleteTask()
     │    │    │
     │    │    ├─ On success:
     │    │    │    ├─ Remove from sync_queue
     │    │    │    └─ Update sync_metadata (etag)
     │    │    │
     │    │    └─ On failure:
     │    │         ├─ Increment retry_count
     │    │         ├─ Store last_error
     │    │         └─ Leave in queue for next sync
     │    │
     │    └─ Return sync result (stats & errors)
     │
     ▼
Display sync summary:
  ✓ Pulled: 15 tasks
  ✓ Pushed: 3 tasks
  ⚠ Conflicts: 2 (resolved via server_wins)
  ⚠ Errors: 0
```

---

## Component Interaction Map

```
                        User Input (CLI)
                             │
                    ┌────────┴────────┐
                    │                 │
              CLI Commands        Main App
              (list/sync/view)    (root action)
                    │                 │
                    ├─────────────────┤
                    │                 │
        ┌───────────▼────────────────▼───────────┐
        │    operations.ExecuteAction()           │
        │  (actions.go: Get/Add/Update handlers)  │
        └────────────────┬────────────────────────┘
                         │
        ┌────────────────┴─────────────────────┐
        │                                      │
    ┌───▼────────────────┐      ┌─────────────▼───────┐
    │  TaskManager       │      │  Internal Modules   │
    │  Interface         │      │  ├─ config/         │
    │  (20+ methods)     │      │  ├─ cache/          │
    │                    │      │  ├─ views/          │
    │  ┌────────────────┐│      │  └─ utils/          │
    │  │ NextcloudBackend││      │                     │
    │  │ (CalDAV/HTTP)  ││      └─────────────────────┘
    │  └────────────────┘│
    │                    │
    │  ┌────────────────┐│  ┌──────────────┐
    │  │ SQLiteBackend  │├─▶│ SyncManager  │
    │  │ (local storage)││  │ (orchestrate)│
    │  └────────────────┘│  └──────┬───────┘
    │                    │         │
    │  ┌────────────────┐│    ┌────▼────┐
    │  │ GitBackend     ││    │ Remote   │
    │  │ (Markdown/Git) ││    │ Backend  │
    │  └────────────────┘│    └──────────┘
    │                    │
    │  ┌────────────────┐│
    │  │ FileBackend    ││
    │  │ (placeholder)  ││
    │  └────────────────┘│
    └────────┬───────────┘
             │
    ┌────────▼──────────────────────┐
    │  Backend Registry & Selector   │
    │  ├─ Initialize backends        │
    │  ├─ Select by priority         │
    │  └─ Auto-detect capabilities   │
    └────────────────────────────────┘
```

---

## Database Schema Relationships

```
┌──────────────────────────────┐
│   schema_version             │
│  ┌──────────────────────────┐│
│  │ version (PK)             ││
│  │ applied_at               ││
│  └──────────────────────────┘│
└──────────────────────────────┘

┌──────────────────────────────────┐
│   list_sync_metadata             │
│  ┌──────────────────────────────┐│
│  │ list_id (PK)                 ││
│  │ list_name                    ││
│  │ list_color                   ││
│  │ last_ctag                    ││ ◄── CTag for efficient sync
│  │ last_full_sync               ││
│  │ sync_token                   ││
│  │ created_at                   ││
│  │ modified_at                  ││
│  └──────────────────────────────┘│
└──────────────────────────────────┘
          ▲
          │ (1:N)
          │
┌─────────┴──────────────────────────┐
│   tasks                            │
│  ┌────────────────────────────────┐│
│  │ id (PK)                        ││
│  │ list_id (FK → list_sync_metadata)
│  │ summary                        ││
│  │ description                    ││
│  │ status                         ││
│  │ priority (0-9)                 ││
│  │ created_at                     ││
│  │ modified_at                    ││
│  │ due_date                       ││
│  │ start_date                     ││
│  │ completed_at                   ││
│  │ parent_uid (FK → tasks.id)     ││ ◄── Subtask hierarchy
│  │ categories                     ││
│  └──────────┬──────────────────────┘│
│             │                       │
└─────────────┼───────────────────────┘
              │
              │ (1:1)
              │
    ┌─────────▼─────────────────────┐
    │   sync_metadata               │
    │  ┌──────────────────────────┐ │
    │  │ task_uid (PK/FK)         │ │
    │  │ list_id (FK)             │ │
    │  │ remote_etag              │ │ ◄── For conflict detection
    │  │ last_synced_at           │ │
    │  │ locally_modified (flag)   │ │ ◄── Tracks local changes
    │  │ locally_deleted (flag)    │ │
    │  │ remote_modified_at       │ │
    │  │ local_modified_at        │ │
    │  └──────────────────────────┘ │
    └───────────────────────────────┘

┌──────────────────────────────────┐
│   sync_queue                     │
│  ┌──────────────────────────────┐│
│  │ id (PK)                      ││
│  │ task_uid                     ││
│  │ list_id                      ││
│  │ operation (create/update/del)││
│  │ created_at                   ││
│  │ retry_count                  ││
│  │ last_error                   ││
│  │ UNIQUE(task_uid, operation)  ││
│  └──────────────────────────────┘│
└──────────────────────────────────┘
```

---

## View Rendering Pipeline

```
User: gosynctasks MyList --view custom_view

     │
     ▼
ViewResolver.ResolveView("custom_view")
     │
     ├─ Check built-in views: basic, all
     ├─ Check custom views: ~/.config/gosynctasks/views/
     │
     ▼
Load View YAML Configuration:
{
  name: "custom_view"
  fields: [
    {name: "status", format: "symbol", color: true},
    {name: "summary", format: "text", width: 50},
    {name: "priority", format: "numeric", color: true},
    {name: "due_date", format: "YYYY-MM-DD", color: true}
  ]
  filters: {
    status: ["TODO", "PROCESSING"],
    priority_min: 1,
    priority_max: 4
  }
}

     │
     ▼
ViewRenderer(view, backend, dateFormat)
     │
     ├─ Initialize formatters:
     │    ├─ StatusFormatter → FieldFormatter (interface)
     │    ├─ SummaryFormatter
     │    ├─ PriorityFormatter
     │    └─ DateFormatter
     │
     ├─ GetTasks() from backend
     │
     ├─ Filter tasks:
     │    ├─ By status (apply view filters)
     │    ├─ By priority range
     │    ├─ By tags
     │    └─ By dates
     │
     ├─ Sort tasks:
     │    └─ By field (asc/desc, per view config)
     │
     ├─ Organize hierarchically:
     │    └─ If show_hierarchy: parent/child indentation
     │
     ├─ Format each task:
     │    ├─ RenderTask(task) with field order
     │    │    └─ For each field:
     │    │        └─ formatter.Format(task, fieldConfig)
     │    │           ├─ StatusFormatter: ✓/●/✗/○ + color
     │    │           ├─ PriorityFormatter: 1-9 + color
     │    │           ├─ DateFormatter: format + urgency color
     │    │           └─ SummaryFormatter: truncate + bold
     │    │
     │    └─ Return formatted line with ANSI codes
     │
     ▼
Display to terminal with colors and formatting
```

---

## Backend Selection Flow

```
App initialization:

config.GetConfig()
     │
     ├─ Load from file (with migration if needed)
     │
     └─ backends: {
          "local": {type: "sqlite", ...},
          "nextcloud": {type: "nextcloud", ...},
          "git": {type: "git", ...}
        }

     │
     ▼
BackendRegistry.NewBackendRegistry()
     │
     ├─ For each config in backends:
     │    ├─ If enabled:
     │    │    └─ Call config.TaskManager()
     │    │         └─ Instantiate concrete backend
     │    │            └─ If initialization fails: skip
     │    │
     │    └─ Store in registry
     │
     └─ Return BackendRegistry

     │
     ▼
BackendSelector.Select() with priority:

  1. Explicit: --backend nextcloud
       └─ registry.GetBackend("nextcloud")

  2. Auto-detect: auto_detect_backend=true
       └─ For each DetectableBackend:
          └─ If CanDetect(): use this one

  3. Default: default_backend="nextcloud"
       └─ registry.GetBackend("nextcloud")

  4. First enabled:
       └─ registry.GetEnabledBackends()[0]

     │
     ▼
Return (selectedBackendName, TaskManager)
```

