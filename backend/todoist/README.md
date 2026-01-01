# Todoist Backend

This backend integrates gosynctasks with Todoist using the Todoist REST API v2.

## Configuration

The Todoist backend can be configured in `config.yaml` with credentials provided through multiple methods.

### Method 1: Using System Keyring (Recommended)

Store your Todoist API token securely in the system keyring:

```bash
gosynctasks credentials set todoist token --prompt
```

Then configure the backend in `config.yaml`:

```yaml
backends:
  todoist:
    type: todoist
    enabled: true
    username: "token"  # Username hint for keyring lookup
```

### Method 2: Using Environment Variables

Set the Todoist API token via environment variable:

```bash
export TODOIST_API_TOKEN="your-api-token-here"
# OR
export GOSYNCTASKS_TODOIST_PASSWORD="your-api-token-here"
```

Then configure the backend in `config.yaml`:

```yaml
backends:
  todoist:
    type: todoist
    enabled: true
```

### Method 3: Direct Configuration (Legacy, Not Recommended)

**Note:** This method stores your API token in plain text in the config file.

```yaml
backends:
  todoist:
    type: todoist
    enabled: true
    api_token: "your-api-token-here"
```

## Getting Your Todoist API Token

1. Log in to [Todoist](https://todoist.com)
2. Go to Settings → Integrations → Developer
3. Copy your API token from the "API token" section

## Testing

### Unit Tests

Run unit tests with mocked API:

```bash
go test ./backend/todoist -v
```

### Integration Tests

Integration tests require a real Todoist API token and will create/modify actual projects and tasks in your Todoist account.

**⚠️ Warning:** Integration tests will create temporary test projects and tasks. They clean up after themselves, but use a test account if possible.

To run integration tests, set the `TODOIST_API_TOKEN` environment variable:

```bash
export TODOIST_API_TOKEN="your-api-token-here"
go test ./backend/todoist -v -run Integration
```

The integration tests will:
- Create temporary projects with names like `gosynctasks-test-20260101-150405`
- Add, update, and delete test tasks
- Clean up all test data after completion

## Features

### Supported Operations

- ✅ List projects (task lists)
- ✅ Get tasks from a project
- ✅ Create tasks
- ✅ Update tasks
- ✅ Mark tasks as complete/incomplete
- ✅ Delete tasks
- ✅ Create projects
- ✅ Rename projects
- ✅ Delete projects
- ✅ Subtasks (via parent_id)
- ✅ Task labels (categories)
- ✅ Task priority mapping
- ✅ Due dates

### Limitations

- ❌ Todoist doesn't have a trash/archive API for projects
- ❌ `RestoreTaskList` is not supported (no trash feature in Todoist)
- ⚠️ Status mapping: Todoist only has TODO/DONE, so PROCESSING and CANCELLED are simulated with labels

## Priority Mapping

Todoist uses priorities 1-4 (1=normal, 4=urgent), while gosynctasks uses 0-9 (1=highest, 9=lowest).

The mapping is:

| Todoist Priority | gosynctasks Priority | Description |
|------------------|---------------------|-------------|
| 4 (Urgent)       | 1-2                 | Highest     |
| 3 (High)         | 3-4                 | High        |
| 2 (Medium)       | 5-6                 | Medium      |
| 1 (Normal)       | 7-9                 | Low/Normal  |
| -                | 0                   | Undefined   |

## API Rate Limits

The Todoist API has a rate limit of approximately 450 requests per 15 minutes. The backend doesn't currently implement rate limiting or exponential backoff, but this could be added in the future if needed.

## Development

### Adding New Features

When adding new features to the Todoist backend:

1. Update the API client in `api.go` if new endpoints are needed
2. Add the corresponding method to `backend.go`
3. Update the mapper functions in `mapper.go` if data transformation is needed
4. Add unit tests with mocked HTTP responses in `backend_comprehensive_test.go`
5. Optionally add integration tests in the same file

### Running Tests During Development

```bash
# Run all tests
go test ./backend/todoist -v

# Run only unit tests (skips integration tests)
go test ./backend/todoist -v -short

# Run with coverage
go test ./backend/todoist -v -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific test
go test ./backend/todoist -v -run TestTodoistBackend_GetTasks
```

## Security Notes

- **Never commit API tokens to version control**
- Use the keyring method for production use
- Environment variables are suitable for CI/CD and temporary testing
- The `api_token` config field is provided for backward compatibility but should be avoided

## Troubleshooting

### "failed to validate Todoist API token"

- Check that your API token is correct
- Ensure you have internet connectivity
- Verify the Todoist API is accessible (https://api.todoist.com/rest/v2)

### "jq not found" error in tests

Some tests require `jq` for JSON parsing. If you see this error:

- The tests will be skipped automatically if `jq` is not available
- To run all tests, install jq: `apt-get install jq` (Ubuntu/Debian) or `brew install jq` (macOS)
- This only affects formatter plugin tests, not core Todoist tests

## References

- [Todoist REST API Documentation](https://developer.todoist.com/rest/v2/)
- [Todoist API Authentication](https://developer.todoist.com/rest/v2/#authorization)
