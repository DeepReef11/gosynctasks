# Issue #94: Remove old config migration code

**Status**: Open
**Created**: 2025-11-16
**Labels**: None
**GitHub**: https://github.com/DeepReef11/gosynctasks/issues/94

## Description

Remove legacy config migration code that is no longer needed.

## Background

The codebase currently contains migration logic for handling old config file formats or locations. This migration code was useful during early development or a previous refactoring, but is no longer necessary.

## Current State

There appears to be config migration logic in the codebase (exact location TBD - needs investigation):
- Possibly in `internal/config/config.go`
- May involve checking for old config locations
- May involve converting from an older format
- Adds complexity to the config loading logic

## Proposed Change

**Remove all old config migration code:**
- Clean up config loading logic to only support current format/location
- Remove any fallback checks for old config paths
- Remove any format conversion logic
- Simplify the codebase

## Rationale

- **No longer needed**: Migration period has passed
- **Code simplicity**: Reduces maintenance burden
- **Cleaner codebase**: Less conditional logic to understand
- **No user impact**: Current users are already on the current config format

## Investigation Needed

Before removing, identify:
1. Where is the migration code located?
2. What does it migrate from/to?
3. Are there any comments explaining when it was added?
4. Is it actually being used (add logging to check)?

## Implementation Steps

1. **Locate migration code** - Search for:
   - Config file path checks
   - Format conversion logic
   - "migrate" or "legacy" in config-related files

2. **Verify it's safe to remove**:
   - Check git history for when it was added
   - Confirm current config system is stable
   - Ensure no recent usage

3. **Remove the code**:
   - Delete migration functions/logic
   - Simplify config loading
   - Update any related tests

4. **Update documentation**:
   - Remove migration-related docs
   - Simplify config setup instructions

## Files Likely Involved

- `internal/config/config.go`
- Any config-related test files
- Documentation (CLAUDE.md, README.md)

## Testing

- [ ] Config still loads correctly after removal
- [ ] No regression in config functionality
- [ ] Clean build with no errors
- [ ] Existing tests pass

## Priority

**Low** - Code cleanup, no functional impact

## Related

- Issue #93: Change config.json to YAML format (will use clean config loading after this cleanup)
