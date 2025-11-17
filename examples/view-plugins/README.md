# View Plugin Examples

This directory contains example plugin scripts for custom field formatters in gosynctasks views.

## What are View Plugins?

View plugins allow you to define custom field formatters via external scripts without modifying source code. Plugins receive task data as JSON on stdin and output formatted text to stdout.

## How to Use

### 1. Configure a Plugin in Your View

Edit your view YAML file (e.g., `~/.config/gosynctasks/views/myview.yaml`):

```yaml
name: myview
description: View with custom formatters
fields:
  - name: status
    plugin:
      command: "/path/to/gosynctasks/examples/view-plugins/status-emoji.sh"
      timeout: 1000  # milliseconds (optional, default: 1000, max: 5000)

  - name: priority
    plugin:
      command: "/path/to/gosynctasks/examples/view-plugins/priority-visual.sh"

  - name: summary
    show: true

  - name: due_date
    plugin:
      command: "/path/to/gosynctasks/examples/view-plugins/date-relative.py"
```

### 2. Use the View

```bash
gosynctasks MyList -v myview
```

## Available Examples

### status-emoji.sh
Displays task status with emoji icons:
- ⏳ TODO
- ✅ DONE
- 🔄 IN PROGRESS
- ❌ CANCELLED

**Usage:**
```yaml
fields:
  - name: status
    plugin:
      command: "./examples/view-plugins/status-emoji.sh"
```

### priority-visual.sh
Shows priority with visual indicators:
- 🔥🔥🔥 P1 CRITICAL
- 🔥🔥 P2 HIGH
- 🔥 P3 HIGH
- 📌 P4-6 MEDIUM
- 💤 P7-9 LOW

**Usage:**
```yaml
fields:
  - name: priority
    plugin:
      command: "./examples/view-plugins/priority-visual.sh"
```

### github-links.sh
Converts `#123` references to GitHub issue links.

**Configuration:**
Set `GITHUB_REPO` environment variable in the plugin config:

```yaml
fields:
  - name: summary
    plugin:
      command: "./examples/view-plugins/github-links.sh"
      env:
        GITHUB_REPO: "DeepReef11/gosynctasks"
```

### date-relative.py
Displays dates as relative to now (e.g., "2 days ago", "in 3 days").

**Requirements:** Python 3

**Usage:**
```yaml
fields:
  - name: due_date
    plugin:
      command: "./examples/view-plugins/date-relative.py"
```

### tags-badges.sh
Formats tags as badges: `[work] [urgent] [review]`

**Usage:**
```yaml
fields:
  - name: tags
    plugin:
      command: "./examples/view-plugins/tags-badges.sh"
```

## Plugin Input Format

Plugins receive JSON on stdin with the following structure:

```json
{
  "uid": "task-123",
  "summary": "Fix bug #42",
  "description": "Detailed description...",
  "status": "TODO",
  "priority": 1,
  "categories": ["work", "urgent"],
  "due_date": "2025-01-15T00:00:00Z",
  "start_date": "2025-01-10T00:00:00Z",
  "created": "2025-01-08T12:00:00Z",
  "modified": "2025-01-09T14:30:00Z",
  "completed": null,
  "parent_uid": "",
  "format": "",
  "width": 0,
  "color": false
}
```

**Note:** Date fields use ISO 8601 format (RFC3339). Fields that are empty or zero-value may be omitted.

## Creating Your Own Plugins

### Requirements

1. **Executable:** Script must have execute permissions (`chmod +x script.sh`)
2. **Input:** Read JSON from stdin
3. **Output:** Write formatted string to stdout
4. **Error Handling:** Write errors to stderr and exit with non-zero code
5. **Timeout:** Complete within configured timeout (default: 1s, max: 5s)

### Minimal Example (Bash)

```bash
#!/bin/sh
# Read JSON from stdin
read -r input

# Extract field using jq
value=$(echo "$input" | jq -r '.summary')

# Output formatted result
echo "TASK: $value"
```

### Minimal Example (Python)

```python
#!/usr/bin/env python3
import sys
import json

# Read JSON from stdin
data = json.load(sys.stdin)

# Format output
print(f"TASK: {data['summary']}")
```

### Minimal Example (Ruby)

```ruby
#!/usr/bin/env ruby
require 'json'

# Read JSON from stdin
data = JSON.parse(STDIN.read)

# Format output
puts "TASK: #{data['summary']}"
```

## Security Considerations

- **Timeout:** Plugins are automatically killed after the configured timeout
- **Sandboxing:** Plugins run as the current user with no special privileges
- **Input Validation:** Plugin output is sanitized (trailing whitespace trimmed)
- **Error Handling:** Plugin errors are caught and displayed as `[plugin error: ...]`

## Tips

- Test your plugins independently before using them in views
- Use `jq` for JSON parsing in shell scripts (it's widely available)
- Keep plugins fast - they run for every task displayed
- Log errors to stderr for debugging
- Use absolute paths or ensure plugins are in PATH
- Consider caching expensive operations (network calls, etc.)

## Troubleshooting

**Plugin not working?**

1. Check execute permissions: `ls -l plugin.sh`
2. Test manually: `echo '{"summary":"test"}' | ./plugin.sh`
3. Check for errors in stderr
4. Verify timeout is sufficient
5. Ensure required tools (jq, python, etc.) are installed

**Timeout errors?**

- Increase timeout in plugin config (max: 5000ms)
- Optimize plugin performance
- Avoid network calls or cache results

**Invalid JSON errors?**

- Ensure jq is installed: `which jq`
- Test JSON parsing: `echo '{"test":"value"}' | jq -r '.test'`
- Check for proper quoting in scripts
