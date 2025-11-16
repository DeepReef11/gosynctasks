# GitHub Issues Summary

**Repository:** DeepReef11/gosynctasks  
**Last Updated:** 2025-11-11  
**Total Issues:** 15 (14 open, 1 closed)

## Priority Issues (Bugs)

### Issue #3: Improve error message for connection failures
**Labels:** bug  
**Status:** open

When connection fails (wrong credentials, wrong URL, network error), the error message shows:
```
Error: list 'tasks' not found
```

**Should show:**
- "Failed to connect to Nextcloud: authentication failed (401 Unauthorized)"
- "Failed to connect to Nextcloud: connection refused - check URL"
- "Failed to connect to Nextcloud: network timeout - check connectivity"

**Fix Strategy:**
1. Improve error propagation from backend to CLI
2. Distinguish error types (network, auth, server, parse)
3. Provide actionable guidance

**Files affected:**
- `cmd/gosynctasks/main.go`
- `backend/nextcloudBackend.go` - GetTaskLists() method

---

### Issue #2: Fix add action status flag (-S) not applying correctly
**Labels:** bug  
**Status:** open

The `-S` flag for setting task status when adding tasks may not be applying correctly.

**Current:** `gosynctasks MyList add "Task summary" -S DONE` may not set the status  
**Expected:** Task should be created with the specified status

**Investigation needed:**
- Verify status flag is being read
- Check if status is passed to backend's AddTask method
- Test with all status values (TODO/T, DONE/D, PROCESSING/P, CANCELLED/C)
- Ensure abbreviations are expanded correctly

**Files affected:**
- `cmd/gosynctasks/main.go` (lines 371-380, 475-490)

---

## Epic Issues

### Issue #10: [EPIC] Multi-Backend Support & Git Backend
**Status:** open

Major feature to support multiple backends including:
- Nextcloud (CalDAV)
- Git (Markdown files in repos)
- SQLite (local with sync)
- File (local file-based)

**Sub-issues:**
- #11: Phase 1 - Config Restructuring
- #12: Phase 2 - Backend Selection Logic
- #13: Phase 3 - Git Backend Implementation
- #14: Phase 4 - Multi-Backend Testing
- #15: Phase 5 - Multi-Backend Documentation

---

### Issue #4: [EPIC] SQLite Sync Implementation with Offline Mode
**Status:** open

Implement local SQLite database with sync capabilities for offline work.

**Sub-issues:**
- #5: Phase 1 - Enhanced SQLite Schema with Sync Metadata
- #6: Phase 2 - SQLite Backend Implementation
- #7: Phase 3 - SyncManager Implementation
- #8: Phase 4 - CLI Integration & Offline Mode
- #9: Phase 5 - Testing & Documentation

**Features:**
- Local SQLite cache
- Offline mode support
- Sync with remote backends
- Conflict resolution
- ETag support

---

## Completed Issues

### Issue #1: Create GitHub Issues Programmatically
**Status:** closed

Successfully created GitHub issues from codebase documentation.

---

## Issue Links

All issues: https://github.com/DeepReef11/gosynctasks/issues

