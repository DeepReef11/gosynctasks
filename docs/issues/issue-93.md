# Issue #93: Change config.json to YAML format

**Status**: Open
**Created**: 2025-11-16
**Labels**: None
**GitHub**: https://github.com/DeepReef11/gosynctasks/issues/93

## Description

Convert the configuration file format from JSON to YAML for better readability and user experience.

## Current State

The application currently uses JSON for configuration:
- Config file: `$XDG_CONFIG_HOME/gosynctasks/config.json`
- Location: `~/.config/gosynctasks/config.json`
- Format: JSON with strict syntax requirements

## Proposed Change

Change to YAML format for the following benefits:
- **Better readability**: No need for quotes on every key, more human-friendly
- **Comments support**: Users can add inline documentation
- **Multi-line strings**: Easier to read long values
- **Consistency**: Views already use YAML format
- **User-friendly**: Less error-prone for manual editing

## User Impact

**NO USER IMPACT** - This is for developer/maintainer use only:
- Only affects the maintainer's config files
- Will be set up once for the maintainer's configurations
- No migration needed for existing users
- No backward compatibility concerns

## Implementation Details

### Files to Update

1. **internal/config/config.go**
   - Change `encoding/json` to `gopkg.in/yaml.v3`
   - Update `LoadConfig()` to parse YAML
   - Update config path from `config.json` to `config.yaml`
   - Update sample config generation

2. **Embedded sample config** (if exists)
   - Convert sample from JSON to YAML format
   - Update any references in code

3. **Documentation**
   - Update CLAUDE.md references from `.json` to `.yaml`
   - Update README.md configuration examples
   - Update any other docs mentioning config format

### Example Format Change

**Current (JSON):**
```json
{
  "backends": {
    "local": {
      "type": "sqlite",
      "enabled": true,
      "db_path": ""
    },
    "nextcloud": {
      "type": "nextcloud",
      "enabled": true,
      "url": "nextcloud://user:pass@server.com"
    }
  },
  "sync": {
    "enabled": true,
    "local_backend": "local",
    "remote_backend": "nextcloud",
    "conflict_resolution": "server_wins"
  },
  "default_backend": "local"
}
```

**Proposed (YAML):**
```yaml
# gosynctasks configuration file
# See https://github.com/DeepReef11/gosynctasks for documentation

backends:
  local:
    type: sqlite
    enabled: true
    db_path: ""  # Leave empty for default location

  nextcloud:
    type: nextcloud
    enabled: true
    url: "nextcloud://user:pass@server.com"

sync:
  enabled: true
  local_backend: local
  remote_backend: nextcloud
  conflict_resolution: server_wins  # Options: server_wins, local_wins, merge, keep_both

default_backend: local
```

## Testing Checklist

- [ ] Config loads correctly from YAML
- [ ] Validation still works with YAML format
- [ ] Sample config generation creates valid YAML
- [ ] All existing features work with new format
- [ ] Error messages are helpful for YAML syntax errors
- [ ] Documentation updated

## Dependencies

May need to add:
- `gopkg.in/yaml.v3` (likely already a dependency for views)

## Priority

**Low** - Quality of life improvement for maintainer

## Related

- Views system already uses YAML (internal/views/)
- This would bring consistency across configuration files
- See issue #94 for removing old config migration code
