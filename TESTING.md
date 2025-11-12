# gosynctasks Testing Guide

Quick testing workflow for verifying core functionality.

## Setup

```bash
# Build the binary
go build -o gosynctasks ./cmd/gosynctasks

# Use test config and Test list
go run ./cmd/gosynctasks --config gosynctasks/config Test
```

## Essential Tests

### 1. Add Tasks
```bash
go run ./cmd/gosynctasks --config gosynctasks/config Test add "Buy groceries" -d "Get milk and eggs" -p 5
go run ./cmd/gosynctasks --config gosynctasks/config Test add "Write report" -p 1
go run ./cmd/gosynctasks --config gosynctasks/config Test add "Call dentist" -p 3
```

**Verify:** Tasks created with different priorities

### 2. List Tasks
```bash
# Basic view
go run ./cmd/gosynctasks --config gosynctasks/config Test

# All metadata
go run ./cmd/gosynctasks --config gosynctasks/config Test -v all
```

**Verify:**
- Priority sorting (1=highest first)
- Colors (1-4=red, 5=yellow, 6-9=blue)
- Status symbols (○ for TODO)
- Description display

### 3. Update Task
```bash
go run ./cmd/gosynctasks --config gosynctasks/config Test update "Call dentist" -p 1 -d "Schedule cleaning"
```

**Verify:** Priority and description updated

### 4. Complete Task (Partial Match)
```bash
echo "y" | go run ./cmd/gosynctasks --config gosynctasks/config Test complete "groceries"
```

**Verify:**
- Confirmation prompt for partial match
- Status changes to ✓ (green)

### 5. Filter by Status
```bash
go run ./cmd/gosynctasks --config gosynctasks/config Test -s TODO
go run ./cmd/gosynctasks --config gosynctasks/config Test -s DONE
```

**Verify:** Filtering works correctly

### 6. Delete All Tasks
```bash
echo "y" | go run ./cmd/gosynctasks --config gosynctasks/config Test delete "Write report"
echo "y" | go run ./cmd/gosynctasks --config gosynctasks/config Test delete "Call dentist"
echo "y" | go run ./cmd/gosynctasks --config gosynctasks/config Test delete "Buy groceries"
```

**Verify:** Tasks removed, list empty

## Test Results

All core features working correctly:
- ✅ Add, update, complete, delete
- ✅ Priority sorting and coloring
- ✅ Partial match search with confirmation
- ✅ Status filtering
- ✅ Terminal width adaptation
- ✅ Nextcloud CalDAV backend integration
